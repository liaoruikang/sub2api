import json

import pytest

from image_generator.config import AppConfig
from image_generator.models import EndpointType, GenerationParams
from image_generator.openai_client import (
    APIError,
    OpenAIImageClient,
    parse_chat_payloads,
    parse_image_payloads,
    parse_text_for_images,
)


def make_client() -> OpenAIImageClient:
    return OpenAIImageClient(AppConfig(base_url="https://api.example.com/", api_key="sk-secret"))


def test_build_images_payload_uses_common_image_fields() -> None:
    client = make_client()
    params = GenerationParams(prompt="a fox", model="gpt-image-1", size="512x512", n=2)

    payload = client.build_images_payload(params)

    assert payload == {
        "model": "gpt-image-1",
        "prompt": "a fox",
        "size": "512x512",
        "n": 2,
        "quality": "standard",
        "style": "natural",
        "response_format": "b64_json",
    }


def test_parse_image_payloads_extracts_b64_and_url_from_same_item() -> None:
    payloads = parse_image_payloads(
        {"data": [{"b64_json": "YWJj", "url": "https://cdn.example.com/image.png"}]}
    )

    assert [(payload.kind, payload.value) for payload in payloads] == [
        ("b64_json", "YWJj"),
        ("url", "https://cdn.example.com/image.png"),
    ]


def test_build_chat_payload_sets_messages_and_stream() -> None:
    client = make_client()
    params = GenerationParams(
        endpoint_type=EndpointType.CHAT_COMPLETIONS,
        prompt="draw a fox",
        model="chat-image-model",
        stream=True,
    )

    payload = client.build_chat_payload(params)

    assert payload == {
        "model": "chat-image-model",
        "messages": [{"role": "user", "content": "draw a fox"}],
        "stream": True,
    }


@pytest.mark.asyncio
async def test_generate_images_parses_b64_response(httpx_mock) -> None:
    httpx_mock.add_response(
        method="POST",
        url="https://api.example.com/v1/images/generations",
        json={"data": [{"b64_json": "YWJj"}]},
    )
    client = make_client()

    payloads = await client.generate(GenerationParams(prompt="fox"))

    assert payloads[0].kind == "b64_json"
    assert payloads[0].value == "YWJj"
    request = httpx_mock.get_request()
    assert request.headers["authorization"] == "Bearer sk-secret"


@pytest.mark.asyncio
async def test_generate_images_raises_api_error_with_status(httpx_mock) -> None:
    httpx_mock.add_response(
        method="POST",
        url="https://api.example.com/v1/images/generations",
        status_code=401,
        json={"error": {"message": "bad key"}},
    )
    client = make_client()

    with pytest.raises(APIError) as exc_info:
        await client.generate(GenerationParams(prompt="fox"))

    assert "401" in str(exc_info.value)
    assert "bad key" in str(exc_info.value)
    assert "sk-secret" not in str(exc_info.value)


@pytest.mark.asyncio
async def test_generate_images_raises_api_error_for_redirect_status(httpx_mock) -> None:
    httpx_mock.add_response(
        method="POST",
        url="https://api.example.com/v1/images/generations",
        status_code=302,
        headers={"location": "https://api.example.com/login"},
    )
    client = make_client()

    with pytest.raises(APIError) as exc_info:
        await client.generate(GenerationParams(prompt="fox"))

    assert "302" in str(exc_info.value)
    assert "sk-secret" not in str(exc_info.value)


def test_parse_chat_payloads_extracts_image_url_from_content() -> None:
    payloads = parse_chat_payloads(
        {
            "choices": [
                {
                    "message": {
                        "content": "Result: https://cdn.example.com/out/image.png"
                    }
                }
            ]
        }
    )

    assert payloads[0].kind == "url"
    assert payloads[0].value == "https://cdn.example.com/out/image.png"


def test_parse_chat_payloads_extracts_image_url_from_multimodal_dict() -> None:
    payloads = parse_chat_payloads(
        {
            "choices": [
                {
                    "message": {
                        "content": [
                            {
                                "type": "image_url",
                                "image_url": {"url": "https://cdn.example.com/out/multi.png"},
                            }
                        ]
                    }
                }
            ]
        }
    )

    assert payloads[0].kind == "url"
    assert payloads[0].value == "https://cdn.example.com/out/multi.png"


def test_parse_chat_payloads_strips_crlf_from_data_url_base64() -> None:
    payloads = parse_chat_payloads(
        {
            "choices": [
                {
                    "message": {
                        "content": "data:image/png;base64,YW\r\nJj"
                    }
                }
            ]
        }
    )

    assert payloads[0].kind == "b64_json"
    assert payloads[0].value == "YWJj"


def test_parse_chat_payloads_strips_spaces_and_tabs_from_data_url_base64() -> None:
    payloads = parse_chat_payloads(
        {
            "choices": [
                {
                    "message": {
                        "content": "data:image/png;base64,YW \tJj"
                    }
                }
            ]
        }
    )

    assert payloads[0].kind == "b64_json"
    assert payloads[0].value == "YWJj"


def test_parse_text_for_images_stops_data_url_before_trailing_prose() -> None:
    payloads = parse_text_for_images("data:image/png;base64,YWJj done")

    assert len(payloads) == 1
    assert payloads[0].kind == "b64_json"
    assert payloads[0].value == "YWJj"


@pytest.mark.asyncio
async def test_streaming_chat_response_raises_api_error_with_status(httpx_mock) -> None:
    httpx_mock.add_response(
        method="POST",
        url="https://api.example.com/v1/chat/completions",
        status_code=401,
        json={"error": {"message": "bad stream key"}},
    )
    client = make_client()

    with pytest.raises(APIError) as exc_info:
        await client.generate(
            GenerationParams(
                endpoint_type=EndpointType.CHAT_COMPLETIONS,
                prompt="fox",
                stream=True,
            )
        )

    assert "401" in str(exc_info.value)
    assert "bad stream key" in str(exc_info.value)
    assert "sk-secret" not in str(exc_info.value)


@pytest.mark.asyncio
async def test_streaming_chat_response_records_events(httpx_mock) -> None:
    events: list[str] = []
    body = "\n".join(
        [
            'data: {"choices":[{"delta":{"content":"https://cdn.example.com/a.png"}}]}',
            "data: [DONE]",
            "",
        ]
    )
    httpx_mock.add_response(
        method="POST",
        url="https://api.example.com/v1/chat/completions",
        content=body.encode("utf-8"),
        headers={"content-type": "text/event-stream"},
    )
    client = make_client()

    payloads = await client.generate(
        GenerationParams(
            endpoint_type=EndpointType.CHAT_COMPLETIONS,
            prompt="fox",
            stream=True,
        ),
        event_callback=events.append,
    )

    assert payloads[0].kind == "url"
    assert payloads[0].value == "https://cdn.example.com/a.png"
    assert "stream event received" in events


@pytest.mark.asyncio
async def test_streaming_chat_response_parses_split_url_across_deltas(httpx_mock) -> None:
    body = "\n".join(
        [
            'data: {"choices":[{"delta":{"content":"https://cdn.example.com/out/"}}]}',
            'data: {"choices":[{"delta":{"content":"split-image.png"}}]}',
            "data: [DONE]",
            "",
        ]
    )
    httpx_mock.add_response(
        method="POST",
        url="https://api.example.com/v1/chat/completions",
        content=body.encode("utf-8"),
        headers={"content-type": "text/event-stream"},
    )
    client = make_client()

    payloads = await client.generate(
        GenerationParams(
            endpoint_type=EndpointType.CHAT_COMPLETIONS,
            prompt="fox",
            stream=True,
        )
    )

    assert payloads[0].kind == "url"
    assert payloads[0].value == "https://cdn.example.com/out/split-image.png"
