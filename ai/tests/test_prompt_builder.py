from app.core.prompt_builder import PromptBuilder, SYSTEM_PROMPT
from app.schemas.chat import ChatRequest


def test_build_prompt():
    request = ChatRequest(
        session_id="test-session-1",
        message="А сколько это стоит?",
        history=[
            {
                "role": "user",
                "content": "Нужен учёт по цехам",
            },
            {
                "role": "assistant",
                "content": "Какие приборы уже установлены?",
            },
        ],
        client_data={
            "company_name": "Тестовая компания",
        },
    )

    fragments = ["Предварительная оценка бесплатна."]

    messages = PromptBuilder().build(request, fragments)

    assert len(messages) == 5
    assert messages[0] == {
        "role": "system",
        "content": SYSTEM_PROMPT,
    }

    context = messages[1]["content"]

    assert "Тестовая компания" in context
    assert "Предварительная оценка бесплатна." in context
    assert '"phone"' not in context

    assert messages[2:] == [
        {"role": "user", "content": "Нужен учёт по цехам"},
        {
            "role": "assistant",
            "content": "Какие приборы уже установлены?",
        },
        {"role": "user", "content": "А сколько это стоит?"},
    ]