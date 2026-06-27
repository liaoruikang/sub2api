from pathlib import Path

from image_generator.models import (
    EndpointType,
    GeneratedImage,
    GenerationParams,
    GenerationTask,
    ImagePayload,
    TaskStatus,
)


def test_generation_params_defaults_are_images_api_defaults() -> None:
    params = GenerationParams(prompt="a watercolor fox")

    assert params.endpoint_type is EndpointType.IMAGES
    assert params.model == "gpt-image-1"
    assert params.prompt == "a watercolor fox"
    assert params.size == "1024x1024"
    assert params.n == 1
    assert params.quality == "standard"
    assert params.style == "natural"
    assert params.response_format == "b64_json"
    assert params.stream is False


def test_generation_task_tracks_status_events_errors_and_results() -> None:
    task = GenerationTask(params=GenerationParams(prompt="mountain"))

    task.set_status(TaskStatus.CONNECTING)
    task.add_event("connected")
    task.set_error("network timeout")
    task.results.append(
        GeneratedImage(path=Path("C:/out/image.png"), source_type="b64_json", source_ref="b64:abcdef")
    )

    assert len(task.id) == 12
    assert task.status is TaskStatus.FAILED
    assert task.events == ["connected"]
    assert task.error == "network timeout"
    assert task.results[0].path.name == "image.png"


def test_image_payload_records_url_or_base64() -> None:
    url_payload = ImagePayload(kind="url", value="https://example.com/image.png")
    b64_payload = ImagePayload(kind="b64_json", value="YWJj")

    assert url_payload.kind == "url"
    assert b64_payload.kind == "b64_json"
