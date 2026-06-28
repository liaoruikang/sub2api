from __future__ import annotations

import base64
import re
from datetime import datetime
from pathlib import Path

import httpx

from image_generator.models import GeneratedImage, GenerationParams, ImagePayload

SAFE_NAME_PATTERN = re.compile(r"[^A-Za-z0-9._-]+")
SIGNATURE_EXTENSIONS: tuple[tuple[bytes, str], ...] = (
    (b"\x89PNG\r\n\x1a\n", ".png"),
    (b"\xff\xd8\xff", ".jpg"),
    (b"GIF8", ".gif"),
)
CONTENT_TYPE_EXTENSIONS = {
    "image/png": ".png",
    "image/jpeg": ".jpg",
    "image/gif": ".gif",
    "image/webp": ".webp",
}
UNSUPPORTED_IMAGE_ERROR = "Image data is empty or not a supported image"


def safe_name_part(value: str) -> str:
    cleaned = SAFE_NAME_PATTERN.sub("-", value.strip()).strip("-._")
    return cleaned[:64] or "value"


def detect_extension(data: bytes, content_type: str | None) -> str:
    for signature, extension in SIGNATURE_EXTENSIONS:
        if data.startswith(signature):
            return extension
    if data.startswith(b"RIFF") and b"WEBP" in data[:16]:
        return ".webp"

    lower_content_type = (content_type or "").lower()
    for expected_content_type, extension in CONTENT_TYPE_EXTENSIONS.items():
        if expected_content_type in lower_content_type:
            return extension
    return ".png"


def is_supported_image(data: bytes) -> bool:
    if not data:
        return False
    if any(data.startswith(signature) for signature, _ in SIGNATURE_EXTENSIONS):
        return True
    return data.startswith(b"RIFF") and b"WEBP" in data[:16]


class ImageStorage:
    def __init__(self, save_dir: Path, http_client: httpx.AsyncClient | None = None) -> None:
        self.save_dir = Path(save_dir).expanduser()
        self.http_client = http_client

    async def save_payloads(
        self,
        payloads: list[ImagePayload],
        params: GenerationParams,
        task_id: str,
    ) -> list[GeneratedImage]:
        self.save_dir.mkdir(parents=True, exist_ok=True)
        handlers = {
            "b64_json": self._save_b64,
            "url": self._save_url,
        }
        results: list[GeneratedImage] = []
        try:
            for index, payload in enumerate(payloads, start=1):
                handler = handlers.get(payload.kind)
                if handler is None:
                    raise ValueError(f"Unsupported image payload kind: {payload.kind}")
                results.append(await handler(payload.value, params, task_id, index))
        except Exception:
            for result in results:
                result.path.unlink(missing_ok=True)
            raise
        return results

    async def _save_b64(
        self,
        b64_json: str,
        params: GenerationParams,
        task_id: str,
        index: int,
    ) -> GeneratedImage:
        try:
            data = base64.b64decode(b64_json, validate=True)
        except Exception as exc:
            raise ValueError("Image base64 data could not be decoded") from exc
        return self._write_image(
            data=data,
            params=params,
            task_id=task_id,
            index=index,
            source_type="b64_json",
            source_ref=f"b64:{len(b64_json)}",
            b64_json=b64_json,
        )

    async def _save_url(
        self,
        url: str,
        params: GenerationParams,
        task_id: str,
        index: int,
    ) -> GeneratedImage:
        response = await self._get_url_response(url)
        response.raise_for_status()
        return self._write_image(
            data=response.content,
            params=params,
            task_id=task_id,
            index=index,
            source_type="url",
            source_ref=url,
            content_type=response.headers.get("content-type"),
        )

    async def _get_url_response(self, url: str) -> httpx.Response:
        if self.http_client is not None:
            return await self.http_client.get(url, follow_redirects=True)
        async with httpx.AsyncClient(timeout=60, follow_redirects=True) as client:
            return await client.get(url)

    def _write_image(
        self,
        data: bytes,
        params: GenerationParams,
        task_id: str,
        index: int,
        source_type: str,
        source_ref: str,
        *,
        b64_json: str | None = None,
        content_type: str | None = None,
    ) -> GeneratedImage:
        if not is_supported_image(data):
            raise ValueError(UNSUPPORTED_IMAGE_ERROR)
        extension = detect_extension(data, content_type)
        path = self._make_path(params, task_id, index, extension)
        try:
            path.write_bytes(data)
        except Exception:
            path.unlink(missing_ok=True)
            raise
        return GeneratedImage(
            path=path,
            source_type=source_type,
            source_ref=source_ref,
            b64_json=b64_json,
        )

    def _make_path(
        self,
        params: GenerationParams,
        task_id: str,
        index: int,
        extension: str,
    ) -> Path:
        timestamp = datetime.now().strftime("%Y%m%d-%H%M%S")
        model = safe_name_part(params.model)
        task = safe_name_part(task_id)
        filename_stem = f"{timestamp}_{model}_{task}_{index}"
        path = self.save_dir / f"{filename_stem}{extension}"
        suffix = 2
        while path.exists():
            path = self.save_dir / f"{filename_stem}_{suffix}{extension}"
            suffix += 1
        return path
