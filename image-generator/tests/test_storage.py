import base64
from datetime import datetime
from pathlib import Path

import httpx
import pytest

import image_generator.storage as storage_module
from image_generator.models import GenerationParams, ImagePayload
from image_generator.storage import ImageStorage, detect_extension, safe_name_part

PNG_BYTES = b"\x89PNG\r\n\x1a\n" + b"\x00" * 16
PNG_B64 = base64.b64encode(PNG_BYTES).decode("ascii")
NON_IMAGE_B64 = base64.b64encode(b"not an image").decode("ascii")
TASK_ID = "abc123"
PROMPT = "fox"


def test_safe_name_part_removes_unsafe_characters() -> None:
    assert safe_name_part("gpt/image:1?") == "gpt-image-1"
    assert safe_name_part("   ") == "value"


@pytest.mark.parametrize(
    ("data", "content_type", "expected_extension"),
    [
        (PNG_BYTES, None, ".png"),
        (b"abc", "image/jpeg", ".jpg"),
        (b"abc", "image/webp", ".webp"),
    ],
)
def test_detect_extension_from_bytes_and_content_type(
    data: bytes,
    content_type: str | None,
    expected_extension: str,
) -> None:
    assert detect_extension(data, content_type) == expected_extension


@pytest.mark.asyncio
async def test_save_base64_payload_writes_file(tmp_path: Path) -> None:
    storage = ImageStorage(tmp_path)
    params = GenerationParams(prompt=PROMPT, model="gpt-image-1")

    results = await storage.save_payloads(
        [ImagePayload(kind="b64_json", value=PNG_B64)],
        params=params,
        task_id=TASK_ID,
    )

    assert len(results) == 1
    assert results[0].path.exists()
    assert results[0].path.suffix == ".png"
    assert results[0].source_type == "b64_json"
    assert results[0].b64_json == PNG_B64


@pytest.mark.asyncio
async def test_save_url_payload_downloads_file(tmp_path: Path) -> None:
    image_url = "https://example.com/image.png"

    def handler(request: httpx.Request) -> httpx.Response:
        assert str(request.url) == image_url
        return httpx.Response(200, content=PNG_BYTES, headers={"content-type": "image/png"})

    async with httpx.AsyncClient(transport=httpx.MockTransport(handler)) as http_client:
        storage = ImageStorage(tmp_path, http_client=http_client)
        results = await storage.save_payloads(
            [ImagePayload(kind="url", value=image_url)],
            params=GenerationParams(prompt=PROMPT),
            task_id=TASK_ID,
        )

    assert results[0].path.exists()
    assert results[0].source_type == "url"
    assert results[0].source_ref == "https://example.com/image.png"


@pytest.mark.parametrize("b64_json", ["", NON_IMAGE_B64])
@pytest.mark.asyncio
async def test_save_base64_rejects_empty_or_non_image_bytes(tmp_path: Path, b64_json: str) -> None:
    storage = ImageStorage(tmp_path)

    with pytest.raises(ValueError, match="Image data is empty or not a supported image"):
        await storage.save_payloads(
            [ImagePayload(kind="b64_json", value=b64_json)],
            params=GenerationParams(prompt=PROMPT),
            task_id=TASK_ID,
        )

    assert list(tmp_path.iterdir()) == []


@pytest.mark.asyncio
async def test_save_url_payload_follows_redirect_and_saves_final_image(tmp_path: Path) -> None:
    redirect_url = "https://example.com/image.png"
    final_url = "https://cdn.example.com/signed-image.png"
    requested_urls: list[str] = []

    def handler(request: httpx.Request) -> httpx.Response:
        requested_urls.append(str(request.url))
        if str(request.url) == redirect_url:
            return httpx.Response(302, headers={"location": final_url})
        assert str(request.url) == final_url
        return httpx.Response(200, content=PNG_BYTES, headers={"content-type": "image/png"})

    async with httpx.AsyncClient(transport=httpx.MockTransport(handler)) as http_client:
        storage = ImageStorage(tmp_path, http_client=http_client)
        results = await storage.save_payloads(
            [ImagePayload(kind="url", value=redirect_url)],
            params=GenerationParams(prompt=PROMPT),
            task_id=TASK_ID,
        )

    assert requested_urls == [redirect_url, final_url]
    assert results[0].path.exists()
    assert results[0].path.read_bytes() == PNG_BYTES


@pytest.mark.asyncio
async def test_repeated_saves_with_same_name_do_not_overwrite(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    class FixedDateTime:
        @classmethod
        def now(cls) -> datetime:
            return datetime(2026, 6, 28, 12, 0, 0)

    monkeypatch.setattr(storage_module, "datetime", FixedDateTime)
    storage = ImageStorage(tmp_path)
    params = GenerationParams(prompt=PROMPT, model="gpt-image-1")

    first = await storage.save_payloads(
        [ImagePayload(kind="b64_json", value=PNG_B64)],
        params=params,
        task_id=TASK_ID,
    )
    second = await storage.save_payloads(
        [ImagePayload(kind="b64_json", value=PNG_B64)],
        params=params,
        task_id=TASK_ID,
    )

    assert first[0].path.name == "20260628-120000_gpt-image-1_abc123_1.png"
    assert second[0].path.name == "20260628-120000_gpt-image-1_abc123_1_2.png"
    assert first[0].path.exists()
    assert second[0].path.exists()
    assert len(list(tmp_path.iterdir())) == 2


@pytest.mark.asyncio
async def test_partial_batch_failure_cleans_up_first_written_file(tmp_path: Path) -> None:
    storage = ImageStorage(tmp_path)

    with pytest.raises(ValueError, match="Image data is empty or not a supported image"):
        await storage.save_payloads(
            [
                ImagePayload(kind="b64_json", value=PNG_B64),
                ImagePayload(kind="b64_json", value=NON_IMAGE_B64),
            ],
            params=GenerationParams(prompt=PROMPT),
            task_id=TASK_ID,
        )

    assert list(tmp_path.iterdir()) == []
