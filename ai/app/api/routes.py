from fastapi import APIRouter, HTTPException, Request

from app.schemas.chat import ChatRequest, ChatResponse
from app.services.rag_adapter import MLBusy, MLInvalidResponse, MLTimeout, MLUnavailable

router = APIRouter()


@router.get("/health")
async def health() -> dict[str, str]:
    return {"status": "ok"}


@router.get("/ready")
async def ready(request: Request) -> dict:
    orchestrator = getattr(request.app.state, "orchestrator", None)
    if orchestrator is None:
        raise HTTPException(503, detail={"code": "not_ready"})
    health = orchestrator.rag_adapter.health()
    return {"status": "degraded" if health["degraded"] else "ready", "ml": health}


@router.post("/api/v1/chat", response_model=ChatResponse)
async def chat(payload: ChatRequest, request: Request) -> ChatResponse:
    orchestrator = getattr(request.app.state, "orchestrator", None)
    if orchestrator is None:
        raise HTTPException(503, detail={"code": "not_ready"})
    try:
        return await orchestrator.chat(payload)
    except MLTimeout:
        raise HTTPException(504, detail={"code": "ml_timeout"}) from None
    except MLBusy:
        raise HTTPException(503, detail={"code": "ml_busy"}, headers={"Retry-After": "2"}) from None
    except MLUnavailable:
        raise HTTPException(503, detail={"code": "ml_unavailable"}) from None
    except MLInvalidResponse:
        raise HTTPException(502, detail={"code": "invalid_ml_response"}) from None
