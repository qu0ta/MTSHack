from time import perf_counter

from app.schemas.chat import ChatRequest, ChatResponse

class AIOrchestrator:
    async def chat(self, request: ChatRequest) -> ChatResponse:
        started_at = perf_counter()
        answer =  "Здравствуйте! Я консультант МТС EnergyTool. Чем могу помочь?"

        response_time_ms = int((perf_counter() - started_at) * 1000)

        return ChatResponse(
            answer=answer,
            client_data=request.client_data,
            sources=[],
            lead=None,
            response_time_ms=response_time_ms,
        )