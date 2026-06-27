from __future__ import annotations

import base64
import re
from datetime import datetime
from pathlib import Path

import httpx

from image_generator.models import GeneratedImage, GenerationParams, ImagePayload

SAFE_NAME_PATTERN = re.compile(r"[^A-Za-z0-9._-]+")


def safe_name_part(value: str) -> str:
    cleaned = SAFE_NAME_PATTERN.sub("-", value.strip()).strip("-._")
    return cleaned[:64] or "value"


def detect_extension(data: bytes, content_type: str | None) -> str:
    lower_content_type = (content_type or "").lower()
    if data.startswith(b"\x89PNG\r\n\x1a\n") or "image/png" in lower_content_type:
        return ".png"
    if data.startswith(b"\xff\xd8\xff") or "image/jpeg" in lower_content_type:
        return ".jpg"
    if data.startswith(b"GIF8") or "image/gif" in lower_content_type:
        return ".gif"
    if data.startswith(b"RIFF") and b"WEBP" in data[:16] or "image/webp" in lower_content_type:
        return ".webp"
    return ".png"


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
        results: list[GeneratedImage] = []
        for index, payload in enumerate(payloads, start=1):
            if payload.kind == "b64_json":
                results.append(await self._save_b64(payload.value, params, task_id, index))
            elif payload.kind == "url":
                results.append(await self._save_url(payload.value, params, task_id, index))
            else:
                raise ValueError(f"Unsupported image payload kind: {payload.kind}")
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
        extension = detect_extension(data, None)
        path = self._make_path(params, task_id, index, extension)
        path.write_bytes(data)
        return GeneratedImage(
            path=path,
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
        if self.http_client is not None:
            response = await self.http_client.get(url)
        else:
            async with httpx.AsyncClient(timeout=60) as client:
                response = await client.get(url)
        response.raise_for_status()
        data = response.content
        extension = detect_extension(data, response.headers.get("content-type"))
        path = self._make_path(params, task_id, index, extension)
        path.write_bytes(data)
        return GeneratedImage(path=path, source_type="url", source_ref=url)

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
        filename = f"{timestamp}_{model}_{task}_{index}{extension}"
        return self.save_dir / filename
