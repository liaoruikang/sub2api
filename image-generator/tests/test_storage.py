import base64
from pathlib import Path

import httpx
import pytest

from image_generator.models import GenerationParams, ImagePayload
from image_generator.storage import ImageStorage, detect_extension, safe_name_part

PNG_BYTES = b"\x89PNG\r\n\x1a\n" + b"\x00" * 16
PNG_B64 = base64.b64encode(PNG_BYTES).decode("ascii")


def test_safe_name_part_removes_unsafe_characters() -> None:
    assert safe_name_part("gpt/image:1?") == "gpt-image-1"
    assert safe_name_part("   ") == "value"


def test_detect_extension_from_bytes_and_content_type() -> None:
    assert detect_extension(PNG_BYTES, None) == ".png"
    assert detect_extension(b"abc", "image/jpeg") == ".jpg"
    assert detect_extension(b"abc", "image/webp") == ".webp"


@pytest.mark.asyncio
async def test_save_base64_payload_writes_file(tmp_path: Path) -> None:
    storage = ImageStorage(tmp_path)
    params = GenerationParams(prompt="fox", model="gpt-image-1")

    results = await storage.save_payloads(
        [ImagePayload(kind="b64_json", value=PNG_B64)],
        params=params,
        task_id="abc123",
    )

    assert len(results) == 1
    assert results[0].path.exists()
    assert results[0].path.suffix == ".png"
    assert results[0].source_type == "b64_json"
    assert results[0].b64_json == PNG_B64


@pytest.mark.asyncio
async def test_save_url_payload_downloads_file(tmp_path: Path) -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        assert str(request.url) == "https://example.com/image.png"
        return httpx.Response(200, content=PNG_BYTES, headers={"content-type": "image/png"})

    async with httpx.AsyncClient(transport=httpx.MockTransport(handler)) as http_client:
        storage = ImageStorage(tmp_path, http_client=http_client)
        results = await storage.save_payloads(
            [ImagePayload(kind="url", value="https://example.com/image.png")],
            params=GenerationParams(prompt="fox"),
            task_id="abc123",
        )

    assert results[0].path.exists()
    assert results[0].source_type == "url"
    assert results[0].source_ref == "https://example.com/image.png"
