from pathlib import Path

import pytest

from image_generator.config import (
    AppConfig,
    default_save_dir,
    load_config,
    redacted_api_key,
    save_config,
    validate_config,
)
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


def test_load_config_recovers_from_malformed_unreadable_or_non_object_json(tmp_path: Path) -> None:
    malformed_path = tmp_path / "malformed.json"
    malformed_path.write_text("{not valid json", encoding="utf-8")
    non_object_path = tmp_path / "non-object.json"
    non_object_path.write_text("[1, 2, 3]", encoding="utf-8")

    malformed_config = load_config(malformed_path)
    unreadable_config = load_config(tmp_path)
    non_object_config = load_config(non_object_path)

    assert malformed_config.base_url == ""
    assert unreadable_config.default_endpoint_type is EndpointType.IMAGES
    assert non_object_config.default_model == "gpt-image-1"


@pytest.mark.parametrize(
    ("persisted_base_url", "persisted_api_key"),
    [
        (None, None),
        ([], {}),
    ],
)
def test_from_dict_defaults_bad_values_and_null_strings(
    persisted_base_url: object, persisted_api_key: object
) -> None:
    config = AppConfig.from_dict(
        {
            "base_url": persisted_base_url,
            "api_key": persisted_api_key,
            "timeout_seconds": "fast",
            "max_concurrency": None,
            "default_save_dir": None,
            "default_endpoint_type": "unknown",
            "default_model": None,
        }
    )

    assert config.base_url == ""
    assert config.api_key == ""
    assert config.timeout_seconds == 120
    assert config.max_concurrency == 3
    assert config.default_save_dir.name == "ImageGenerator"
    assert config.default_endpoint_type is EndpointType.IMAGES
    assert config.default_model == "gpt-image-1"


def test_from_dict_uses_default_save_dir_for_empty_persisted_directory() -> None:
    config = AppConfig.from_dict(
        {
            "base_url": "https://api.example.com",
            "api_key": "sk-test-secret",
            "default_save_dir": "",
        }
    )

    assert config.default_save_dir == default_save_dir()


def test_normalized_strips_base_url_whitespace_before_trailing_slashes() -> None:
    config = AppConfig(base_url="  https://api.example.com/path///  ", api_key="sk-test-secret")

    normalized = config.normalized()

    assert normalized.base_url == "https://api.example.com/path"


def test_validate_config_treats_whitespace_only_base_url_as_missing() -> None:
    config = AppConfig(base_url="   ", api_key="sk-test-secret")

    messages = validate_config(config)

    assert "Base URL is required" in messages


def test_validate_config_reports_raw_numeric_limits() -> None:
    config = AppConfig(
        base_url="https://api.example.com",
        api_key="sk-test-secret",
        timeout_seconds=0,
        max_concurrency=-2,
    )

    messages = validate_config(config)

    assert "Timeout must be at least 1 second" in messages
    assert "Concurrency must be at least 1" in messages


def test_validate_config_reports_actionable_messages(tmp_path: Path) -> None:
    config = AppConfig(base_url="", api_key="", default_save_dir=tmp_path / "not-created")

    messages = validate_config(config)

    assert "Base URL is required" in messages
    assert "API key is required" in messages


def test_redacted_api_key_never_returns_full_secret() -> None:
    assert redacted_api_key("") == "Not configured"
    assert redacted_api_key("abc") == "Configured"
    assert redacted_api_key("abcd") == "Configured"
    assert redacted_api_key("sk-short") == "Configured ending with hort"
    assert redacted_api_key("sk-1234567890abcdef") == "Configured ending with cdef"
