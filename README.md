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

Infrastructure connection strings and the JWT signing key live in `.env` (see `.env.example`). Stripe keys, webhook secrets, and feature flags live in the `app_settings` table and are refreshed every 30 seconds. See [docs/architecture.md](docs/architecture.md).

## Makefile targets

| Target | Description |
|--------|-------------|
| `make deps` | Download modules |
| `make test` | Run unit tests |
| `make build` | Build `bin/api` and `bin/worker` |
| `make docker-up` | Start full stack |
| `make docker-down` | Tear down stack |

## Branching

- `main` — stable releases
- `dev` — integration branch

Use [Conventional Commits](https://www.conventionalcommits.org/) for all changes.

## Live demo

**URL:** _Pending deployment_ — [GitHub](https://github.com/koro-manoj/koro-go-services)
