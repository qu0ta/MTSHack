from app.core.dialog_manager import DialogManager
from app.schemas.chat import ChatRequest


def test_search_query_without_history():
    request = ChatRequest(
        session_id="test-session-1",
        message="Нужен учёт по цехам",
        history=[],
        client_data={},
    )

    query = DialogManager().build_search_query(request)

    assert query == "Нужен учёт по цехам"


def test_search_query_with_history():
    request = ChatRequest(
        session_id="test-session-1",
        message="А сколько это стоит?",
        history=[
            {
                "role": "user",
                "content": "Хотим снизить расходы",
            },
            {
                "role": "user",
                "content": "Нужен учёт по цехам",
            },
            {
                "role": "assistant",
                "content": "Какие приборы уже установлены?",
            },
        ],
        client_data={},
    )

    query = DialogManager().build_search_query(request)

    assert query == (
        "Предыдущая реплика клиента: Нужен учёт по цехам\n"
        "Текущий вопрос: А сколько это стоит?"
    )