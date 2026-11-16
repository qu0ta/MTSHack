# MTS EnergyTool — ИИ-консультант для бизнеса

Команда **AIblanki**, трек AI от МТС, Hack&Change 2026.

Репозиторий: https://github.com/qu0ta/MTSHack

## Какую задачу решает продукт

Клиенту бизнеса нужно понять, как снизить расходы на электроэнергию, автоматизировать учёт энергоресурсов или разобраться с потерями. Для выбора решения важны параметры объекта, оборудование и ограничения продуктов.

Консультант на сайте EnergyTool принимает вопрос обычным языком, ищет информацию в базе знаний и объясняет подходящие направления. Он учитывает историю диалога, задаёт уточняющие вопросы и сообщает, если данных недостаточно. Когда клиент готов продолжить, интерфейс собирает имя, телефон и компанию, а backend сохраняет заявку с резюме.

Основные направления консультаций: учёт электроэнергии и АСКУЭ, оптимизация расходов и оптовый рынок, коммерческие потери, солнечная генерация.

## Что реализовано

- React-интерфейс с чатом, стартовыми подсказками и формой заявки.
- Настоящие HTTP-запросы из frontend в Go API.
- RAG по предоставленной базе знаний: BM25, векторный поиск и reranker при доступности моделей у провайдера.
- Ответы с перечнем документов-источников и временем генерации в панели отладки.
- История сообщений и заявки в PostgreSQL, восстановление диалога после обновления страницы.
- Уточнение потребностей и явное подтверждение создания заявки.
- Проверка входных данных, обработка ошибок и защита от параллельной отправки в один чат.
- Docker Compose для backend, AI и PostgreSQL с автоматическими миграциями.

## Архитектура

```text
Браузер: React + TypeScript + Vite
    |
    | HTTP /api через Vite proxy
    v
Go API :8080
    |-- PostgreSQL: чаты, история, карточка клиента, заявки
    |
    | POST /api/v1/chat
    v
FastAPI :8000 + RAG
    |-- База знаний EnergyTool
    |-- BM25 и дополнительные модели поиска
    |
    v
OpenAI-совместимый API провайдера LLM
```

Go хранит состояние, формирует историю для AI и сохраняет результаты. FastAPI организует вызов RAG. Веса LLM в репозиторий и образ приложения не входят: генерация выполняется через настроенный API.

## Что потребуется

- Git для клонирования.
- Docker Desktop с работающим Linux engine и Docker Compose v2.
- Node.js и npm для frontend. Рекомендуется Node.js 22.12+.
- Доступ в интернет для сборки, скачивания базы знаний и запросов к API.
- Действующий адрес API, ключ и имя доступной модели генерации.

Устанавливать Go, Python или PostgreSQL на компьютер для запуска через Docker не нужно.

## Полный запуск

Команды ниже рассчитаны на PowerShell. Выполняйте их из корня репозитория, если не указано иное.

### 1. Скачать проект

```powershell
git clone https://github.com/qu0ta/MTSHack.git
cd MTSHack
```

Если проект уже скачан, откройте его корневую папку, содержащую compose.yaml.

### 2. Создать настройки

Если локальных файлов ещё нет:

```powershell
Copy-Item .env.example .env
Copy-Item ai/.env.example ai/.env
```

Не перезаписывайте заполненные .env при повторном запуске.

Корневой .env содержит настройки Compose. Для стандартного локального запуска достаточно:

```dotenv
APP_ENV=local
BACKEND_PORT=8080
ML_PORT=8000
POSTGRES_PORT=5433
POSTGRES_DB=mtshack
POSTGRES_USER=mtshack
POSTGRES_PASSWORD=mtshack
```

Внутри Compose адреса PostgreSQL и AI задаются автоматически через имена сервисов. Значения ML_SERVICE_URL и DATABASE_URL из шаблона для запуска вручную менять в Compose не требуется.

В ai/.env заполните обязательные настройки:

```dotenv
LLM_BASE_URL=https://openrouter.ai/api/v1
LLM_API_KEY=YOUR_API_KEY
LLM_MODEL=YOUR_AVAILABLE_MODEL

EMB_MODEL=baai/bge-m3
RERANKER_MODEL=qwen/qwen3-reranker-8b

ML_MODULE=app.ml.rag_api
TEMPERATURE=0
MAX_TOKENS=1500
AI_REQUEST_TIMEOUT_SECONDS=55
AI_MAX_CONCURRENCY=2
RAG_BUDGET_SECONDS=45
RAG_MAX_HISTORY=12
RAG_RETRIEVAL_TURNS=6
```

OpenRouter здесь служит примером OpenAI-совместимого провайдера. Укажите точный идентификатор модели, доступной вашему ключу. Для другого провайдера замените LLM_BASE_URL, ключ и имена моделей. Не вставляйте Markdown-разметку в значения URL.

EMBEDDER_API_KEY и RERANKER_API_KEY необязательны: без них используется LLM_API_KEY. Текущая реализация обращается ко всем моделям по общему LLM_BASE_URL. Если embeddings или reranker недоступны, RAG использует BM25 и отражает состояние в /ready. Для генерации нужен рабочий LLM API.

При автоматическом сабмите платформа передаёт свои API-настройки. При локальном запуске Compose читает ваши значения из ai/.env.

### 3. Запустить серверную часть

Откройте Docker Desktop и дождитесь запуска engine, затем выполните:

```powershell
docker compose up -d --build
docker compose ps
```

При первой сборке Docker автоматически скачивает официальный knowledge-base.zip и проверяет SHA-256. Отдельная локальная папка базы и переменная KB_SOURCE не требуются. Путь KB_PATH=/opt/mtshack/knowledge-base.zip задаётся в Compose.

Compose создаёт:

| Сервис | Назначение | Адрес с компьютера |
|---|---|---|
| backend | Go API | http://localhost:8080 |
| ml-service | FastAPI и RAG | http://localhost:8000 |
| postgres | PostgreSQL 16 | localhost:5433 |
| migrate | Применение SQL-миграций | Однократная задача без порта |

Дождитесь healthy у постоянных сервисов. migrate после успешного выполнения завершается с кодом 0 — это нормальное поведение.

Проверка готовности:

```powershell
Invoke-RestMethod http://localhost:8080/health
Invoke-RestMethod http://localhost:8080/ready
Invoke-RestMethod http://localhost:8000/ready
```

Go /ready проверяет соединение с БД. AI /ready показывает состояние индекса и доступность дополнительных поисковых компонентов. Полную цепочку проверяет отправка вопроса из браузера.

### 4. Запустить frontend

В новом терминале из корня проекта:

```powershell
cd frontend
npm install
npm run dev
```

Откройте адрес, который напечатает Vite, обычно:

**http://localhost:5173/assistant**

В текущем vite.config.ts запросы /api, /health и /ready проксируются на http://localhost:8080. Если изменили BACKEND_PORT, измените target в frontend/vite.config.ts.

Сервер frontend работает в этом терминале. Для остановки нажмите Ctrl+C. Серверную часть продолжает обслуживать Docker.

## Быстрая проверка продукта

1. Откройте интерфейс и задайте вопрос: «У нас три склада и 40 счётчиков. Хотим автоматизировать сбор показаний. Что нужно уточнить?»
2. Дождитесь ответа. Консультант уточняет оборудование и текущий способ сбора данных.
3. Откройте «Режим отладки», чтобы посмотреть документы-источники и время ответа.
4. Задайте уточняющий вопрос в том же диалоге.
5. Обновите страницу: сообщения восстановятся с сервера.
6. Нажмите «Заявка менеджеру», заполните имя, телефон, компанию и подтвердите согласие.
7. Отправьте форму и дождитесь сообщения «Заявка сохранена».

Заявка хранится в PostgreSQL с привязкой к чату. В текущем прототипе подтверждение означает сохранение в системе.

## Проверка API без frontend

В Postman импортируйте файлы из postman/:
- MTSHack.postman_collection.json
- MTSHack.local.postman_environment.json

Выберите локальное окружение с base_url=http://localhost:8080. Запускайте коллекцию по порядку: она создаёт чат и сохраняет chat_id.

Основные ручки:

| Метод | Путь | Назначение |
|---|---|---|
| POST | /api/v1/chats | Создать чат |
| GET | /api/v1/chats/{id} | Получить чат |
| GET | /api/v1/chats | Список чатов |
| PATCH | /api/v1/chats/{id} | Изменить заголовок |
| POST | /api/v1/chats/{id}/close | Закрыть чат |
| POST | /api/v1/chats/{id}/messages | Получить ответ AI / сохранить заявку |
| GET | /api/v1/chats/{id}/messages | Получить историю |
| GET | /api/v1/chats/{id}/application | Получить заявку |

Пример тела запроса сообщения:

```json
{
  "content": "Как автоматизировать сбор показаний на нескольких складах?",
  "client_data": {},
  "lead_requested": false
}
```

Для заявки используйте ту же ручку с lead_requested=true и client_data.contact_name, phone, company_name. Историю frontend не передаёт: Go загружает её из БД.

## Управление запуском

Логи и состояние:

```powershell
docker compose ps
docker compose logs -f backend ml-service
```

После изменения ai/.env пересоздайте AI-контейнер:

```powershell
docker compose up -d --force-recreate ml-service
```

После изменения исходников серверной части:

```powershell
docker compose up -d --build
```

Остановка серверной части с сохранением БД:

```powershell
docker compose down
```

PostgreSQL сохраняет данные в именованном volume. Для обычной остановки не добавляйте -v.

## Проверки для разработчиков

Frontend:

```powershell
cd frontend
npm test
npm run lint
npm run build
```

Backend, если Go установлен локально:

```powershell
cd backend
go test ./...
go vet ./...
```

AI, в отдельном Python-окружении с зависимостями ai/requirements.txt:

```powershell
cd ai
python -m pytest -q
```

Тесты AI лежат в ai/tests и не копируются в production-образ Docker.

## Если запуск требует внимания

- Missing settings: заполните LLM_BASE_URL, LLM_API_KEY, LLM_MODEL в ai/.env и пересоздайте контейнер.
- 401/403 при генерации: проверьте доступ ключа и доступность выбранной модели у провайдера.
- Ошибка SHA-256 базы: проверьте ссылку и контрольную сумму в ai/Dockerfile.
- Браузер не получает ответ: посмотрите логи backend и ml-service, проверьте адрес proxy.
- После сетевого обрыва нажмите «Обновить историю» перед повтором, чтобы сверить сохранённый результат.

## Структура репозитория

```text
backend/       Go API, бизнес-логика, SQL-миграции и тесты
ai/            FastAPI, интегрированный RAG, тесты и конфигурация AI
frontend/      React + TypeScript + Vite
rag_module/    Дополнительные инструменты и интерфейс ML-модуля
solution.py    Скрипт пакетной обработки для автоматического сабмита
postman/       Коллекция запросов и локальное окружение
compose.yaml   Запуск серверных сервисов
Dockerfile     Образ Go backend
```

## Настройки и данные

Заполненные .env и API-ключи не публикуются в Git. В браузере ключи LLM не используются: frontend обращается к Go. База знаний загружается из официальных материалов во время сборки.

Репозиторий: https://github.com/qu0ta/MTSHack

