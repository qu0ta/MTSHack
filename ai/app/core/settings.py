import importlib
import math
import os
from functools import partial
from pathlib import Path

from app.services.rag_adapter import RAGAdapter


def create_ml_adapter() -> RAGAdapter:
    """Import ML only after environment has been configured by the launcher."""
    required = ("LLM_BASE_URL", "LLM_API_KEY", "LLM_MODEL")
    missing = [name for name in required if not os.getenv(name, "").strip()]
    if missing:
        raise ValueError("Missing settings: " + ", ".join(missing))
    # Same explicit defaults as the delivered ML package; provider may reject them.
    os.environ.setdefault("EMB_MODEL", "baai/bge-m3")
    os.environ.setdefault("RERANKER_MODEL", "qwen/qwen3-reranker-8b")
    kb_setting = os.getenv("KB_PATH", "").strip()
    kb_path = (
        Path(kb_setting).expanduser() if kb_setting
        else Path(os.getenv("DATA_DIR", "data")) / "knowledge-base.zip"
    ).resolve()
    if not kb_path.exists():
        raise ValueError("Knowledge base not found; configure KB_PATH or DATA_DIR")
    timeout = float(os.getenv("AI_REQUEST_TIMEOUT_SECONDS", "55"))
    concurrency = int(os.getenv("AI_MAX_CONCURRENCY", "2"))
    if not math.isfinite(timeout) or not 0 < timeout <= 300:
        raise ValueError("AI_REQUEST_TIMEOUT_SECONDS must be in (0, 300]")
    if not 1 <= concurrency <= 8:
        raise ValueError("AI_MAX_CONCURRENCY must be between 1 and 8")
    rag_budget = float(os.getenv("RAG_BUDGET_SECONDS", "45"))
    if not math.isfinite(rag_budget) or not 0 < rag_budget < timeout:
        raise ValueError("RAG_BUDGET_SECONDS must be positive and below AI_REQUEST_TIMEOUT_SECONDS")
    for name, default in (("RAG_MAX_HISTORY", "12"), ("RAG_RETRIEVAL_TURNS", "6")):
        if not 1 <= int(os.getenv(name, default)) <= 100:
            raise ValueError(f"{name} must be between 1 and 100")
    module = importlib.import_module(os.getenv("ML_MODULE", "app.ml.rag_api"))
    if not callable(getattr(module, "init", None)) or not callable(getattr(module, "generate_answer", None)):
        raise ValueError("ML_MODULE must expose init and generate_answer")
    return RAGAdapter(
        generate_function=module.generate_answer,
        init_function=partial(module.init, kb_path=kb_path),
        timeout_seconds=timeout,
        max_concurrency=concurrency,
        health_function=getattr(module, "health", None),
    )
