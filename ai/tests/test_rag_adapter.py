import asyncio
import threading
from unittest.mock import Mock

import pytest

from app.services.rag_adapter import RAGAdapter, MLBusy, MLInvalidResponse, MLTimeout, MLUnavailable


def test_sync_ml_runs_in_worker_and_initializes_once():
    event_loop_thread = threading.get_ident()
    init = Mock()
    history = [{"role": "user", "content": "Завод"}]

    def generate(*, query, history):
        assert threading.get_ident() != event_loop_thread
        assert query == "900 кВт"
        assert history == [{"role": "user", "content": "Завод"}]
        return {"answer": " Ответ ML ", "sources": ["a.md", " a.md ", ""], "timings_ms": {"total": 5}}

    async def scenario():
        adapter = RAGAdapter(generate, init)
        await asyncio.gather(adapter.initialize(), adapter.initialize())
        return await adapter.generate("900 кВт", history)

    result = asyncio.run(scenario())
    init.assert_called_once()
    assert result.answer == "Ответ ML"
    assert result.sources == ["a.md"]


@pytest.mark.parametrize("result", [None, {}, {"answer": " ", "sources": []}, {"answer": "OK", "sources": "file.md"}])
def test_rejects_invalid_contract(result):
    async def scenario():
        adapter = RAGAdapter(lambda **kwargs: result, lambda: None)
        await adapter.initialize()
        with pytest.raises(MLInvalidResponse):
            await adapter.generate("question", [])
    asyncio.run(scenario())


def test_timeout_keeps_worker_slot_until_sync_call_finishes():
    entered = threading.Event()
    release = threading.Event()

    def generate(**kwargs):
        entered.set()
        release.wait(2)
        return {"answer": "OK", "sources": []}

    async def scenario():
        adapter = RAGAdapter(generate, lambda: None, timeout_seconds=0.05, max_concurrency=1)
        await adapter.initialize()
        try:
            with pytest.raises(MLTimeout):
                await adapter.generate("first", [])
            assert entered.is_set()
            with pytest.raises(MLBusy):
                await adapter.generate("second", [])
        finally:
            release.set()
            await asyncio.gather(*adapter._tasks)
            await asyncio.sleep(0)
        assert (await adapter.generate("third", [])).answer == "OK"
    asyncio.run(scenario())


def test_init_failure_does_not_mark_adapter_ready():
    async def scenario():
        adapter = RAGAdapter(Mock(), Mock(side_effect=RuntimeError("failure")))
        with pytest.raises(MLUnavailable):
            await adapter.initialize()
        with pytest.raises(MLUnavailable):
            await adapter.generate("hello", [])
    asyncio.run(scenario())
