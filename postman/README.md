# Проверка MTSHack через Postman

## Подготовка

Из корня проекта запустите стек:

```powershell
docker compose up -d --build --wait
docker compose ps
```

## Импорт и запуск

В Postman нажмите `Import` и импортируйте:

1. `MTSHack.postman_collection.json`
2. `MTSHack.local.postman_environment.json`

Выберите environment `MTSHack Local Docker`. Откройте коллекцию `MTSHack API — Full Regression`, нажмите `Run` и запустите все запросы в исходном порядке с одной итерацией.

Коллекция сама создаёт чат и сохраняет `chat_id`. Она проверяет health/readiness, CRUD чата, путь Go → ML → PostgreSQL, сохранение истории, валидацию, `404`, закрытие чата и `409` после закрытия.

Ожидаемый результат — все тесты зелёные. Каждый полный запуск оставляет один закрытый тестовый чат в локальной БД.

Если backend работает не на `localhost:8080`, измените `base_url` в environment.

## Автоматический прогон без Postman GUI

Из корня проекта можно запустить официальный Newman-контейнер:

```powershell
docker run --rm --mount "type=bind,source=${PWD}\postman,target=/etc/newman,readonly" postman/newman:alpine run /etc/newman/MTSHack.postman_collection.json --environment /etc/newman/MTSHack.local.postman_environment.json --env-var base_url=http://host.docker.internal:8080 --reporters cli
```

Успешный прогон завершается с exit code `0`. На проверенной версии коллекции выполняются 14 запросов и 29 assertions.
