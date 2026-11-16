# MTSHack · EnergyTool AI-консультант

**EnergyTool AI-консультант** — веб-прототип для бизнеса, который помогает сформулировать задачу в области электроэнергетики, получить консультацию по базе знаний и, при необходимости, сохранить заявку на общение со специалистом.

Разработка команды **AIblanki** для хакатона MTSHack.

**Репозиторий:** https://github.com/qu0ta/MTSHack

> **Статус (16 ноября 2026):** в `main` есть React-интерфейс, Go API, PostgreSQL, Python FastAPI и ML/RAG-код. Интеграция присутствует в исходниках, но полноценная сквозная работа на конкретной машине зависит от настроек AI и доступности внешних моделей. Наличие кода и тестов не равнозначно подтверждённому end-to-end запуску. CRM и отправка писем менеджерам в текущий прототип не входят.

## 1. Какую проблему решаем

Компании сталкиваются с вопросами об энергопотреблении, коммерческом учёте, автоматизации показаний, потерях, собственной генерации и выходе на оптовый рынок. Часто клиенту сложно быстро понять, какое решение подходит его объекту, какие параметры нужны для оценки и к кому передать запрос.

**Наш подход:** единая точка входа в виде чата. Пользователь описывает ситуацию своими словами; AI-консультант опирается на доступную базу знаний, учитывает историю диалога, задаёт уточняющие вопросы, показывает источники и предлагает следующий шаг. Если нужна персональная консультация, пользователь заполняет форму; Go сохраняет заявку и краткое резюме в PostgreSQL.

Продукт **не** гарантирует экономию в процентах, юридическую/техническую пригодность решения или фактическую передачу заявки в CRM. Эти эффекты требуют отдельной проверки и интеграций.

## 2. Что реализовано

- **Чат:** создание, чтение, список, переименование и закрытие диалогов; загрузка истории сообщений.
- **Консультация:** передача сообщения, предыдущих реплик и структурированных данных в Python ML-сервис; сохранение ответа и источников в PostgreSQL.
- **RAG:** Python FastAPI вызывает ML-пайплайн, использующий доступную базу знаний; при отказе embedding/reranker может работать в явно обозначенном упрощённом режиме BM25 + LLM.
- **Заявка эксперту:** форма имени, телефона, компании и согласия; подготовка `lead` только при `lead_requested: true`; сохранение с `UNIQUE(chat_id)` (одна актуальная заявка на чат).
- **Интерфейс:** React + TypeScript + Vite, быстрые темы для начала диалога, панель источников и времени ответа, восстановление последнего чата через `localStorage`.
- **Инфраструктура:** Docker Compose для `backend`, `ml-service`, `postgres`, `migrate`; фронтенд запускается отдельно через npm.

### Как проходит запрос

```text
Пользователь (React / Vite :5173)
    │ POST /api/v1/chats/{id}/messages
    ▼
Go API (:8080) ───── PostgreSQL 16 (в Docker :5432; на хосте :5433)
    │                    ├─ chats + client_data
    │                    ├─ messages + metadata.sources
    │                    └─ applications
    │ POST /api/v1/chat
    ▼
Python FastAPI (:8000) → RAG / база знаний / внешние модели
    │ answer + sources + client_data + lead
    ▼
Go: атомарно сохраняет данные, возвращает ответ в React
```

Go управляет сессиями и хранением; Python не сохраняет диалоги или заявки.

## 3. Быстрый запуск — Windows PowerShell

### Требования

- Docker Desktop с запущенным Docker Engine и поддержкой `docker compose`;
- Node.js **20.19+ или 22.12+**, npm;
- Git;
- доступ к модели генерации и корректные настройки для AI-сервиса;
- доступ к базе знаний при сборке AI-образа (см. ограничения ниже).

Для запуска всего backend-стека Python и Go на хосте **не нужны** — они собираются Docker. Для локальных тестов вне Docker понадобятся Go 1.26.10 и Python 3.11+.

**Шаг 1. Получить проект:**

```powershell
git clone https://github.com/qu0ta/MTSHack.git
cd MTSHack
```

**Шаг 2. Создать конфигурацию:**

```powershell
Copy-Item .env.example .env
Copy-Item ai\.env.example ai\.env
notepad .env
notepad ai\.env
```

В корневом `.env` можно оставить локальные порты и учётные данные из примера. В `ai/.env` необходимо указать действительные `LLM_BASE_URL`, `LLM_API_KEY`, `LLM_MODEL`, а также модели и ключи для embeddings/reranker (когда они отличаются от LLM). **Не коммитьте реальные ключи.** Значения в примерах не гарантируют доступность моделей у конкретного провайдера.

> Важно: Compose использует **`ai/.env`** как `env_file` для `ml-service`. Без этого файла запуск невозможен. Переменная `ML_SERVICE_URL` из корневого `.env.example` не управляет Docker-соединением: Compose задаёт бэкенду `http://ml-service:8000` напрямую.

**Шаг 3. Поднять PostgreSQL, миграции, Go и AI:**

```powershell
docker compose up -d --build --wait
```

При первом запуске сборка может быть долгой из-за загрузки зависимостей и индексации базы знаний. Если сервис не стал healthy, изучите логи из раздела «Диагностика». Контейнер AI не является WireMock-заглушкой: в актуальном `main` используется `ai/Dockerfile` с FastAPI.

**Шаг 4. Запустить фронтенд в отдельном терминале:**

```powershell
cd frontend
npm install
npm run dev
```

Откройте адрес Vite из терминала, обычно **http://localhost:5173**. Запросы `/api`, `/health`, `/ready` Vite перенаправляет на `http://localhost:8080` — вручную настраивать CORS для этого режима не требуется.

### Проверка здоровья сервиса

В новом PowerShell из корня `MTSHack`:

```powershell
docker compose ps -a
curl.exe http://localhost:8080/health
curl.exe http://localhost:8080/ready
curl.exe http://localhost:8000/health
curl.exe http://localhost:8000/ready
```

- Go `/health`: проверяет работу HTTP-сервера.
- Go `/ready`: проверяет доступность PostgreSQL, **не** модельного API.
- AI `/health`: отвечает, если FastAPI работает.
- AI `/ready`: показывает готовность индекса и возможное `status: degraded`. HTTP 200 с `degraded` **не** означает, что полный RAG и все внешние модели доступны.

### Остановка

```powershell
docker compose down
```

Данные PostgreSQL хранятся в именованном volume и при обычной остановке **сохраняются**. Удалить их без возможности восстановления:

```powershell
docker compose down -v
```

## 4. Режимы и порты

| Компонент | Адрес с компьютера | Роль |
|---|---|---|
| Frontend | `http://localhost:5173` | React/Vite, отдельный `npm run dev` |
| Go API | `http://localhost:8080` | Чаты, сообщения, заявки, БД |
| Python AI | `http://localhost:8000` | RAG, ответы, подготовка lead |
| PostgreSQL | `localhost:5433` | PostgreSQL 16; внутренний порт 5432 |
| `migrate` | отдельного порта нет | Одноразовое применение SQL-миграций |

Порты хоста задаются `BACKEND_PORT`, `ML_PORT` и `POSTGRES_PORT` в корневом `.env`; порт фронтенда Vite при необходимости выбирает автоматически.

## 5. Проверка API без фронтенда

Примеры ниже выполняются из **PowerShell** и не содержат фиктивного ID: сначала создаётся реальный чат.

```powershell
$chat = Invoke-RestMethod -Uri http://localhost:8080/api/v1/chats -Method Post -ContentType 'application/json' -Body '{}'
$chat.id
```

Затем можно отправить обычный вопрос (нужен работающий AI-сервис):

```powershell
$body = @{ content = 'У нас завод с присоединённой мощностью 900 кВт. Хотим разобраться с выходом на оптовый рынок.' } | ConvertTo-Json -Compress
$result = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/chats/$($chat.id)/messages" -Method Post -ContentType 'application/json; charset=utf-8' -Body ([System.Text.Encoding]::UTF8.GetBytes($body))
$result.assistant_message.content
```

Прочитать историю и заявку по этому же чату:

```powershell
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/chats/$($chat.id)/messages?limit=100&offset=0"
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/chats/$($chat.id)/application"
```

До создания заявки второй запрос ожидаемо возвращает **404**.

Чтобы проверить подтверждённую заявку через API, сначала обсудите задачу, затем используйте реальные тестовые данные и явный флаг `lead_requested` — аналог формы фронтенда. Не отправляйте чужие номера или персональные данные в демонстрации без согласия.

### Доступные маршруты Go

| Метод | Путь | Назначение |
|---|---|---|
| `GET` | `/health` | Проверка Go HTTP |
| `GET` | `/ready` | Проверка доступности PostgreSQL |
| `POST` | `/api/v1/chats` | Создать чат |
| `GET` | `/api/v1/chats` | Список чатов |
| `GET` | `/api/v1/chats/{id}` | Получить чат |
| `PATCH` | `/api/v1/chats/{id}` | Изменить заголовок |
| `POST` | `/api/v1/chats/{id}/close` | Закрыть чат |
| `GET` | `/api/v1/chats/{id}/messages` | История сообщений (пагинация) |
| `POST` | `/api/v1/chats/{id}/messages` | Вопрос → ML → сохранение ответа; заявка при подтверждении |
| `GET` | `/api/v1/chats/{id}/application` | Прочитать заявку чата |

Python использует `POST /api/v1/chat` и `GET /health`, `GET /ready` на порту 8000. Полная схема запроса/ответа — [`ai/API_CONTRACT.md`](ai/API_CONTRACT.md).

## 6. PostgreSQL: как открыть терминал

Из корня репозитория, где лежит `compose.yaml`:

```powershell
docker compose exec postgres psql -U mtshack -d mtshack
```

Имя Docker-контейнера генерируется Compose. Поэтому **не нужно** угадывать имена через `docker exec -it mtshack-postgres`: используйте `docker compose exec postgres`.

Внутри `psql`:

```sql
\dt
SELECT id, status, created_at FROM chats ORDER BY created_at DESC LIMIT 10;
SELECT role, content, created_at FROM messages ORDER BY created_at DESC LIMIT 10;
SELECT id, chat_id, full_name, company, status FROM applications ORDER BY created_at DESC LIMIT 10;
\q
```

Если терминал не принимает ввод, попробуйте обычный Windows PowerShell вместо встроенного терминала IDE. Для разовых команд интерактивный режим не нужен:

```powershell
docker compose exec -T postgres psql -U mtshack -d mtshack -c "SELECT count(*) FROM applications;"
```

## 7. Тесты и проверка кода

**Go** (если установлен локально Go 1.26.10):

```powershell
cd backend
go test ./...
go vet ./...
cd ..
```

**Frontend:**

```powershell
cd frontend
npm test
npm run lint
npm run build
cd ..
```

**Python AI** (Python 3.11+ и зависимости из `ai/requirements.txt`):

```powershell
cd ai
python -m pip install -r requirements.txt
python -m pytest -q
python -m scripts.probe_models
python -m scripts.check_service
cd ..
```

Последние две Python-команды требуют действующих API-ключей/базы знаний и обращаются к внешним моделям; офлайн-тесты и онлайн-проверка — не одно и то же.

## 8. Что происходит с заявкой

1. Пользователь открывает «Заявка менеджеру» в правой панели.
2. Заполняет имя, телефон, компанию и подтверждает согласие.
3. Frontend отправляет сообщение и `lead_requested: true` вместе с `client_data`.
4. Python проверяет заполненность полей и подготавливает `lead` с кратким резюме запроса.
5. Go в транзакции сохраняет заявку в `applications`, пользовательское сообщение и ответ в `messages`, а параметры карточки — в `chats.client_data`.
6. При повторном подтверждении выполняется **upsert** по `chat_id`: остаётся одна заявка на чат.
7. UI показывает «Заявка сохранена» только при фактическом ответе Go с объектом `application`.

**Важно:** заявка пока сохраняется в базе, но **не отправляется** письмом или в CRM автоматически. Наличие согласия в UI ещё не означает, что продукт готов к промышленной обработке персональных данных.

## 9. Диагностика

**Контейнер не найден:**

```powershell
docker compose ps -a
docker compose logs --tail=100 postgres backend ml-service migrate
```

**Ошибка `ai/.env` не найден:** создайте файл командой `Copy-Item ai\.env.example ai\.env` и заполните настройки моделей. Ключи в репозиторий не добавляйте.

**AI не ready или сбой при сборке:** `ai/Dockerfile` загружает ZIP базы знаний по зафиксированному URL и проверяет SHA-256. При недоступности архива или несовпадении контрольной суммы сборка завершится ошибкой. Для запуска нужна доступная корректная база знаний и настроенный поставщик моделей. Нельзя маскировать ошибку автоматической заменой на заглушку и считать её рабочим RAG.

**`status: degraded` или 403 от embeddings/reranker:** в документации AI от 13 ноября зафиксирован такой случай. В этом режиме допустима локальная проверка BM25 + LLM, но не следует говорить о работе полной поисковой цепочки. Проверить реальные доступы через `python -m scripts.probe_models`.

**Не получается отправить сообщение:** убедитесь, что `http://localhost:8080/ready` и `http://localhost:8000/ready` отвечают; затем откройте логи:

```powershell
docker compose logs -f backend ml-service
```

**Порт 5433 занят:** выберите свободный `POSTGRES_PORT` в `.env`. Сервисы внутри Docker всё равно подключаются к `postgres:5432`.

**После ошибки запроса неизвестно, был ли ответ сохранён:** используйте кнопку «Обновить историю» во фронтенде, прежде чем повторно отправлять сообщение.

## 10. Результаты и ограничения прототипа

Реализованы кодовые модули React, Go, PostgreSQL, FastAPI, SQL-миграции, тесты и сценарий заявки. Согласно `ai/README.md`, **13 ноября 2026** локально проверялись отдельные ML-сценарии: рекомендация по параметрам объекта, уточняющий вопрос и корректное признание отсутствия подтверждённой информации. Эти проверки выполнялись в **BM25 + LLM**; embeddings и reranker у использованного провайдера возвращали HTTP 403. Это **не** доказательство сквозной работоспособности всех контейнеров или бизнес-эффекта.

Перед промышленной эксплуатацией нужны: проверка полного RAG и end-to-end сценариев с PostgreSQL, интеграция CRM/уведомлений, учёт и хранение согласий, авторизация/контроль доступа, политика хранения персональных данных, эксплуатационный мониторинг и измерение качества консультаций.

## 11. Полезные файлы проекта

- [`compose.yaml`](compose.yaml) — сборка и запуск backend/AI/БД/миграций;
- [`frontend/README.md`](frontend/README.md) — запуск React;
- [`ai/README.md`](ai/README.md) — настройка Python/ML и результаты локальных проверок;
- [`ai/API_CONTRACT.md`](ai/API_CONTRACT.md) — контракт Go ↔ Python;
- [`backend/migrations/`](backend/migrations/) — SQL-таблицы и `client_data`;
- [`backend/cmd/main.go`](backend/cmd/main.go) — HTTP-маршруты Go.

---

**Команда AIblanki · MTSHack 2026**  
Репозиторий: https://github.com/qu0ta/MTSHack