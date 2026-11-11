from fastapi import APIRouter

from app.core.orchestrator import AIOrchestrator
from app.schemas.chat import ChatRequest, ChatResponse

router = APIRouter()

@router.get("/health")
async def health() -> dict[str,str]:
    return {"status": "ok"}

@router.post("/api/v1/chat", response_model=ChatResponse)
async def chat(request: ChatRequest) -> ChatResponse:
    orchestrator = AIOrchestrator()
    return await orchestrator.chat(request)
