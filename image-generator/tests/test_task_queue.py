import asyncio
from pathlib import Path

import pytest

from image_generator.models import GeneratedImage, GenerationParams, ImagePayload, TaskStatus
from image_generator.task_queue import GenerationQueue


class FakeClient:
    def __init__(self, delay: float = 0.01, fail: bool = False) -> None:
        self.delay = delay
        self.fail = fail
        self.active = 0
        self.max_active = 0

    async def generate(self, params: GenerationParams, event_callback=None) -> list[ImagePayload]:
        self.active += 1
        self.max_active = max(self.max_active, self.active)
        if event_callback:
            event_callback("fake event")
        try:
            await asyncio.sleep(self.delay)
            if self.fail:
                raise RuntimeError("fake upstream failure")
            return [ImagePayload(kind="b64_json", value="YWJj")]
        finally:
            self.active -= 1


class FakeStorage:
    async def save_payloads(
        self,
        payloads: list[ImagePayload],
        params: GenerationParams,
        task_id: str,
    ) -> list[GeneratedImage]:
        return [GeneratedImage(path=Path(f"C:/out/{task_id}.png"), source_type="b64_json", source_ref="b64:4")]


@pytest.mark.asyncio
async def test_queue_completes_task_and_records_result() -> None:
    updates = []
    queue = GenerationQueue(FakeClient(), FakeStorage(), max_concurrency=2, on_task_update=updates.append)

    task = queue.submit(GenerationParams(prompt="fox"))
    await queue.wait_idle()

    assert task.status is TaskStatus.COMPLETED
    assert task.results[0].path.name == f"{task.id}.png"
    assert "fake event" in task.events
    assert updates[-1].status is TaskStatus.COMPLETED


@pytest.mark.asyncio
async def test_queue_limits_concurrency() -> None:
    client = FakeClient(delay=0.03)
    queue = GenerationQueue(client, FakeStorage(), max_concurrency=2)

    for index in range(5):
        queue.submit(GenerationParams(prompt=f"fox {index}"))
    await queue.wait_idle()

    assert client.max_active == 2


@pytest.mark.asyncio
async def test_queue_marks_failures_without_stopping_other_tasks() -> None:
    class PartlyFailingClient(FakeClient):
        async def generate(self, params: GenerationParams, event_callback=None) -> list[ImagePayload]:
            if params.prompt == "fail":
                raise RuntimeError("bad prompt")
            return await super().generate(params, event_callback)

    queue = GenerationQueue(PartlyFailingClient(), FakeStorage(), max_concurrency=2)
    failed = queue.submit(GenerationParams(prompt="fail"))
    ok = queue.submit(GenerationParams(prompt="ok"))

    await queue.wait_idle()

    assert failed.status is TaskStatus.FAILED
    assert failed.error == "bad prompt"
    assert ok.status is TaskStatus.COMPLETED


@pytest.mark.asyncio
async def test_cancel_running_task_marks_cancelled() -> None:
    queue = GenerationQueue(FakeClient(delay=0.2), FakeStorage(), max_concurrency=1)
    task = queue.submit(GenerationParams(prompt="slow"))

    await asyncio.sleep(0.02)
    queue.cancel_task(task.id)
    await queue.wait_idle()

    assert task.status is TaskStatus.CANCELLED


@pytest.mark.asyncio
async def test_retry_creates_new_task_with_same_params() -> None:
    queue = GenerationQueue(FakeClient(), FakeStorage(), max_concurrency=1)
    original = queue.submit(GenerationParams(prompt="fox"))
    retry = queue.retry_task(original.id)

    assert retry is not None
    assert retry.id != original.id
    assert retry.params.prompt == "fox"
    await queue.wait_idle()


@pytest.mark.asyncio
async def test_delete_task_rejects_submitted_task_before_it_runs() -> None:
    queue = GenerationQueue(FakeClient(delay=0.01), FakeStorage(), max_concurrency=1)
    task = queue.submit(GenerationParams(prompt="queued"))

    removed = queue.delete_task(task.id)
    await asyncio.sleep(0.03)

    assert removed is False
    assert task.id in queue.tasks
    await queue.wait_idle()


@pytest.mark.asyncio
async def test_delete_task_removes_completed_task() -> None:
    queue = GenerationQueue(FakeClient(), FakeStorage(), max_concurrency=1)
    task = queue.submit(GenerationParams(prompt="fox"))
    await queue.wait_idle()

    removed = queue.delete_task(task.id)

    assert removed is True
    assert task.id not in queue.tasks
