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
        self.semaphore = asyncio.Semaphore(max(1, int(max_concurrency)))
        self.on_task_update = on_task_update
        self.tasks: dict[str, GenerationTask] = {}
        self._running: dict[str, asyncio.Task[None]] = {}

    def submit(self, params: GenerationParams) -> GenerationTask:
        task = GenerationTask(params=replace(params))
        self.tasks[task.id] = task
        self._notify(task)
        self._running[task.id] = asyncio.create_task(self._execute(task))
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
            params = replace(base_params, prompt=stripped)
            tasks.append(self.submit(params))
        return tasks

    def cancel_task(self, task_id: str) -> None:
        task = self.tasks.get(task_id)
        if task is None:
            return
        if task.status in {TaskStatus.COMPLETED, TaskStatus.FAILED, TaskStatus.CANCELLED}:
            return
        handle = self._running.get(task_id)
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
        handle = self._running.get(task_id)
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

    async def _execute(self, task: GenerationTask) -> None:
        try:
            async with self.semaphore:
                if task.status is TaskStatus.CANCELLED:
                    return
                task.set_status(TaskStatus.CONNECTING)
                self._notify(task)
                payloads = await self.client.generate(task.params, event_callback=task.add_event)
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
            self._running.pop(task.id, None)

    def _notify(self, task: GenerationTask) -> None:
        if self.on_task_update is None:
            return
        try:
            self.on_task_update(task)
        except Exception:
            return
