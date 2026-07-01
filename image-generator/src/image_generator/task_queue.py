from __future__ import annotations

import asyncio
from collections.abc import Callable
from dataclasses import replace
from typing import Protocol

from image_generator.models import (
    GeneratedImage,
    GenerationParams,
    GenerationTask,
    ImagePayload,
    TaskStatus,
)


class ImageClientProtocol(Protocol):
    async def generate(
        self,
        params: GenerationParams,
        event_callback: Callable[[str], None] | None = None,
    ) -> list[ImagePayload]:
        raise NotImplementedError


class ImageStorageProtocol(Protocol):
    async def save_payloads(
        self,
        payloads: list[ImagePayload],
        params: GenerationParams,
        task_id: str,
    ) -> list[GeneratedImage]:
        raise NotImplementedError


TaskUpdateCallback = Callable[[GenerationTask], None]


class GenerationQueue:
    def __init__(
        self,
        client: ImageClientProtocol,
        storage: ImageStorageProtocol,
        max_concurrency: int,
        on_task_update: TaskUpdateCallback | None = None,
    ) -> None:
        self.client = client
        self.storage = storage
        self.max_concurrency = max(1, int(max_concurrency))
        self.semaphore = asyncio.Semaphore(self.max_concurrency)
        self.on_task_update = on_task_update
        self.tasks: dict[str, GenerationTask] = {}
        self._running: dict[str, asyncio.Task[None]] = {}
        self._task_handles: dict[str, asyncio.Task[None]] = {}
        self._pending: asyncio.Queue[GenerationTask] = asyncio.Queue()

    def submit(self, params: GenerationParams) -> GenerationTask:
        task = GenerationTask(params=replace(params))
        self._track_task(task)
        handle = asyncio.create_task(self._execute(task))
        self._running[task.id] = handle
        self._task_handles[task.id] = handle
        return task

    def submit_batch(
        self,
        prompts: list[str],
        base_params: GenerationParams,
    ) -> list[GenerationTask]:
        tasks: list[GenerationTask] = []
        for prompt in prompts:
            stripped = prompt.strip()
            if not stripped:
                continue
            task = GenerationTask(params=replace(base_params, prompt=stripped))
            self._track_task(task)
            self._pending.put_nowait(task)
            tasks.append(task)
        self._ensure_workers()
        return tasks

    def _track_task(self, task: GenerationTask) -> None:
        self.tasks[task.id] = task
        self._notify(task)

    def _ensure_workers(self) -> None:
        active_workers = sum(1 for handle in self._running.values() if not handle.done())
        worker_count = min(self._pending.qsize(), max(0, self.max_concurrency - active_workers))
        for index in range(worker_count):
            worker_id = f"batch-worker-{id(self)}-{index}-{len(self._running)}"
            self._running[worker_id] = asyncio.create_task(self._run_pending(worker_id))

    def cancel_task(self, task_id: str) -> None:
        task = self.tasks.get(task_id)
        if task is None:
            return
        if task.status in {TaskStatus.COMPLETED, TaskStatus.FAILED, TaskStatus.CANCELLED}:
            return
        handle = self._task_handles.get(task_id) or self._running.get(task_id)
        if handle and not handle.done():
            handle.cancel()
        task.set_status(TaskStatus.CANCELLED)
        self._notify(task)

    def retry_task(self, task_id: str) -> GenerationTask | None:
        task = self.tasks.get(task_id)
        if task is None:
            return None
        return self.submit(replace(task.params))

    def delete_task(self, task_id: str) -> bool:
        task = self.tasks.get(task_id)
        if task is None:
            return False
        handle = self._task_handles.get(task_id) or self._running.get(task_id)
        if handle and not handle.done():
            return False
        if task.status in {TaskStatus.CONNECTING, TaskStatus.GENERATING, TaskStatus.SAVING}:
            return False
        self.tasks.pop(task_id, None)
        self._running.pop(task_id, None)
        return True

    async def wait_idle(self) -> None:
        while True:
            handles = [handle for handle in self._running.values() if not handle.done()]
            if not handles:
                return
            await asyncio.gather(*handles, return_exceptions=True)

    async def _run_pending(self, worker_id: str) -> None:
        try:
            while not self._pending.empty():
                task = self._pending.get_nowait()
                if task.id not in self.tasks or task.status is TaskStatus.CANCELLED:
                    continue
                current = asyncio.current_task()
                if current is not None:
                    self._task_handles[task.id] = current
                await self._execute(task)
        finally:
            self._running.pop(worker_id, None)

    async def _execute(self, task: GenerationTask) -> None:
        try:
            async with self.semaphore:
                if task.status is TaskStatus.CANCELLED:
                    return
                task.set_status(TaskStatus.CONNECTING)
                self._notify(task)
                task.set_status(TaskStatus.GENERATING)
                self._notify(task)
                payloads = await self.client.generate(
                    task.params,
                    event_callback=lambda event: self._add_event(task, event),
                )
                task.set_status(TaskStatus.SAVING)
                self._notify(task)
                task.results = await self.storage.save_payloads(payloads, task.params, task.id)
                task.set_status(TaskStatus.COMPLETED)
                self._notify(task)
        except asyncio.CancelledError:
            task.set_status(TaskStatus.CANCELLED)
            self._notify(task)
        except Exception as exc:
            task.set_error(str(exc))
            self._notify(task)
        finally:
            self._task_handles.pop(task.id, None)
            self._running.pop(task.id, None)
            self._ensure_workers()

    def _add_event(self, task: GenerationTask, event: str) -> None:
        task.add_event(event)
        self._notify(task)

    def _notify(self, task: GenerationTask) -> None:
        if self.on_task_update is None:
            return
        try:
            self.on_task_update(task)
        except Exception:
            return
