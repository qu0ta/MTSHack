from types import SimpleNamespace
from unittest.mock import Mock
import asyncio

import pytest

from app.core.settings import create_ml_adapter


def test_missing_settings_fail_before_import(monkeypatch):
    for name in ("LLM_BASE_URL", "LLM_API_KEY", "LLM_MODEL", "EMB_MODEL", "RERANKER_MODEL"):
        monkeypatch.delenv(name, raising=False)
    with pytest.raises(ValueError, match="Missing settings"):
        create_ml_adapter()


def test_runtime_import_and_explicit_kb_path(monkeypatch, tmp_path):
    for name in ("LLM_BASE_URL", "LLM_API_KEY", "LLM_MODEL", "EMB_MODEL", "RERANKER_MODEL"):
        monkeypatch.setenv(name, "test-value")
    monkeypatch.setenv("KB_PATH", str(tmp_path))
    monkeypatch.setenv("ML_MODULE", "ml.rag_api")
    module = SimpleNamespace(init=Mock(), generate_answer=Mock(return_value={"answer": "OK", "sources": []}))
    import_module = Mock(return_value=module)
    monkeypatch.setattr("app.core.settings.importlib.import_module", import_module)
    adapter = create_ml_adapter()
    asyncio.run(adapter.initialize())
    import_module.assert_called_once_with("ml.rag_api")
    module.init.assert_called_once_with(kb_path=tmp_path)


def test_ml_budget_must_fit_http_wait_budget(monkeypatch, tmp_path):
    for name in ('LLM_BASE_URL', 'LLM_API_KEY', 'LLM_MODEL'):
        monkeypatch.setenv(name, 'test-value')
    monkeypatch.setenv('KB_PATH', str(tmp_path))
    monkeypatch.setenv('AI_REQUEST_TIMEOUT_SECONDS', '55')
    monkeypatch.setenv('RAG_BUDGET_SECONDS', '60')
    with pytest.raises(ValueError, match='RAG_BUDGET_SECONDS'):
        create_ml_adapter()
