from pathlib import Path

from image_generator.config import AppConfig, load_config, redacted_api_key, save_config, validate_config
from image_generator.models import EndpointType


def test_config_round_trips_json(tmp_path: Path) -> None:
    path = tmp_path / "config.json"
    config = AppConfig(
        base_url="https://api.example.com/",
        api_key="sk-test-secret",
        timeout_seconds=90,
        max_concurrency=4,
        default_save_dir=tmp_path / "images",
        default_endpoint_type=EndpointType.CHAT_COMPLETIONS,
        default_model="image-model",
    )

    save_config(config, path)
    loaded = load_config(path)

    assert loaded.base_url == "https://api.example.com"
    assert loaded.api_key == "sk-test-secret"
    assert loaded.timeout_seconds == 90
    assert loaded.max_concurrency == 4
    assert loaded.default_save_dir == tmp_path / "images"
    assert loaded.default_endpoint_type is EndpointType.CHAT_COMPLETIONS
    assert loaded.default_model == "image-model"


def test_missing_config_loads_defaults(tmp_path: Path) -> None:
    config = load_config(tmp_path / "missing.json")

    assert config.base_url == ""
    assert config.api_key == ""
    assert config.timeout_seconds == 120
    assert config.max_concurrency == 3
    assert config.default_save_dir.name == "ImageGenerator"
    assert config.default_endpoint_type is EndpointType.IMAGES


def test_validate_config_reports_actionable_messages(tmp_path: Path) -> None:
    config = AppConfig(base_url="", api_key="", default_save_dir=tmp_path / "not-created")

    messages = validate_config(config)

    assert "Base URL is required" in messages
    assert "API key is required" in messages


def test_redacted_api_key_never_returns_full_secret() -> None:
    assert redacted_api_key("") == "Not configured"
    assert redacted_api_key("sk-short") == "Configured ending with hort"
    assert redacted_api_key("sk-1234567890abcdef") == "Configured ending with cdef"
