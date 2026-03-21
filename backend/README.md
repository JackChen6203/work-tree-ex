# Backend Bootstrap

License: proprietary, all rights reserved. No use or redistribution is permitted
without a separate written commercial license.

This backend is scaffolded with Go + Gin and follows the engineering rules in `rules.md`.

## Structure

- `cmd/api`: API entrypoint
- `cmd/worker`: Background worker entrypoint
- `internal/*`: Domain modules and platform helpers
- `migrations`: Versioned SQL migrations

## Quick start

1. Install Go 1.23+
2. Run in backend directory:

```bash
go mod tidy
go run ./cmd/api
```

The API starts at `http://0.0.0.0:8080` by default.

## Environment variables

- `APP_ENV` (default: `dev`)
- `TRIPS_STORE` (default: `memory`, options: `memory` | `postgres`)
- `HTTP_HOST` (default: `0.0.0.0`)
- `HTTP_PORT` (default: `8080`)
- `HTTP_READ_TIMEOUT_SEC` (default: `10`)
- `HTTP_WRITE_TIMEOUT_SEC` (default: `15`)
- `HTTP_SHUTDOWN_TIMEOUT_SEC` (default: `10`)
- `FRONTEND_BASE_URL` (default: `http://localhost:5173`)

OAuth provider client IDs (optional, used when available):

- `OAUTH_GOOGLE_CLIENT_ID`
- `OAUTH_APPLE_CLIENT_ID`
- `OAUTH_FACEBOOK_CLIENT_ID`
- `OAUTH_X_CLIENT_ID`
- `OAUTH_GITHUB_CLIENT_ID`
- `OAUTH_LINE_CLIENT_ID`
- `OAUTH_KAKAO_CLIENT_ID`
- `OAUTH_WECHAT_CLIENT_ID`
- `OAUTH_TRIPADVISOR_CLIENT_ID`
- `OAUTH_BOOKING_CLIENT_ID`

When a provider client ID is not configured, `/api/v1/auth/oauth/:provider/start` runs in development shortcut mode and redirects directly to callback with a generated code so you can test the login flow end-to-end.

## MVP status

- API/worker runnable skeleton
- Request ID, structured access log, panic recovery middleware
- `/healthz` endpoint
- MVP API for trips: list/create/get/patch (in-memory storage)
- Route contract skeleton for remaining domains
- Initial migrations added

## Test environment

1. Start dependencies:

```bash
docker compose -f docker-compose.test.yml up -d
```

2. Run tests:

```bash
go test ./...
```

Or run the helper script from repository root:

```powershell
./scripts/test-backend.ps1
```

Run migration smoke test from repository root:

```powershell
./scripts/migrate-smoke.ps1
```
