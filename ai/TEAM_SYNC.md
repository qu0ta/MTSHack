# Сверка веток и запуск с общим backend

Проверено 13 ноября 2026 через GitHub API/raw. Ветки не скачивались через Git,
checkout/fetch/merge не выполнялись. Локальные Go/React-файлы не менялись.

## Проверенные версии

| Ветка | Commit | Вывод |
|---|---|---|
| ml-engineer | `8752b81fc7d1d5b6f1443fcba8083a6924c23555` | solution.py и rag_api.py побайтово совпадают с установленной site1 |
| backend | `1a553722a602c3d151e18fdee6c45cdf9e785ca2` | Версия не изменилась; клиент AI и сохранение заявки уже подключены |
| main | `63bce6807a2e01dbabca551302eae4e559688eb5` | Схемы Go совместимы; базовый Compose пока запускает WireMock |
| feature/fastapi-compose | `f56699ae69f530d15ead48272f17c4297f683ed1` | Добавлен overlay сборки FastAPI, но отсутствуют передача API-настроек и монтирование KB |
| feature/confirmed-lead | `496e55b248860426fd1847e9837e2bdc76cb5841` | Это AI-ветка, а не Go. lead_requested и nullable history уже поддержаны нашей версией; её оркестратор ещё stub |

Источники: [ML](https://github.com/qu0ta/MTSHack/tree/8752b81fc7d1d5b6f1443fcba8083a6924c23555),
[backend](https://github.com/qu0ta/MTSHack/tree/1a553722a602c3d151e18fdee6c45cdf9e785ca2),
[Compose main](https://github.com/qu0ta/MTSHack/blob/63bce6807a2e01dbabca551302eae4e559688eb5/compose.yaml),
[FastAPI overlay](https://github.com/qu0ta/MTSHack/blob/f56699ae69f530d15ead48272f17c4297f683ed1/compose.fastapi.yaml),
[ветка заявки](https://github.com/qu0ta/MTSHack/tree/496e55b248860426fd1847e9837e2bdc76cb5841).

Не заменять текущий оркестратор файлом из feature/confirmed-lead: это вернёт stub
вместо реального ML. Её важные изменения уже учтены: строгий lead_requested,
history=null, отказ от заявки при неполных контактах и ограничения числовых полей.
Добавлена совместимость пустых строк формы с незаполненными полями.

## Готовое дополнение Compose в ai/

`compose.integration.yaml` применяется поверх командного корневого `compose.yaml`.
Он сохраняет имя `ml-service`, порт 8000 и контракт Go. Обеспечивает:

- сборку из Dockerfile AI-сервиса;
- передачу локального env-файла в контейнер, без помещения секретов в образ;
- read-only подключение приватной KB вместо старого WireMock volume;
- фиксированный путь KB внутри контейнера — `/knowledge-base`;
- Python healthcheck на `/ready`, без wget, которого нет в python:slim;
- ожидание готовности ML перед запуском backend.

Нужен Docker Compose >= 2.24.4 (`!override`).
Команды выполнять **из рабочей копии общего проекта, где есть backend/ и compose.yaml**.
В AI-ветке этих файлов сейчас нет — запуск только из ai/ не создаст Go/БД.

```bash
export AI_SOURCE=/absolute/path/to/MTSHack/ai
export AI_ENV_FILE="$AI_SOURCE/.env"
export KB_SOURCE=/absolute/path/to/knowledge-base

docker compose -f compose.yaml -f "$AI_SOURCE/compose.integration.yaml" up --build --wait --wait-timeout 300
```

KB_SOURCE — существующая абсолютная папка Markdown или ZIP. Её не нужно копировать в
репозиторий. Если путь неверный, Compose не создаст вместо него пустую папку.
AI_ENV_FILE содержит LLM_BASE_URL/LLM_API_KEY/LLM_MODEL и, при необходимости,
EMB_MODEL/RERANKER_MODEL/отдельные ключи. Host KB_PATH из .env переопределяется
контейнерным `/knowledge-base`, остальные настройки сохраняются.

Не передавать полный вывод `docker compose config` в общий чат: при чтении настоящего
env он может содержать ключи. Для проверки синтаксиса без печати значений:

```bash
docker compose -f compose.yaml -f "$AI_SOURCE/compose.integration.yaml" config --quiet
curl http://localhost:8000/ready
curl http://localhost:8080/ready
```

Можно обойтись этим одним overlay без compose.fastapi.yaml из отдельной ветки.
Если команда уже использует тот overlay, наш должен идти последним; совместное
объединение тоже проверено, dockerfile_inline и настройки WireMock сбрасываются.

## Зависимости

Новый ML requirements содержит только httpx 0.28.1, numpy 2.2.6 и rank-bm25 0.2.2.
FastAPI/Pydantic/Uvicorn по-прежнему нужны нашему HTTP-сервису. NumPy закреплён 2.2.6
для Python <3.14 (в том числе Docker 3.11 и предлагаемый коллегой 3.13); для локального
Python 3.14 оставлена проверенная версия 2.3.3. Зависимости глобально не устанавливались.

## Что проверено и что ещё открыто

- `docker compose config` успешно объединил main + наш overlay.
- Успешно объединились feature base + feature FastAPI overlay + наш overlay.
- В проверках использовался пустой временный env: настоящие секреты не выводились.
- Проверены адрес Go → ML, сброс WireMock, env, read-only KB, build и readiness.
- Docker Compose доступен, но Docker daemon не запущен. Образы и весь стек не запускались.
- Предыдущие реальные проверки OpenRouter дали 403 для embeddings/reranker. Новая
  публикация ML содержит ту же пару файлов и не устраняет права доступа к внешним API.
- Удалённая AI-Engineer всё ещё на `505e4b0…`: актуальный рабочий код пока локальный.
  Перед запуском командой его нужно закоммитить/передать; push этой задачей не выполнялся.

После запуска стека проверить реальный диалог, историю, сохранение карточки и заявки,
повтор подтверждения заявки и ошибки зависимостей. Сборку контейнера и запись в БД
нельзя считать проверенными только по успешному объединению Compose.
