"""The delivered site1 ML pair, with external HTTP replaced by deterministic responses."""
import importlib
import json
from concurrent.futures import ThreadPoolExecutor

import httpx
import pytest
from fastapi.testclient import TestClient

from app.main import create_app
from app.services.rag_adapter import RAGAdapter


@pytest.fixture
def ml_runtime(tmp_path, monkeypatch):
    ml = importlib.import_module('app.ml.rag_api')
    monkeypatch.setattr(ml, '_state', {})
    for name in ('SYSTEM_PROMPT', 'STRICT', 'TIMEOUT', 'OPT_TIMEOUT', 'MAX_ATTEMPTS', 'DEADLINE', 'EXPAND_UNTIL', 'THINK_UNTIL'):
        monkeypatch.setattr(ml.S, name, getattr(ml.S, name))
    for name, value in {
        'BASE_URL': 'https://test.invalid/v1', 'LLM_KEY': 'test-only', 'EMB_KEY': 'test-only', 'RERANK_KEY': 'test-only',
        'LLM_MODEL': 'test-chat', 'EMB_MODEL': 'test-embedding', 'RERANKER_MODEL': 'test-reranker',
    }.items():
        monkeypatch.setattr(ml.S, name, value)
    (tmp_path / '22-askue.md').write_text('# Учёт электроэнергии\n\nТЕСТОВЫЙ_ДОКУМЕНТ: система учёта помогает анализировать потребление.', encoding='utf-8')
    (tmp_path / '23-solar.md').write_text('# Солнце\n\nПанели на кровле здания.', encoding='utf-8')
    (tmp_path / 'other.md').write_text('# Документ\n\nДополнительные сведения о погоде.', encoding='utf-8')
    calls = []
    failures = set()

    def reply(request):
        body = json.loads(request.content)
        calls.append((request.url.path, body))
        assert request.url.host == 'test.invalid'
        if request.url.path in failures:
            return httpx.Response(401, json={'error': 'synthetic authorization failure'})
        if request.url.path == '/v1/embeddings':
            values = body['input']
            values = [values] if isinstance(values, str) else values
            return httpx.Response(200, json={'data': [{'index': i, 'embedding': [1.0, 2.0, 3.0, 4.0]} for i, _ in enumerate(values)]})
        if request.url.path == '/v1/rerank':
            return httpx.Response(200, json={'results': [{'index': i, 'relevance_score': 1 / (i + 1)} for i in range(min(len(body['documents']), body['top_n']))]})
        assert request.url.path == '/v1/chat/completions'
        assert body['model'] == 'test-chat'
        assert ml.S.SITE_LEAD_RULE in body['messages'][0]['content']
        assert ml.S.LEAD_RULE not in body['messages'][0]['content']
        answer = 'Ответ исходной цепочки ML.'
        for i in range(4):
            if f'CHAT_{i}' in body['messages'][-1]['content']:
                answer = f'Ответ CHAT_{i}'
        return httpx.Response(200, json={'choices': [{'message': {'content': answer}, 'finish_reason': 'stop'}]})

    original_init = httpx.Client.__init__
    def transport_init(self, *args, **kwargs):
        if kwargs.get('transport') is None:
            kwargs['transport'] = httpx.MockTransport(reply)
        original_init(self, *args, **kwargs)
    monkeypatch.setattr(httpx.Client, '__init__', transport_init)
    adapter = RAGAdapter(ml.generate_answer, lambda: ml.init(kb_path=tmp_path), health_function=ml.health)
    return ml, adapter, calls, failures


def test_actual_ml_pipeline_behind_fastapi(ml_runtime):
    ml, adapter, calls, _ = ml_runtime
    with TestClient(create_app(adapter)) as client:
        response = client.post('/api/v1/chat', json={'session_id': 'integration', 'message': 'Нужен учёт электроэнергии'})
        assert response.status_code == 200
        assert response.json()['answer'] == 'Ответ исходной цепочки ML.'
        assert '22-askue.md' in response.json()['sources']
        assert response.json()['degraded'] == []
        assert response.json()['timings_ms']['total'] >= 0
    assert sum(path == '/v1/chat/completions' for path, _ in calls) == 1
    assert any(path == '/v1/embeddings' for path, _ in calls)
    assert any(path == '/v1/rerank' for path, _ in calls)
    assert ml.S.STRICT is True


def test_degradation_is_exposed_at_startup_and_per_request(ml_runtime):
    _, adapter, _, failures = ml_runtime
    failures.update(('/v1/embeddings', '/v1/rerank'))
    with TestClient(create_app(adapter)) as client:
        ready = client.get('/ready').json()
        assert ready['status'] == 'degraded'
        assert 'embeddings_unavailable' in ready['ml']['degraded']
        response = client.post('/api/v1/chat', json={'session_id': 'degraded', 'message': 'Нужен учёт электроэнергии'})
        assert response.status_code == 200
        assert 'reranker_failed' in response.json()['degraded']


def test_generation_failure_is_http_503_not_fake_answer(ml_runtime):
    _, adapter, _, failures = ml_runtime
    failures.add('/v1/chat/completions')
    with TestClient(create_app(adapter)) as client:
        response = client.post('/api/v1/chat', json={'session_id': 'error', 'message': 'Нужен учёт'})
        assert response.status_code == 503
        assert response.json() == {'detail': {'code': 'ml_unavailable'}}


def test_history_keeps_parameters_but_excludes_them_from_search(ml_runtime):
    ml, _, _, _ = ml_runtime
    params = {'role': 'user', 'content': 'Параметры объекта из карточки клиента: Москва'}
    history = [params] + [{'role': 'user', 'content': f'Реплика {i}'} for i in range(20)]
    generation, retrieval = ml._split('Текущий вопрос', history)
    assert generation[0][1] == params['content']
    assert len(generation) == 14
    assert len(retrieval) == 7
    assert not any('Параметры объекта' in content for _, content in retrieval)
    assert generation[-1] == ('user', 'Текущий вопрос')


def test_parallel_ml_calls_do_not_mix_answers(ml_runtime):
    ml, adapter, _, _ = ml_runtime
    with TestClient(create_app(adapter)):
        with ThreadPoolExecutor(max_workers=4) as pool:
            answers = list(pool.map(lambda i: ml.generate_answer(f'Учёт CHAT_{i}', history=[])['answer'], range(4)))
    assert answers == [f'Ответ CHAT_{i}' for i in range(4)]


def test_expired_ml_budget_does_not_start_another_http_request(ml_runtime):
    ml, _, calls, _ = ml_runtime
    ml.S._local.deadline = 0
    try:
        with httpx.Client() as client:
            with pytest.raises(TimeoutError, match='budget'):
                ml.S.post(client, '/chat/completions', 'test-only', {})
        assert not calls
    finally:
        ml.S._local.deadline = None
