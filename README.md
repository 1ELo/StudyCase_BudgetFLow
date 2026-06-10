# BudgetFlow

A Go backend API for budget management with JWT authentication, RBAC authorization, ledger-safe transactions, and anti-double-spend guarantees.

## Tech Stack

- **Language:** Go 1.22+
- **Framework:** Gin (HTTP), GORM (ORM)
- **Database:** PostgreSQL
- **Auth:** JWT (HS256, 24h expiry)
- **Migration:** golang-migrate (file-based SQL)

## Setup (Docker Compose - Recommended)

The easiest way to run the entire stack (Database, Migrations, and Application) is using Docker Compose.

1. Copy `.env.example` to `.env`:
   ```bash
   cp .env.example .env
   ```

2. Start the stack:
   ```bash
   docker compose up --build -d
   ```
   *This will automatically start PostgreSQL, run the migrations, and start the Gin API server on port 8080.*

3. Seed default accounts:
   ```bash
   # Run the seeder against the running database container
   go run seeds/finance_seed.go
   ```

## Setup (Manual)

If you prefer to run the Go app locally instead of in Docker:

1. Start PostgreSQL (local or Docker):
   ```bash
   docker run -d --name budgetflow-db \
     -e POSTGRES_USER=postgres \
     -e POSTGRES_PASSWORD=postgres \
     -e POSTGRES_DB=budgetflow \
     -p 5432:5432 \
     postgres:16-alpine
   ```

2. Run migrations:
   ```bash
   make migrate-up
   ```

3. Seed default accounts:
   ```bash
   make seed
   ```

4. Start the server:
   ```bash
   make run
   ```

## API Base URL

```
http://localhost:8080/api/v1
```

## Default Accounts

| Role     | Email                  | Password      |
|----------|------------------------|---------------|
| Finance  | finance@budgetflow.id  | Finance123!   |
| Manager  | manager@budgetflow.id  | Manager123!   |
| Employee | employee@budgetflow.id | Employee123!  |

## API Endpoints

| Method | Path                                | Auth | Role              |
|--------|-------------------------------------|------|-------------------|
| POST   | `/auth/register`                    | No   | —                 |
| POST   | `/auth/login`                       | No   | —                 |
| POST   | `/auth/refresh`                     | No   | —                 |
| GET    | `/me/balance`                       | Yes  | All               |
| POST   | `/topups`                           | Yes  | Manager           |
| POST   | `/topups/:public_id/review`         | Yes  | Finance           |
| GET    | `/me/topups`                        | Yes  | Manager           |
| POST   | `/projects`                         | Yes  | Manager           |
| GET    | `/projects`                         | Yes  | All               |
| GET    | `/projects/:public_id`              | Yes  | All               |
| DELETE | `/projects/:public_id`              | Yes  | Manager           |
| POST   | `/projects/:public_id/restore`      | Yes  | Manager           |
| POST   | `/projects/:public_id/claims`       | Yes  | Employee          |
| POST   | `/claims/:public_id/review`         | Yes  | Finance, Manager  |
| GET    | `/me/claims`                        | Yes  | Employee          |
| POST   | `/payouts`                          | Yes  | Employee          |
| GET    | `/payouts`                          | Yes  | Employee          |
| POST   | `/payouts/:public_id/review`        | Yes  | Finance           |

## Running Tests

```bash
# Run unit tests only (Does not require database)
make test

# Run integration & concurrent tests (Requires running PostgreSQL)
make test-integration
```

## Project Structure

```
budgetflow/
├── cmd/api/main.go              # Wiring, DI, server start
├── internal/
│   ├── domain/                  # Entities, enums, DTOs (zero framework deps)
│   ├── usecase/                 # Business logic, TX orchestration
│   ├── repository/              # GORM data access, WithTx pattern
│   ├── delivery/http/           # Handlers & router
│   ├── middleware/              # JWT auth & RBAC
│   └── shared/                  # Errors, response, JWT, bcrypt, query builder
├── migrations/                  # SQL migration files
├── seeds/                       # Database seeders
└── Makefile
```

### Fully Implemented
- **Atomicity & Idempotency**: Handled strictly via GORM `WithTx(tx)` pattern and an `Idempotency-Key` caching middleware.
- **Refresh Token Rotation**: Implemented with a `sessions` table (7-day expiry) that blocks token reuse.
- **Rate Limiting**: IP-based rate limiting (100 req/min) using `x/time/rate`.
- **Soft Delete**: Projects can be securely soft-deleted and restored via `deleted_at`.
- **Observability**: Handled using structured JSON logging (`log/slog`).
- **Graceful Shutdown**: Enabled with a 5-second drain window on SIGINT/SIGTERM.
