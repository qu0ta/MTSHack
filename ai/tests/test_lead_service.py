from unittest.mock import Mock

from fastapi.testclient import TestClient

from app.main import create_app
from app.services.rag_adapter import RAGAdapter


CONTACTS = {"contact_name": "Анна", "phone": "+79991234567", "company_name": "ООО Тест"}


def test_complete_contacts_do_not_implicitly_create_lead():
    generate = Mock(return_value={"answer": "Консультация", "sources": []})
    with TestClient(create_app(RAGAdapter(generate, lambda: None))) as client:
        result = client.post("/api/v1/chat", json={
            "session_id": "one", "message": "Не хочу заявку", "client_data": CONTACTS,
        })
        assert result.status_code == 200
        assert result.json()["lead"] is None
        generate.assert_called_once()


def test_explicit_lead_uses_backend_data_and_no_llm_call():
    generate = Mock()
    with TestClient(create_app(RAGAdapter(generate, lambda: None))) as client:
        result = client.post("/api/v1/chat", json={
            "session_id": "one", "message": "Свяжитесь со мной", "lead_requested": True,
            "client_data": {**CONTACTS, "connected_power_kw": 900},
            "history": [
                {"role": "user", "content": "Хочу снизить расходы"},
                {"role": "assistant", "content": "Экономия гарантирована"},
            ],
        })
        assert result.status_code == 200
        body = result.json()
        assert body["lead"]["phone"] == CONTACTS["phone"]
        assert "Хочу снизить расходы" in body["lead"]["summary"]
        assert "900" in body["lead"]["summary"]
        assert "Экономия гарантирована" not in body["lead"]["summary"]
        assert "подготовлены" in body["answer"]
        assert "отправлена" not in body["answer"]
        generate.assert_not_called()


def test_incomplete_lead_returns_missing_fields_without_guessing():
    generate = Mock()
    with TestClient(create_app(RAGAdapter(generate, lambda: None))) as client:
        response = client.post("/api/v1/chat", json={
            "session_id": "one", "message": "Хочу заявку", "lead_requested": True,
            "client_data": {"contact_name": "Анна", "phone": " "},
        })
        assert response.status_code == 200
        assert response.json()["lead"] is None
        assert "телефон" in response.json()["answer"]
        assert "название компании" in response.json()["answer"]
        generate.assert_not_called()


def test_new_backend_partial_form_with_null_history():
    generate = Mock()
    with TestClient(create_app(RAGAdapter(generate, lambda: None))) as client:
        response = client.post("/api/v1/chat", json={
            "session_id": "one", "message": "Подтверждаю", "lead_requested": True,
            "history": None,
            "client_data": {"contact_name": "Анна", "phone": "", "company_name": "  ", "region": ""},
        })
        assert response.status_code == 200
        body = response.json()
        assert body["lead"] is None
        assert body["client_data"]["phone"] is None
        assert body["client_data"]["company_name"] is None
        assert "телефон" in body["answer"]
        assert "название компании" in body["answer"]
        generate.assert_not_called()


def test_summary_keeps_task_before_contact_exchange():
    from app.schemas.chat import ChatRequest
    from app.services.lead_service import LeadService
    request = ChatRequest(session_id='summary', message='Свяжитесь со мной', lead_requested=True,
        client_data={**CONTACTS, 'region': 'Москва'}, history=[
            {'role': 'user', 'content': 'Нужно наладить учёт электроэнергии на складе.'},
            {'role': 'assistant', 'content': 'Гарантирую экономию 50 процентов.'},
            {'role': 'user', 'content': 'Да'},
            {'role': 'user', 'content': 'Хочу заявку'},
            {'role': 'user', 'content': 'Анна'},
            {'role': 'user', 'content': '+79991234567'},
        ])
    lead = LeadService().prepare(request)
    assert 'Нужно наладить учёт электроэнергии на складе.' in lead.summary
    assert 'Москва' in lead.summary
    assert '50 процентов' not in lead.summary
    assert 'Следующий шаг' in lead.summary
    assert len(lead.summary) < 1000
