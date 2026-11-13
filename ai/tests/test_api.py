from fastapi.testclient import TestClient

from app.main import app

client = TestClient(app)

def test_health():
    response = client.get("/health")

    assert response.status_code == 200
    assert response.json() == {"status": "ok"}

def test_chat():
    payload = {
        "session_id": "test-session-1",
        "message": "Хочу снизить расходы на электроэнергию",
        "history": [],
        "client_data": {},
    }

    response = client.post("/api/v1/chat", json=payload)

    assert response.status_code == 200

    data = response.json()

    assert data["answer"] == (
        "Здравствуйте! Я консультант МТС EnergyTool. Чем могу помочь?"
    )
    assert data["sources"] == []
    assert isinstance(data["response_time_ms"], int)
    assert data["response_time_ms"] >= 0

def test_chat_validation():
    payload = {
        "session_id": "test-session-1",
        "history": [],
        "client_data": {},
    }

    response = client.post("/api/v1/chat", json=payload)

    assert response.status_code == 422

    errors = response.json()["detail"]

    assert any(
        error["loc"] == ["body", "message"]
        and error["type"] == "missing"
        for error in errors
    )

def test_chat_preserves_client_data():
    client_data = {
        "company_name": "Тестовая компания",
        "contact_name": "Тестовый Клиент",
        "phone": "+70000000000",
        "industry": "Производство",
        "region": "Москва",
        "connected_power_kw": 150.5,
        "monthly_consumption_kwh": 25000.0,
    }

    payload = {
        "session_id": "test-session-1",
        "message": "Нужна консультация",
        "history": [],
        "client_data": client_data,
    }

    response = client.post("/api/v1/chat", json=payload)

    assert response.status_code == 200
    assert response.json()["client_data"] == client_data

def test_chat_has_no_lead():
    payload = {
        "session_id": "test-session-1",
        "message": "Хочу связаться с менеджером",
        "history": [],
        "client_data": {
            "contact_name": "Тестовый Клиент",
            "phone": "+70000000000",
            "company_name": "Тестовая компания",
        },
    }

    response = client.post("/api/v1/chat", json=payload)

    assert response.status_code == 200

    data = response.json()

    assert "lead" in data
    assert data["lead"] is None
def test_confirmed_lead():
    payload = {"session_id": "test", "message": "Подтверждаю", "history": None, "lead_requested": True}
    incomplete = client.post("/api/v1/chat", json=payload).json()
    assert incomplete["lead"] is None and "телефон" in incomplete["answer"]
    payload["client_data"] = {"contact_name": "Анна", "company_name": "Пример", "phone": "+70000000000"}
    response = client.post("/api/v1/chat", json=payload)
    assert response.status_code == 200
    assert response.json()["lead"] == {**payload["client_data"], "summary": "Подтверждаю"}
