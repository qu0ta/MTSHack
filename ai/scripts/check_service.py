"""Real web acceptance examples; uses .env, writes only local reports, no leaderboard calls."""
import json
import argparse
import os
from datetime import datetime
from pathlib import Path
from time import perf_counter

from dotenv import load_dotenv
from fastapi.testclient import TestClient

AI_DIR = Path(__file__).resolve().parents[1]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--scenario', choices=('recommendation', 'clarification', 'no_information'))
    args = parser.parse_args()
    load_dotenv(AI_DIR / '.env', override=False)
    # Explicit defaults from the delivered ML package, reported below.
    os.environ.setdefault('EMB_MODEL', 'baai/bge-m3')
    os.environ.setdefault('RERANKER_MODEL', 'qwen/qwen3-reranker-8b')
    from app.main import create_app
    output = AI_DIR / 'dist' / ('service-' + datetime.now().strftime('%Y%m%d-%H%M%S') + '.json')
    output.parent.mkdir(exist_ok=True)
    report = {'models': {name: os.environ.get(name) for name in ('LLM_MODEL', 'EMB_MODEL', 'RERANKER_MODEL')}, 'cases': []}
    def save():
        output.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding='utf-8')
    started = perf_counter()
    try:
        with TestClient(create_app()) as client:
            report['startup_seconds'] = round(perf_counter() - started, 2)
            report['readiness'] = client.get('/ready').json()
            print('Готовность:', json.dumps(report['readiness'], ensure_ascii=False), flush=True)
            cases = [
                ('recommendation', 'У нас завод, присоединённая мощность 900 кВт. Хотим выйти на оптовый рынок. Какой вариант подходит?'),
                ('clarification', 'Завод потребляет 1 200 000 кВт·ч в месяц. Этого достаточно для допуска на оптовый рынок?'),
                ('no_information', 'Есть ли у EnergyTool готовая интеграция с нашей внутренней ERP «СкладПоток-X17»? Нужна подтверждённая совместимость, а не предположение.'),
            ]
            cases = [case for case in cases if args.scenario is None or case[0] == args.scenario]
            for name, question in cases:
                response = client.post('/api/v1/chat', json={'session_id': name, 'message': question, 'history': [], 'client_data': {}})
                result = {'scenario': name, 'question': question, 'http_status': response.status_code, 'response': response.json()}
                report['cases'].append(result)
                save()
                print(json.dumps(result, ensure_ascii=False), flush=True)
            lead_response = client.post('/api/v1/chat', json={
                'session_id': 'lead-acceptance', 'message': 'Свяжитесь со мной', 'lead_requested': True,
                'history': [{'role': 'user', 'content': 'Нужно наладить учёт электроэнергии на складе.'}],
                'client_data': {'contact_name': 'Тестовый клиент', 'phone': '+70000000000', 'company_name': 'Тестовая компания', 'region': 'Москва'},
            })
            report['lead'] = {'http_status': lead_response.status_code, 'response': lead_response.json()}
    except Exception as exc:
        report['error_type'] = type(exc).__name__
        print('Ошибка запуска:', type(exc).__name__, flush=True)
    save()
    print('Отчёт:', output, flush=True)
    return 0 if len(report['cases']) == (1 if args.scenario else 3) and all(case['http_status'] == 200 for case in report['cases']) else 1


if __name__ == '__main__':
    raise SystemExit(main())
