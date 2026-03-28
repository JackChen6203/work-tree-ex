# Manual Todo (Keys / Env / External Service Decisions)

詳細操作手冊：[`docs/manual-todo-handbook.md`](./docs/manual-todo-handbook.md)

## 1) GitHub Actions deployment secrets

- [ ] Add `ORACLE_SSH_KEY` in GitHub repo secrets.
  - Where: GitHub Repository → Settings → Secrets and variables → Actions.
  - Value: private deploy SSH key for `opc@217.142.247.83`.
- [ ] Add `APP_ENV_FILE` in GitHub repo secrets.
  - Where: same as above.
  - Value: full root `.env` content for production runtime.
- [ ] Add `MIGRATE_DATABASE_URL` in GitHub repo secrets.
  - Where: same as above.
  - Value: full production Postgres URL for migration job.

## 2) Recommended environment-specific secrets (staging/production split)

- [ ] Add `STAGING_SSH_KEY` (optional, fallback to `ORACLE_SSH_KEY`).
- [ ] Add `STAGING_APP_ENV_FILE` (optional, fallback to `APP_ENV_FILE`).
- [ ] Add `STAGING_MIGRATE_DATABASE_URL` (optional, fallback to `MIGRATE_DATABASE_URL`).
- [ ] Add `PRODUCTION_SSH_KEY` (optional, fallback to `ORACLE_SSH_KEY`).
- [ ] Add `PRODUCTION_APP_ENV_FILE` (optional, fallback to `APP_ENV_FILE`).
- [ ] Add `PRODUCTION_MIGRATE_DATABASE_URL` (optional, fallback to `MIGRATE_DATABASE_URL`).

## 3) Supabase project values (RLS + frontend anon client)

- [ ] Fill frontend Supabase env when enabling browser direct Supabase access:
  - File: `apps/web/.env.local` (local) or CI/CD runtime env injection.
  - Keys:
    - `VITE_SUPABASE_URL`
    - `VITE_SUPABASE_ANON_KEY`
- [ ] Keep service role key only on backend secret scope:
  - File: root `.env` / secret manager.
  - Key: `SUPABASE_SERVICE_ROLE_KEY` (do not expose in Vite env).

## 4) Map / Push / Email providers (when enabling real external integration)

- [ ] Map provider:
  - Root `.env`: `GOOGLE_MAPS_API_KEY` and/or `MAPBOX_API_KEY`
- [ ] Firebase push:
  - Root `.env`: `FCM_SERVICE_ACCOUNT_JSON` (or `FCM_SERVICE_ACCOUNT_FILE`) and optionally `FCM_PROJECT_ID`
  - Frontend `apps/web/.env.local`: Firebase web config keys (`VITE_FIREBASE_*`)
- [ ] Email provider:
  - Root `.env`: set `EMAIL_PROVIDER_PRIMARY` and provider key (`RESEND_API_KEY` or `SENDGRID_API_KEY`)

## 5) Rollback operation readiness

- [ ] Confirm production server has this repository at `/home/opc/apps/work-tree-ex`.
- [ ] Confirm server has Docker Compose and network access to DB/Redis.
- [ ] Dry-run `workflow_dispatch` with `target=production` and `strategy=rolling` once.
- [ ] Dry-run `workflow_dispatch` with `target=rollback` once (non-peak hours).
