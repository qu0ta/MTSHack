from contextlib import asynccontextmanager

from fastapi import FastAPI

from app.api.routes import router
from app.core.orchestrator import AIOrchestrator
from app.core.settings import create_ml_adapter
from app.services.rag_adapter import RAGAdapter


def create_app(adapter: RAGAdapter | None = None) -> FastAPI:
    @asynccontextmanager
    async def lifespan(application: FastAPI):
        ml = adapter if adapter is not None else create_ml_adapter()
        await ml.initialize()
        application.state.orchestrator = AIOrchestrator(ml)
        try:
            yield
        finally:
            application.state.orchestrator = None

    application = FastAPI(title="MTS EnergyTool AI", version="0.2.0", lifespan=lifespan)
    application.include_router(router)
    return application


app = create_app()
