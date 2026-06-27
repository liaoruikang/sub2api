from __future__ import annotations

import json
import os
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from platformdirs import user_config_dir

from image_generator.models import EndpointType

CONFIG_DIR_ENV = "IMAGE_GENERATOR_CONFIG_DIR"
APP_NAME = "ImageGenerator"


def default_save_dir() -> Path:
    return Path.home() / "Pictures" / APP_NAME


def config_dir() -> Path:
    override = os.environ.get(CONFIG_DIR_ENV)
    if override:
        return Path(override).expanduser()
    return Path(user_config_dir(APP_NAME, appauthor=False))


def config_path() -> Path:
    return config_dir() / "config.json"


@dataclass(slots=True)
class AppConfig:
    base_url: str = ""
    api_key: str = ""
    timeout_seconds: float = 120
    max_concurrency: int = 3
    default_save_dir: Path = default_save_dir()
    default_endpoint_type: EndpointType = EndpointType.IMAGES
    default_model: str = "gpt-image-1"

    def normalized(self) -> "AppConfig":
        return AppConfig(
            base_url=self.base_url.rstrip("/"),
            api_key=self.api_key.strip(),
            timeout_seconds=max(1, float(self.timeout_seconds)),
            max_concurrency=max(1, int(self.max_concurrency)),
            default_save_dir=Path(self.default_save_dir).expanduser(),
            default_endpoint_type=self.default_endpoint_type,
            default_model=self.default_model.strip() or "gpt-image-1",
        )

    def to_dict(self) -> dict[str, Any]:
        normalized = self.normalized()
        return {
            "base_url": normalized.base_url,
            "api_key": normalized.api_key,
            "timeout_seconds": normalized.timeout_seconds,
            "max_concurrency": normalized.max_concurrency,
            "default_save_dir": str(normalized.default_save_dir),
            "default_endpoint_type": normalized.default_endpoint_type.value,
            "default_model": normalized.default_model,
        }

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "AppConfig":
        endpoint_value = data.get("default_endpoint_type", EndpointType.IMAGES.value)
        return cls(
            base_url=str(data.get("base_url", "")),
            api_key=str(data.get("api_key", "")),
            timeout_seconds=float(data.get("timeout_seconds", 120)),
            max_concurrency=int(data.get("max_concurrency", 3)),
            default_save_dir=Path(str(data.get("default_save_dir", default_save_dir()))),
            default_endpoint_type=EndpointType(endpoint_value),
            default_model=str(data.get("default_model", "gpt-image-1")),
        ).normalized()


def load_config(path: Path | None = None) -> AppConfig:
    target = path or config_path()
    if not target.exists():
        return AppConfig().normalized()
    data = json.loads(target.read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        return AppConfig().normalized()
    return AppConfig.from_dict(data)


def save_config(config: AppConfig, path: Path | None = None) -> None:
    target = path or config_path()
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(
        json.dumps(config.to_dict(), ensure_ascii=False, indent=2),
        encoding="utf-8",
    )


def validate_config(config: AppConfig) -> list[str]:
    messages: list[str] = []
    normalized = config.normalized()
    if not normalized.base_url:
        messages.append("Base URL is required")
    if not normalized.api_key:
        messages.append("API key is required")
    if normalized.timeout_seconds < 1:
        messages.append("Timeout must be at least 1 second")
    if normalized.max_concurrency < 1:
        messages.append("Concurrency must be at least 1")
    return messages


def redacted_api_key(api_key: str) -> str:
    stripped = api_key.strip()
    if not stripped:
        return "Not configured"
    return f"Configured ending with {stripped[-4:]}"
