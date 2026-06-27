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


def _string_value(value: Any, default: str = "") -> str:
    if value is None:
        return default
    if isinstance(value, str):
        return value
    return default


def _path_value(value: Any, default: Path) -> Path:
    if value is None:
        return default
    if isinstance(value, os.PathLike):
        return Path(value).expanduser()
    if isinstance(value, str):
        stripped = value.strip()
        if stripped:
            return Path(stripped).expanduser()
    return default


def _float_value(value: Any, default: float) -> float:
    try:
        return float(value)
    except (TypeError, ValueError):
        return default


def _positive_float_value(value: Any, default: float) -> float:
    converted = _float_value(value, default)
    if converted < 1:
        return default
    return converted


def _int_value(value: Any, default: int) -> int:
    try:
        return int(value)
    except (TypeError, ValueError):
        return default


def _positive_int_value(value: Any, default: int) -> int:
    converted = _int_value(value, default)
    if converted < 1:
        return default
    return converted


def _endpoint_type(value: Any) -> EndpointType:
    try:
        return EndpointType(value)
    except (TypeError, ValueError):
        return EndpointType.IMAGES


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
        normalized_base_url = _string_value(self.base_url).strip().rstrip("/")
        return AppConfig(
            base_url=normalized_base_url,
            api_key=_string_value(self.api_key).strip(),
            timeout_seconds=max(1, _float_value(self.timeout_seconds, 120)),
            max_concurrency=max(1, _int_value(self.max_concurrency, 3)),
            default_save_dir=_path_value(self.default_save_dir, default_save_dir()),
            default_endpoint_type=_endpoint_type(self.default_endpoint_type),
            default_model=_string_value(self.default_model, "gpt-image-1").strip() or "gpt-image-1",
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
        return cls(
            base_url=_string_value(data.get("base_url")),
            api_key=_string_value(data.get("api_key")),
            timeout_seconds=_positive_float_value(data.get("timeout_seconds", 120), 120),
            max_concurrency=_positive_int_value(data.get("max_concurrency", 3), 3),
            default_save_dir=_path_value(data.get("default_save_dir"), default_save_dir()),
            default_endpoint_type=_endpoint_type(
                data.get("default_endpoint_type", EndpointType.IMAGES.value)
            ),
            default_model=_string_value(data.get("default_model"), "gpt-image-1"),
        ).normalized()


def load_config(path: Path | None = None) -> AppConfig:
    target = path or config_path()
    if not target.exists():
        return AppConfig().normalized()
    try:
        data = json.loads(target.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError, UnicodeDecodeError):
        return AppConfig().normalized()
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
    if _float_value(config.timeout_seconds, 0) < 1:
        messages.append("Timeout must be at least 1 second")
    if _int_value(config.max_concurrency, 0) < 1:
        messages.append("Concurrency must be at least 1")
    return messages


def redacted_api_key(api_key: str) -> str:
    stripped = api_key.strip()
    if not stripped:
        return "Not configured"
    if len(stripped) <= 4:
        return "Configured"
    return f"Configured ending with {stripped[-4:]}"
