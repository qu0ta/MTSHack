from time import perf_counter

from app.core.dialog_manager import DialogManager
from app.schemas.chat import ChatRequest, ChatResponse
from app.services.rag_adapter import RAGAdapter
from app.services.lead_service import LeadService


class AIOrchestrator:
    def __init__(self, rag_adapter: RAGAdapter) -> None:
        self.rag_adapter = rag_adapter
        self.dialog_manager = DialogManager()
        self.lead_service = LeadService()

    async def chat(self, request: ChatRequest) -> ChatResponse:
        started = perf_counter()
        if request.lead_requested:
            lead = self.lead_service.prepare(request)
            missing = self.lead_service.missing_fields(request.client_data)
            answer = (
                "Данные заявки подготовлены для передачи менеджеру."
                if lead is not None else
                "Для подготовки заявки нужны: " + ", ".join(missing) + "."
            )
            return ChatResponse(
                answer=answer, sources=[], lead=lead,
                client_data=request.client_data.model_copy(deep=True),
                response_time_ms=int((perf_counter() - started) * 1000),
            )
        result = await self.rag_adapter.generate(
            query=request.message,
            history=self.dialog_manager.history_for_ml(request),
        )
        return ChatResponse(
            answer=result.answer,
            sources=result.sources,
            client_data=request.client_data.model_copy(deep=True),
            lead=None,
            response_time_ms=int((perf_counter() - started) * 1000),
            degraded=result.degraded,
            timings_ms=result.timings_ms,
        )
