import threading
from unittest.mock import Mock

import pytest
from fastapi.testclient import TestClient

from app.main import create_app
from app.services.rag_adapter import RAGAdapter


def payload(**changes):
    return {"session_id": "00000000-0000-0000-0000-000000000001", "message": "Нужен учёт", "history": [], "client_data": {}, **changes}


def test_go_contract_and_independent_sessions():
    generate = Mock(return_value={"answer": "Ответ ML", "sources": ["22-askue.md"]})
    init = Mock()
    with TestClient(create_app(RAGAdapter(generate, init))) as client:
        assert client.get("/health").json() == {"status": "ok"}
        assert client.get("/ready").status_code == 200
        data = {"company_name": "Компания", "connected_power_kw": 900}
        history = [{"role": "user", "content": "500 кВт"}]
        response = client.post("/api/v1/chat", json=payload(history=history, client_data=data))
        assert response.status_code == 200
        body = response.json()
        assert set(body) == {"answer", "sources", "client_data", "lead", "response_time_ms", "degraded", "timings_ms"}
        assert body["answer"] == "Ответ ML"
        assert body["sources"] == ["22-askue.md"]
        assert body["client_data"]["connected_power_kw"] == 900
        assert body["lead"] is None
        assert body["response_time_ms"] >= 0
        generate.assert_called_once()
        assert generate.call_args.kwargs["query"] == "Нужен учёт"
        assert generate.call_args.kwargs["history"][1:] == history
        assert client.post("/api/v1/chat", json=payload(session_id="second", history=None)).status_code == 200
        assert generate.call_args.kwargs["history"] == []
    init.assert_called_once()


@pytest.mark.parametrize("changes", [{"message": " "}, {"history": [{"role": "system", "content": "instructions"}]}, {"client_data": {"connected_power_kw": -1}}, {"message": "a" * 10001}])
def test_invalid_request_does_not_call_ml(changes):
    generate = Mock()
    with TestClient(create_app(RAGAdapter(generate, lambda: None))) as client:
        assert client.post("/api/v1/chat", json=payload(**changes)).status_code == 422
    generate.assert_not_called()


@pytest.mark.parametrize("result, status, code", [
    ({"sources": []}, 502, "invalid_ml_response"),
    (RuntimeError("secret-provider-error"), 503, "ml_unavailable"),
])
def test_dependency_errors_are_safe(result, status, code):
    generate = Mock(side_effect=result) if isinstance(result, Exception) else Mock(return_value=result)
    with TestClient(create_app(RAGAdapter(generate, lambda: None))) as client:
        response = client.post("/api/v1/chat", json=payload())
        assert response.status_code == status
        assert response.json() == {"detail": {"code": code}}
        assert "secret-provider-error" not in response.text


def test_http_timeout_and_busy_do_not_queue_unbounded_workers():
    release = threading.Event()
    def generate(**kwargs):
        release.wait(2)
        return {"answer": "late", "sources": []}
    with TestClient(create_app(RAGAdapter(generate, lambda: None, timeout_seconds=0.05, max_concurrency=1))) as client:
        try:
            assert client.post("/api/v1/chat", json=payload()).status_code == 504
            assert client.post("/api/v1/chat", json=payload()).status_code == 503
            assert client.get("/health").status_code == 200
        finally:
            release.set()


@pytest.mark.parametrize('field, limit', [('phone', 30), ('contact_name', 255), ('company_name', 255)])
def test_lead_contact_lengths_match_backend(field, limit):
    generate = Mock()
    with TestClient(create_app(RAGAdapter(generate, lambda: None))) as client:
        response = client.post('/api/v1/chat', json=payload(client_data={field: 'a' * (limit + 1)}, lead_requested=True))
        assert response.status_code == 422
        generate.assert_not_called()
