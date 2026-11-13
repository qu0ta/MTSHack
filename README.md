# MTSHack

AI-консультант MTS EnergyTool. В текущей ветке находятся Go API, PostgreSQL и локальный контрактный mock ML-сервиса.

## Запуск в Docker

```bash
docker compose up --build --wait
```

Compose запускает:

- `backend` — Go API на `http://localhost:8080`;
- `postgres` — PostgreSQL на `localhost:5433`;
- `migrate` — однократное применение SQL-миграций;
- `ml-service` — WireMock на `http://localhost:8000`, совместимый с текущим Go-клиентом.

Проверки после запуска:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
curl http://localhost:8000/health
```

Посмотреть состояние и логи:

```bash
docker compose ps
docker compose logs -f backend ml-service
```

Остановить стек:

```bash
docker compose down
```

Удалить также локальные данные PostgreSQL:

```bash
docker compose down -v
```

## Настройки

Compose автоматически читает `.env` в корне. Для локальных значений скопируйте `.env.example` в `.env` и измените нужные параметры:

```dotenv
BACKEND_PORT=8080
ML_PORT=8000
POSTGRES_PORT=5433
POSTGRES_DB=mtshack
POSTGRES_USER=mtshack
POSTGRES_PASSWORD=mtshack
```

Файл `.env` не должен попадать в Git.

## Контракт Go → ML

Backend отправляет `POST /api/v1/chat` на `http://ml-service:8000`:

```json
{
  "session_id": "chat-uuid",
  "message": "Текст сообщения",
  "history": [
    {"role": "user", "content": "Предыдущее сообщение"},
    {"role": "assistant", "content": "Предыдущий ответ"}
  ],
  "client_data": {}
}
```

Ожидаемый ответ:

```json
{
  "answer": "Ответ консультанта",
  "client_data": {},
  "sources": [],
  "lead": null,
  "response_time_ms": 1
}
```

Mock нужен только для полностью воспроизводимого локального запуска backend-ветки. После добавления FastAPI-сервиса его контейнер должен сохранить имя `ml-service`, внутренний порт `8000`, health endpoint `/health` и контракт `/api/v1/chat`. Go-код и `ML_SERVICE_URL` при этом менять не потребуется.

## Локальная проверка Go

```bash
cd backend
go test ./...
go vet ./...
```
