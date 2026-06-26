# Subscriptions API (Effective Mobile test)

REST-сервис для учёта онлайн-подписок пользователей.

## Запуск

```bash
cp .env.example .env
docker compose up --build
```

API: http://localhost:8080  
Swagger: http://localhost:8080/swagger.yaml

Postgres снаружи: `localhost:5433`

## Примеры

Создать подписку:

```bash
curl -X POST http://localhost:8080/api/subscriptions \
  -H "Content-Type: application/json" \
  -d '{"service_name":"Yandex Plus","price":400,"user_id":"60601fee-2bf1-4721-ae6f-7636e79a0cba","start_date":"07-2025"}'
```

Список:

```bash
curl "http://localhost:8080/api/subscriptions?user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba"
```

Сумма за период:

```bash
curl "http://localhost:8080/api/subscriptions/total?from=07-2025&to=12-2025&user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba"
```

## Локально без Docker

```bash
go run ./cmd/api
```

Нужен Postgres и `DATABASE_URL` в `.env`.

## Стек

Go, chi, PostgreSQL, docker-compose, swagger.yaml
