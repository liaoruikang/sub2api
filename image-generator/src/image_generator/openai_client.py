from __future__ import annotations

import json
import re
from collections.abc import Callable
from typing import Any

import httpx

from image_generator.config import AppConfig
from image_generator.models import EndpointType, GenerationParams, ImagePayload

IMAGE_URL_RE = re.compile(r"https?://[^\s\"'<>]+", re.I)
DATA_URL_RE = re.compile(r"data:image/(?:png|jpeg|jpg|webp|gif);base64,", re.I)
BASE64_CHARS = frozenset("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=")
ASCII_WHITESPACE = frozenset(" \t\r\n\f\v")
TRAILING_URL_PUNCTUATION = ".,;:!?)]}"


class APIError(RuntimeError):
    pass


EventCallback = Callable[[str], None]


def _join_openai_url(base_url: str, endpoint: str) -> str:
    normalized = base_url.rstrip("/")
    if normalized.endswith("/v1"):
        return f"{normalized}/{endpoint.lstrip('/')}"
    return f"{normalized}/v1/{endpoint.lstrip('/')}"


def _error_message(response: httpx.Response) -> str:
    try:
        payload = response.json()
    except ValueError:
        body = response.text[:300]
        return body or response.reason_phrase
    if isinstance(payload, dict):
        error = payload.get("error")
        if isinstance(error, dict) and error.get("message"):
            return str(error["message"])
        if payload.get("message"):
            return str(payload["message"])
    return response.reason_phrase


def parse_image_payloads(payload: dict[str, Any]) -> list[ImagePayload]:
    data = payload.get("data")
    if not isinstance(data, list):
        return []
    results: list[ImagePayload] = []
    for item in data:
        if not isinstance(item, dict):
            continue
        b64_json = item.get("b64_json")
        url = item.get("url")
        if isinstance(b64_json, str) and b64_json:
            results.append(ImagePayload(kind="b64_json", value=b64_json))
        if isinstance(url, str) and url:
            results.append(ImagePayload(kind="url", value=url))
    return results


def _read_data_url_base64(text: str, start: int) -> str:
    chars: list[str] = []
    index = start
    while index < len(text):
        char = text[index]
        if char in BASE64_CHARS:
            chars.append(char)
            index += 1
            continue
        if char not in ASCII_WHITESPACE:
            break

        whitespace_start = index
        while index < len(text) and text[index] in ASCII_WHITESPACE:
            index += 1
        whitespace = text[whitespace_start:index]
        next_char = text[index] if index < len(text) else ""
        if not chars or next_char not in BASE64_CHARS:
            break
        if len(chars) % 4 != 0 or "\r" in whitespace or "\n" in whitespace:
            continue
        break
    return "".join(chars)


def parse_text_for_images(text: str) -> list[ImagePayload]:
    results: list[ImagePayload] = []
    data_url_ranges: list[range] = []
    for match in DATA_URL_RE.finditer(text):
        b64_json = _read_data_url_base64(text, match.end())
        if b64_json:
            results.append(ImagePayload(kind="b64_json", value=b64_json))
            data_url_ranges.append(range(match.start(), match.end() + len(b64_json)))
    for match in IMAGE_URL_RE.finditer(text):
        if any(match.start() in data_url_range for data_url_range in data_url_ranges):
            continue
        url = match.group(0).rstrip(TRAILING_URL_PUNCTUATION)
        if url:
            results.append(ImagePayload(kind="url", value=url))
    return results


def _content_texts(container: dict[str, Any]) -> list[str]:
    content = container.get("content")
    if isinstance(content, str):
        return [content]
    if not isinstance(content, list):
        return []

    texts: list[str] = []
    for part in content:
        if not isinstance(part, dict):
            continue

        text = part.get("text")
        image_url = part.get("image_url")
        url = part.get("url")
        if isinstance(text, str):
            texts.append(text)
        elif isinstance(image_url, dict) and isinstance(image_url.get("url"), str):
            texts.append(image_url["url"])
        elif isinstance(image_url, str):
            texts.append(image_url)
        elif isinstance(url, str):
            texts.append(url)
    return texts


def _delta_texts(payload: dict[str, Any]) -> list[str]:
    choices = payload.get("choices")
    if not isinstance(choices, list):
        return []

    texts: list[str] = []
    for choice in choices:
        if not isinstance(choice, dict):
            continue
        delta = choice.get("delta")
        if isinstance(delta, dict):
            texts.extend(_content_texts(delta))
    return texts


def parse_chat_payloads(payload: dict[str, Any]) -> list[ImagePayload]:
    results: list[ImagePayload] = []
    choices = payload.get("choices")
    if not isinstance(choices, list):
        return results
    for choice in choices:
        if not isinstance(choice, dict):
            continue
        message = choice.get("message")
        delta = choice.get("delta")
        for container in (message, delta):
            if not isinstance(container, dict):
                continue
            for text in _content_texts(container):
                results.extend(parse_text_for_images(text))
    return results


class OpenAIImageClient:
    def __init__(self, config: AppConfig) -> None:
        self.config = config.normalized()

    @property
    def images_url(self) -> str:
        return _join_openai_url(self.config.base_url, "images/generations")

    @property
    def chat_url(self) -> str:
        return _join_openai_url(self.config.base_url, "chat/completions")

    def build_images_payload(self, params: GenerationParams) -> dict[str, Any]:
        return {
            "model": params.model,
            "prompt": params.prompt,
            "size": params.size,
            "n": params.n,
            "quality": params.quality,
            "style": params.style,
            "response_format": params.response_format,
        }

    def build_chat_payload(self, params: GenerationParams) -> dict[str, Any]:
        return {
            "model": params.model,
            "messages": [{"role": "user", "content": params.prompt}],
            "stream": params.stream,
        }

    async def generate(
        self,
        params: GenerationParams,
        event_callback: EventCallback | None = None,
    ) -> list[ImagePayload]:
        headers = {
            "Authorization": f"Bearer {self.config.api_key}",
            "Content-Type": "application/json",
        }
        timeout = httpx.Timeout(self.config.timeout_seconds)
        async with httpx.AsyncClient(timeout=timeout, headers=headers) as client:
            if params.endpoint_type is EndpointType.IMAGES:
                return await self._generate_images(client, params, event_callback)
            return await self._generate_chat(client, params, event_callback)

    async def _generate_images(
        self,
        client: httpx.AsyncClient,
        params: GenerationParams,
        event_callback: EventCallback | None,
    ) -> list[ImagePayload]:
        if event_callback:
            event_callback("connecting to images endpoint")
        response = await client.post(self.images_url, json=self.build_images_payload(params))
        self._raise_for_status(response)
        if event_callback:
            event_callback("images response received")
        payloads = parse_image_payloads(response.json())
        if not payloads:
            raise APIError("Images response did not contain url or b64_json data")
        return payloads

    async def _generate_chat(
        self,
        client: httpx.AsyncClient,
        params: GenerationParams,
        event_callback: EventCallback | None,
    ) -> list[ImagePayload]:
        if params.stream:
            return await self._generate_chat_stream(client, params, event_callback)
        if event_callback:
            event_callback("connecting to chat completions endpoint")
        response = await client.post(self.chat_url, json=self.build_chat_payload(params))
        self._raise_for_status(response)
        payloads = parse_chat_payloads(response.json())
        if not payloads:
            raise APIError("Chat response did not contain an image URL or base64 image data")
        return payloads

    async def _generate_chat_stream(
        self,
        client: httpx.AsyncClient,
        params: GenerationParams,
        event_callback: EventCallback | None,
    ) -> list[ImagePayload]:
        chunks: list[str] = []
        if event_callback:
            event_callback("connecting to streaming chat completions endpoint")
        async with client.stream(
            "POST",
            self.chat_url,
            json=self.build_chat_payload(params),
        ) as response:
            if response.status_code < 200 or response.status_code > 299:
                await response.aread()
            self._raise_for_status(response)
            async for line in response.aiter_lines():
                if not line.startswith("data:"):
                    continue
                data = line.removeprefix("data:").strip()
                if data == "[DONE]":
                    break
                if event_callback:
                    event_callback("stream event received")
                try:
                    payload = json.loads(data)
                except json.JSONDecodeError:
                    chunks.append(data)
                    continue
                chunks.extend(_delta_texts(payload))
        parsed_from_text = parse_text_for_images("".join(chunks))
        if parsed_from_text:
            return parsed_from_text
        raise APIError("Streaming response ended without an image URL or base64 image data")

    def _raise_for_status(self, response: httpx.Response) -> None:
        if 200 <= response.status_code <= 299:
            return
        message = _error_message(response)
        raise APIError(f"Upstream API returned HTTP {response.status_code}: {message}")
