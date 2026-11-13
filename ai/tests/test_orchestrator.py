import asyncio
from unittest.mock import AsyncMock

from app.core.orchestrator import AIOrchestrator
from app.schemas.chat import ChatRequest
from app.services.rag_adapter import MLAnswer, RAGAdapter


def test_orchestrator_uses_ml_answer_once_and_preserves_client_data():
    adapter = AsyncMock(spec=RAGAdapter)
    adapter.generate.return_value = MLAnswer(answer="Ответ ML", sources=["21-orem.md"])
    request = ChatRequest(session_id="test", message="А при 900 кВт?", history=[
        {"role": "user", "content": "Мощность 500 кВт"},
    ], client_data={"region": "Москва", "connected_power_kw": 900})
    response = asyncio.run(AIOrchestrator(adapter).chat(request))
    adapter.generate.assert_awaited_once()
    call = adapter.generate.call_args.kwargs
    assert call["query"] == request.message
    assert '"connected_power_kw":900.0' in call["history"][0]["content"]
    assert call["history"][1:] == [m.model_dump() for m in request.history]
    assert response.answer == "Ответ ML"
    assert response.sources == ["21-orem.md"]
    assert response.client_data == request.client_data
    assert response.client_data is not request.client_data
    assert response.lead is None
    assert response.response_time_ms >= 0
