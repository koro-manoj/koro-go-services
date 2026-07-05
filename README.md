# Koro Go Services

Modular Go microservices: HTTP API with JWT auth and a Redis-backed webhook worker. Application secrets and third-party API keys are stored in PostgreSQL and loaded at runtime — not in environment variables.

## Stack

| Component | Role |
|-----------|------|
| `cmd/api` | HTTP API — health, auth, webhook ingestion |
| `cmd/worker` | Redis queue consumer for async webhook processing |
| PostgreSQL | Primary data store + `app_settings` config table |
| Redis | Job queue (`webhooks:incoming`) |

## Modules

- `internal/auth` — JWT issue/validate, HTTP middleware
- `internal/config` — bootstrap env + DB-backed settings cache
- `internal/webhooks` — ingest handler, Redis queue, processor

## Prerequisites

- Go 1.22+
- Docker & Docker Compose (recommended)

## Quick start

```bash
cp .env.example .env
make docker-up
```

API: http://localhost:8080

### Health check

```bash
curl -s http://localhost:8080/health | jq
```

### Issue a JWT

```bash
curl -s -X POST http://localhost:8080/auth/token \
  -H 'Content-Type: application/json' \
  -d '{"subject":"demo","role":"operator"}' | jq
```

### Protected route

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/auth/token \
  -H 'Content-Type: application/json' \
  -d '{"subject":"demo","role":"operator"}' | jq -r .access_token)

curl -s http://localhost:8080/me -H "Authorization: Bearer $TOKEN" | jq
```

### Enqueue a webhook

```bash
curl -s -X POST http://localhost:8080/webhooks/stripe \
  -H 'Content-Type: application/json' \
  -H 'X-Webhook-Event: payment.succeeded' \
  -d '{"amount":1000}' | jq
```

Worker logs show processing output in `docker compose logs -f worker`.

### Portal API (koro-web-apps live mode)

Demo credentials are seeded via `migrations/002_portal_demo.sql` (`demo@koro.io` / `password`).

```bash
curl -s -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@koro.io","password":"password"}' | jq

TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"demo@koro.io","password":"password"}' | jq -r '.data.token')

curl -s http://localhost:8080/api/auth/me -H "Authorization: Bearer $TOKEN" | jq
curl -s http://localhost:8080/api/dashboard/overview -H "Authorization: Bearer $TOKEN" | jq
```

Set `NEXT_PUBLIC_API_URL=http://localhost:8080` and `NEXT_PUBLIC_USE_MOCK_API=false` in the portal app to use live mode.

## Local development (without Docker)

```bash
# Start Postgres + Redis yourself, then:
export $(grep -v '^#' .env | xargs)
make run-api    # terminal 1
make run-worker # terminal 2
```

Apply migrations:

```bash
psql "$DATABASE_URL" -f migrations/001_initial.sql
```

## Configuration model

Infrastructure goes in `.env`. Stripe keys and webhook secrets live in `app_settings` (Postgres), refreshed every 30s.

## Makefile targets

| Target | Description |
|--------|-------------|
| `make deps` | Download modules |
| `make test` | Run unit tests |
| `make build` | Build `bin/api` and `bin/worker` |
| `make docker-up` | Start full stack |
| `make docker-down` | Tear down stack |

## Live demo

Pending deployment.
