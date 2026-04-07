# Deployment

## Runtime model

- `web`: static frontend served by nginx, proxies `/api` to the API container
- `api`: Go Gin HTTP API
- `worker`: background worker
- `postgres`: Postgres / PostGIS (local profile)
- `redis`: queue/cache support

## Local run

1. Copy `.env.example` to `.env`.
2. If you run Vite directly, also copy `apps/web/.env.example` to `apps/web/.env.local`.
3. Read `docs/env-guide.md` for the full variable inventory and setup notes.
4. Start backend stack:
   - `docker compose up -d postgres redis`
   - `docker compose --profile tools run --rm migrate`
   - `docker compose up -d api worker web`
5. Open:
   - Web: `http://localhost`
   - API health: `http://localhost/healthz`

## Supabase local dev bridge (`supabase start`)

1. Ensure Supabase CLI is installed.
2. Run:
   - `make supabase-start`
3. This command starts local Supabase and generates:
   - `.env.supabase.local` (backend/compose bridge)
   - `apps/web/.env.supabase.local` (frontend bridge)
4. Run migrations and stack with generated env:
   - `make docker-migrate-supabase`
   - `make docker-up-supabase`

## CI/CD pipeline

Workflow: `.github/workflows/deploy.yml`

- `push main`:
  - verify frontend
  - stop staging `api`/`worker`, then run staging migration
  - deploy staging (rolling)
- `workflow_dispatch`:
  - `target=production` with strategy `rolling` / `blue-green` / `canary`
  - `target=rollback` for emergency rollback

Deployment runs are serialized by workflow concurrency so two migrations/deploys do not race on the same database and host.
For single-host deployment, `blue-green` / `canary` use a candidate stack on alternate port first, then promote to primary stack after health checks.

## Required GitHub Actions secrets

Minimum (existing production path):

- `ORACLE_SSH_KEY`
- `APP_ENV_FILE`
- `MIGRATE_DATABASE_URL`

Recommended per environment:

- Staging:
  - `STAGING_SSH_KEY`
  - `STAGING_APP_ENV_FILE`
  - `STAGING_MIGRATE_DATABASE_URL`
- Production:
  - `PRODUCTION_SSH_KEY`
  - `PRODUCTION_APP_ENV_FILE`
  - `PRODUCTION_MIGRATE_DATABASE_URL`

If staging/production-specific SSH or app env secrets are not set, workflow falls back to the existing production secret names. Staging migrations only run when `STAGING_MIGRATE_DATABASE_URL` is configured; otherwise they are skipped so push deploys do not touch the production/shared migration connection. Production migrations can still fall back to `MIGRATE_DATABASE_URL`.

## Rollback

Use `workflow_dispatch` with `target=rollback` and optional `rollback_ref`.

- If `rollback_ref` is empty, workflow uses `.deploy/previous_successful_sha` on server.
- Detailed procedure: `docs/rollback-playbook.md`.
