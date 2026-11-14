"""Check configured API capabilities with synthetic inputs; never print credentials."""
import json
import os
from pathlib import Path

import httpx
from dotenv import load_dotenv


def probe() -> dict:
    load_dotenv(Path(__file__).resolve().parents[1] / '.env', override=False)
    base = os.getenv('LLM_BASE_URL', '').rstrip('/')
    key = os.getenv('LLM_API_KEY', '')
    if not base or not key:
        raise ValueError('Set LLM_BASE_URL and LLM_API_KEY')
    configs = [
        ('embeddings', '/embeddings', os.getenv('EMBEDDER_API_KEY') or key,
         {'model': os.getenv('EMB_MODEL') or 'baai/bge-m3', 'input': ['Тест учёта электроэнергии']}),
        ('reranker', '/rerank', os.getenv('RERANKER_API_KEY') or key,
         {'model': os.getenv('RERANKER_MODEL') or 'qwen/qwen3-reranker-8b', 'query': 'учёт', 'documents': ['Учёт электроэнергии'], 'top_n': 1}),
        ('generation', '/chat/completions', key,
         {'model': os.getenv('LLM_MODEL', ''), 'messages': [{'role': 'user', 'content': 'Ответь одним словом: готово'}], 'max_tokens': 64}),
    ]
    results = []
    with httpx.Client(timeout=10.0, follow_redirects=False) as client:
        for kind, endpoint, api_key, body in configs:
            item = {'component': kind, 'model': body['model']}
            try:
                response = client.post(base + endpoint, headers={'Authorization': 'Bearer ' + api_key}, json=body)
                item['http_status'] = response.status_code
                response.raise_for_status()
                data = response.json()
                if kind == 'embeddings':
                    valid = bool(data['data'][0]['embedding'])
                elif kind == 'reranker':
                    valid = bool(data.get('results') or data.get('data'))
                else:
                    valid = bool(data['choices'][0]['message'].get('content'))
                    item['served_model'] = data.get('model')
                item['ok'] = valid
            except Exception as exc:
                item.update(ok=False, error_type=type(exc).__name__)
            results.append(item)
            print(json.dumps(item, ensure_ascii=False), flush=True)
    report = {'all_available': all(item['ok'] for item in results), 'checks': results}
    output = Path(__file__).resolve().parents[1] / 'dist' / 'model_probe.json'
    output.parent.mkdir(exist_ok=True)
    output.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding='utf-8')
    return report


if __name__ == '__main__':
    raise SystemExit(0 if probe()['all_available'] else 1)
