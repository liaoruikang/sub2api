from __future__ import annotations

from dataclasses import dataclass, field
from enum import StrEnum
from pathlib import Path
from typing import Literal
from uuid import uuid4


class EndpointType(StrEnum):
    IMAGES = "images"
    CHAT_COMPLETIONS = "chat_completions"


class TaskStatus(StrEnum):
    QUEUED = "queued"
    CONNECTING = "connecting"
    GENERATING = "generating"
    SAVING = "saving"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"


ImagePayloadKind = Literal["url", "b64_json"]
ImageSourceType = Literal["url", "b64_json", "chat_text"]


@dataclass(slots=True)
class GenerationParams:
    prompt: str
    endpoint_type: EndpointType = EndpointType.IMAGES
    model: str = "gpt-image-1"
    size: str = "1024x1024"
    n: int = 1
    quality: str = "standard"
    style: str = "natural"
    response_format: str = "b64_json"
    stream: bool = False


@dataclass(slots=True)
class ImagePayload:
    kind: ImagePayloadKind
    value: str


@dataclass(slots=True)
class GeneratedImage:
    path: Path
    source_type: ImageSourceType
    source_ref: str
    b64_json: str | None = None


@dataclass(slots=True)
class GenerationTask:
    params: GenerationParams
    id: str = field(default_factory=lambda: uuid4().hex[:12])
    status: TaskStatus = TaskStatus.QUEUED
    events: list[str] = field(default_factory=list)
    error: str | None = None
    results: list[GeneratedImage] = field(default_factory=list)

    def set_status(self, status: TaskStatus) -> None:
        self.status = status
        if status is not TaskStatus.FAILED:
            self.error = None

    def add_event(self, event: str) -> None:
        if event:
            self.events.append(event)

    def set_error(self, message: str) -> None:
        self.status = TaskStatus.FAILED
        self.error = message
