# Go → Python → ML: контракт интеграции

Проверено чтением GitHub без fetch/checkout 13 ноября 2026:

- [модели Go](https://github.com/qu0ta/MTSHack/blob/1a553722a602c3d151e18fdee6c45cdf9e785ca2/backend/internal/mlclient/models.go)
- [HTTP-клиент Go](https://github.com/qu0ta/MTSHack/blob/1a553722a602c3d151e18fdee6c45cdf9e785ca2/backend/internal/mlclient/client.go)
- [сервис консультаций](https://github.com/qu0ta/MTSHack/blob/1a553722a602c3d151e18fdee6c45cdf9e785ca2/backend/internal/consultations/service.go)
- [main.go](https://github.com/qu0ta/MTSHack/blob/1a553722a602c3d151e18fdee6c45cdf9e785ca2/backend/cmd/main.go)

В новой backend-версии подключён клиент AI, сохраняются client_data и заявки.
Схемы исполняемого Go-кода приоритетнее старого предложения в backend README.
Основные поля совпадают с Python. Сквозной запуск с PostgreSQL/Go ещё не выполнен.

## Консультация

`POST /api/v1/chat`, `Content-Type: application/json`:

```json
{
  "session_id": "UUID чата",
  "message": "А при 900 кВт?",
  "history": [
    {"role": "user", "content": "Мощность завода 500 кВт"},
    {"role": "assistant", "content": "При мощности ниже 750 кВт выход невозможен."}
  ],
  "client_data": {"connected_power_kw": 900, "region": "Москва"}
}
```

История в хронологическом порядке, БЕЗ текущего `message`. Python передаёт текущий
вопрос в `query` один раз. История хранится в Go, Python не хранит состояние сессий.
`history` может отсутствовать или быть `null` (nil slice Go), в обоих случаях это `[]`.
До 100 предыдущих сообщений, до 10 000 символов на реплику. Роли только user/assistant.
Пустой вопрос, неверная роль, отрицательные/бесконечные мощности — HTTP 422.

ClientData: необязательные `company_name`, `contact_name`, `phone`, `industry`, `region`,
`connected_power_kw` (кВт), `monthly_consumption_kwh` (кВт·ч/месяц); отсутствующие — null.
Не смешивать мощность и потребление. Значения возвращаются без извлечения из текста.

Ответ содержит: `answer: string`, `client_data: object`, `sources: string[]`,
`lead: object|null`, `response_time_ms: integer >= 0`.
Дополнительно: `degraded: string[]`, `timings_ms: object`. Текущий Go-декодер допускает
дополнительные поля, но пока не сохраняет их для UI. `degraded` описывает сбои поиска,
а `timings_ms` — времена retrieval/generation/total из ML.
Источники — имена документов, не URL. Пустые источники не означают сбой HTTP.

## Явная подготовка заявки — поддерживается текущим Go

Тот же запрос с дополнительным **boolean** `lead_requested: true`.
Отсутствие поля означает false и сохраняет совместимость с текущим Go-клиентом.
Go выставляет true только после подтверждения пользователем отправки заявки.
Не выводить этот флаг только из наличия контактов.

```json
{
  "session_id": "UUID чата",
  "message": "Прошу связаться со мной",
  "history": [{"role":"user","content":"Нужно наладить учёт на складе"}],
  "client_data": {
    "contact_name": "Анна", "phone": "+79991234567",
    "company_name": "ООО Пример", "region": "Москва"
  },
  "lead_requested": true
}
```

Тогда Python возвращает `lead` с `contact_name`, `phone`, `company_name`, `summary`.
Формирование локальное, без ещё одного вызова LLM. Если поля неполны — `lead: null`
и сообщение о недостающих данных. HTTP 200 означает обработку запроса, не сохранение заявки.
Go сохраняет объект с привязкой к session_id, защищает от повторов и только после успеха
сообщает пользователю, что заявка отправлена. В Python нет регистрации/CRM/БД.

## Ошибки

```json
{"detail": {"code": "ml_timeout"}}
```

| HTTP | code | Значение |
|---|---|---|
| 422 | стандартный detail FastAPI | Неверные данные запроса |
| 502 | invalid_ml_response | ML нарушил контракт ответа |
| 503 | not_ready / ml_unavailable | Не готов или ошибка ML |
| 503 | ml_busy | Все потоки заняты, Retry-After: 2 |
| 504 | ml_timeout | Истёк бюджет ожидания ответа |

При 504 вычисление в синхронном потоке может продолжаться. Не повторять автоматически
сообщение без ограничения числа попыток. В site1 сбой генерации приводит к исключению и HTTP 503; отсутствие сведений в KB
остаётся нормальным ответом. Недоступность поисковых моделей отражается в `degraded`.

## Что уже подключено в backend 1a55372

- `main.go` создаёт HTTP-клиент AI и регистрирует обработчик консультаций.
- Консультация отправляет актуальный `client_data` и `lead_requested` в Python.
- `lead` сохраняется через `applications.Upsert`, обновляется карточка клиента.
- HTTP-клиент ждёт 65 секунд, WriteTimeout сервера — 95 секунд.
- Проверяются обязательные поля ответа и коды 422/502/503/504.

Сообщение «веб-обработчик не вызывает ML» относилось к старому опубликованному
AI-коммиту. Текущий локальный `app/api/routes.py` вызывает оркестратор и ML.
Для проверки команде нужно получить именно эту версию `ai/` после коммита/объединения.

## Совместная приёмка

Go обращается по `ML_SERVICE_URL=http://ml-service:8000` в Docker или
`http://127.0.0.1:8000` при локальном запуске. Со стороны UI:
`POST /api/v1/chats/{id}/messages` с `content`, `client_data` и `lead_requested`.
Go сам преобразует `content` в `message` и подставляет историю.

Осталось запустить общий проект и проверить два чата, исправление карточки,
подтверждение заявки, повторную отправку (одна заявка на чат) и сбой API.
На локальном `127.0.0.1:8080` при проверке backend не был доступен, поэтому запись
в PostgreSQL и весь путь UI → Go → Python ещё не подтверждены запуском.
Это не требует изменений Go со стороны AI Engineer.

## Сверка новых интеграционных веток

Проверены main `63bce68`, ML `8752b81` и feature/fastapi-compose `f56699a`.
Схема Go не изменилась; nullable history и явное lead_requested совместимы.
Пустые строки необязательных полей нормализуются в null, чтобы неполная форма заявки
возвращала перечень недостающих данных вместо HTTP 422.
Запуск контейнеров и ограничения проверки описаны в [TEAM_SYNC.md](TEAM_SYNC.md).
