# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run the server locally
go run ./cmd/server

# Build
go build -o server ./cmd/server

# Run tests
go test ./...

# Run a single test
go test ./internal/handlers/... -run TestFunctionName

# Run with Docker Compose (includes Postgres)
docker compose up --build
```

## Environment Variables

| Variable | Description |
|---|---|
| `PORT` | Server port (default implied: 8004) |
| `DATABASE_URL` | Postgres DSN |
| `SHARED_JWT_SECRET` | HS256 secret shared across all services for JWT validation |
| `INTERNAL_SECRET` | Value required in `X-Internal-Secret` header for internal routes |
| `PROFILE_SERVICE_URL` | Base URL for Profile Service (default: `http://profile:8002`) |

## Architecture

This is the **Payment Service** (`:8004`) in the TransFans microservices platform. It is one of four services (Auth `:8001`, Profile `:8002`, Content `:8003`, Payment `:8004`).

**Entry point:** `cmd/server/main.go` — registers all HTTP routes on a `net/http` ServeMux, no framework.

**Routes implemented:**
- `POST /checkout` — fetch tier from Profile Service, record transaction, create subscription on Profile Service
- `GET /transactions` — fan's transaction history
- `GET /balance` — creator's available/earned/paid balance
- `POST /payouts` — creator requests payout (always `completed` immediately)
- `GET /payouts` — creator's payout history
- `GET /revenue` — creator's revenue breakdown by tier (supports `from`/`to` query params)

**Helpers (`internal/handlers/helpers.go`):**
- `readJSON` — decodes request body with 1MB limit, rejects multiple JSON objects
- `WriteJSON` — writes JSON response with correct Content-Type
- `WriteError` — writes `{ "error": { "code", "message", "request_id" } }` shape

**Key checkout flow:** `POST /checkout` → call `GET /internal/tiers/{id}` on Profile Service → record transaction as `success` → call `POST /internal/subscriptions` on Profile Service → return 201. No real payment processing — always succeeds.

**Auth:** JWT HS256 validated locally via `SHARED_JWT_SECRET`. Internal routes called by this service use `X-Internal-Secret` header.

**Database:** PostgreSQL via `DATABASE_URL`. No ORM — plain Go `database/sql`.

## Agreed Stack

| Concern | Library |
|---|---|
| Routing + middleware | [chi](https://github.com/go-chi/chi) |
| Postgres driver | [pgx](https://github.com/jackc/pgx) |
| Type-safe SQL | [sqlc](https://sqlc.dev/) |
| Migrations | [goose](https://github.com/pressly/goose) |
| JWT validation | [golang-jwt/jwt](https://github.com/golang-jwt/jwt) |
| Config (env) | [caarlos0/env](https://github.com/caarlos0/env) |
| Logging | `log/slog` (stdlib) |

Migration files go in `internal/db/migrations/`. Run via embedded goose in the binary on startup, or with the goose CLI against `DATABASE_URL`.

SQL queries are defined in `internal/db/queries/` and generated into `internal/db/` by sqlc (`sqlc generate`).
