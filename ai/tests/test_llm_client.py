import asyncio
import pytest
import httpx

from app.core.llm_client import LLMClient, LLMError


def test_missing_llm_settings(monkeypatch):
    monkeypatch.delenv("LLM_BASE_URL", raising=False)
    monkeypatch.delenv("LLM_API_KEY", raising=False)
    monkeypatch.delenv("LLM_MODEL", raising=False)

    with pytest.raises(ValueError, match="Для подключения LLM"):
        LLMClient()

def test_generate_success(monkeypatch):
    monkeypatch.setenv("LLM_BASE_URL", "https://llm.example/v1")
    monkeypatch.setenv("LLM_API_KEY", "test-key")
    monkeypatch.setenv("LLM_MODEL", "test-model")

    messages = [
        {"role": "user", "content": "Здравствуйте!"},
    ]

    async def fake_post(self, url, *, headers, json):
        assert url == "https://llm.example/v1/chat/completions"
        assert headers["Authorization"] == "Bearer test-key"
        assert json["model"] == "test-model"
        assert json["messages"] == messages
        assert json["temperature"] == 0.0

        return httpx.Response(
            status_code=200,
            request=httpx.Request("POST", url),
            json={
                "choices": [
                    {"message": {"content": "  Тестовый ответ  "}},
                ],
            },
        )

    monkeypatch.setattr(httpx.AsyncClient, "post", fake_post)

    client = LLMClient()
    answer = asyncio.run(client.generate(messages))

    assert answer == "Тестовый ответ"


def test_generate_http_error(monkeypatch):
    monkeypatch.setenv("LLM_BASE_URL", "https://llm.example/v1")
    monkeypatch.setenv("LLM_API_KEY", "test-key")
    monkeypatch.setenv("LLM_MODEL", "test-model")

    async def fake_post(self, url, *, headers, json):
        return httpx.Response(
            status_code=503,
            request=httpx.Request("POST", url),
        )

    monkeypatch.setattr(httpx.AsyncClient, "post", fake_post)

    client = LLMClient()

    with pytest.raises(LLMError, match="Ошибка обращения к LLM"):
        asyncio.run(
            client.generate(
                [{"role": "user", "content": "Здравствуйте!"}]
            )
        )

def test_generate_empty_answer(monkeypatch):
    monkeypatch.setenv("LLM_BASE_URL", "https://llm.example/v1")
    monkeypatch.setenv("LLM_API_KEY", "test-key")
    monkeypatch.setenv("LLM_MODEL", "test-model")

    async def fake_post(self, url, *, headers, json):
        return httpx.Response(
            status_code=200,
            request=httpx.Request("POST", url),
            json={
                "choices": [
                    {"message": {"content": "   "}},
                ],
            },
        )

    monkeypatch.setattr(httpx.AsyncClient, "post", fake_post)

    client = LLMClient()

    with pytest.raises(
        LLMError,
        match="LLM вернула пустой или некорректный ответ",
    ):
        asyncio.run(
            client.generate(
                [{"role": "user", "content": "Здравствуйте!"}]
            )
        )

@pytest.mark.parametrize(
    "response_body",
    [
        {},
        {"choices": []},
        {"choices": [{"message": None}]},
    ],
)
def test_generate_invalid_response(monkeypatch, response_body):
    monkeypatch.setenv("LLM_BASE_URL", "https://llm.example/v1")
    monkeypatch.setenv("LLM_API_KEY", "test-key")
    monkeypatch.setenv("LLM_MODEL", "test-model")

    async def fake_post(self, url, *, headers, json):
        return httpx.Response(
            status_code=200,
            request=httpx.Request("POST", url),
            json=response_body,
        )

    monkeypatch.setattr(httpx.AsyncClient, "post", fake_post)

    client = LLMClient()

    with pytest.raises(
        LLMError,
        match="Некорректный формат ответа LLM",
    ):
        asyncio.run(
            client.generate(
                [{"role": "user", "content": "Здравствуйте!"}]
            )
        )

def test_generate_timeout(monkeypatch):
    monkeypatch.setenv("LLM_BASE_URL", "https://llm.example/v1")
    monkeypatch.setenv("LLM_API_KEY", "test-key")
    monkeypatch.setenv("LLM_MODEL", "test-model")

    async def fake_post(self, url, *, headers, json):
        raise httpx.ReadTimeout(
            "Test timeout",
            request=httpx.Request("POST", url),
        )

    monkeypatch.setattr(httpx.AsyncClient, "post", fake_post)

    client = LLMClient()

    with pytest.raises(LLMError, match="Ошибка обращения к LLM"):
        asyncio.run(
            client.generate(
                [{"role": "user", "content": "Здравствуйте!"}]
            )
        )