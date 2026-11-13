from time import perf_counter

from app.schemas.chat import ChatRequest, ChatResponse

from app.services.lead_service import prepare_lead


class AIOrchestrator:
    async def chat(self, request: ChatRequest) -> ChatResponse:
        started_at = perf_counter()
        answer =  "Здравствуйте! Я консультант МТС EnergyTool. Чем могу помочь?"

        lead = None
        if request.lead_requested:
            answer, lead = prepare_lead(request)

        response_time_ms = int((perf_counter() - started_at) * 1000)

        return ChatResponse(
            answer=answer,
            client_data=request.client_data,
            sources=[],
            lead=lead,
            response_time_ms=response_time_ms,
        )