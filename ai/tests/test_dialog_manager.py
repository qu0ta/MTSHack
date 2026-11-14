from app.core.dialog_manager import DialogManager
from app.schemas.chat import ChatRequest


def test_history_is_passed_without_repeating_current_question():
    request = ChatRequest(session_id="one", message="Теперь 900 кВт", history=[
        {"role": "user", "content": "500 кВт"},
        {"role": "assistant", "content": "Уточните мощность"},
    ])
    assert DialogManager().history_for_ml(request) == [m.model_dump() for m in request.history]
    assert DialogManager().history_for_ml(ChatRequest(session_id="two", message="Привет")) == []


def test_business_context_does_not_forward_structured_contacts():
    request = ChatRequest(session_id="one", message="Что предложите?", client_data={
        "region": "Москва", "phone": "+79991234567", "contact_name": "Анна", "company_name": "ООО Тест",
    })
    text = DialogManager().history_for_ml(request)[0]["content"]
    assert "Москва" in text
    assert "+79991234567" not in text
    assert "Анна" not in text
    assert "ООО Тест" not in text
