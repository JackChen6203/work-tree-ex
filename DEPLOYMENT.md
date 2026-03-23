# Deployment

## Runtime model

- `web`: static frontend served by nginx, proxies `/api` to the API container
- `api`: Go Gin HTTP API
- `worker`: background worker
- `postgres`: Postgres / PostGIS
- `redis`: queue/cache support

## Local run

1. Copy `.env.example` to `.env`
2. If you run Vite directly, also copy `apps/web/.env.example` to `apps/web/.env.local`
3. Read `docs/env-guide.md` for the full variable inventory and setup notes
4. Start backend stack:
   - `docker compose up -d postgres redis`
   - `docker compose --profile tools run --rm migrate`
   - `docker compose up -d api worker web`
5. Open:
   - Web: `http://localhost`
   - API health: `http://localhost/healthz`

## GitHub Actions secrets

Create these repository secrets before enabling deployment:

- `ORACLE_SSH_KEY`: private key content for the VPS
- `APP_ENV_FILE`: full production `.env` file contents
- `MIGRATE_DATABASE_URL`: production database URL for GitHub Actions migrations

The deploy workflow now:

1. validates the frontend
2. runs migrations against `MIGRATE_DATABASE_URL`
3. pushes the repository over SSH
4. runs `docker compose up -d --build redis api worker web`
