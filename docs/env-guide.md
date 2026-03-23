# Environment Guide

This guide was produced by scanning the current project source, Docker Compose, GitHub Actions workflows, and Terraform.

## 1. Where each env file is used

### Root `.env`

Used by:

- `docker-compose.yml`
- backend API container
- backend worker container
- GitHub Actions deploy secret `APP_ENV_FILE`

Create it by copying:

```bash
cp .env.example .env
```

### `apps/web/.env.local`

Used only by Vite when you run the frontend locally with `npm run dev`.

Create it by copying:

```bash
cp apps/web/.env.example apps/web/.env.local
```

### GitHub Actions Secrets

Used by:

- `.github/workflows/deploy.yml`

Required secrets:

- `ORACLE_SSH_KEY`
- `APP_ENV_FILE`
- `MIGRATE_DATABASE_URL`

### Terraform inputs

Used by:

- `infra/terraform/*.tf`

These are not read from the app runtime `.env`. Set them in `terraform.tfvars` or with `TF_VAR_*`.

## 2. Variables found in the project

### Root `.env` variables

| Variable | Required | Used by | Notes |
| --- | --- | --- | --- |
| `APP_ENV` | Yes | API, worker | `dev`, `staging`, `prod` |
| `TRIPS_STORE` | Yes | API, worker | `postgres` for real persistence, `memory` for quick local API-only dev |
| `HTTP_HOST` | Yes | API | Usually `0.0.0.0` |
| `HTTP_PORT` | Yes | API | Default `8080` |
| `HTTP_READ_TIMEOUT_SEC` | No | API | Default `10` |
| `HTTP_WRITE_TIMEOUT_SEC` | No | API | Default `15` |
| `HTTP_SHUTDOWN_TIMEOUT_SEC` | No | API | Default `10` |
| `FRONTEND_BASE_URL` | Recommended | API OAuth redirect | Important when OAuth callback needs to redirect back to frontend |
| `CORS_ALLOWED_ORIGINS` | Yes | API | Comma-separated origins |
| `POSTGRES_IMAGE` | No | Docker Compose | Local database image tag |
| `DB_HOST` | Yes | API, worker, Compose | `localhost` locally, database host/IP in deployed env |
| `DB_PORT` | Yes | API, worker, Compose | Usually `5432` |
| `DB_USER` | Yes | API, worker, Compose | DB username |
| `DB_PASSWORD` | Yes | API, worker, Compose | DB password |
| `DB_NAME` | Yes | API, worker, Compose | DB name |
| `DB_SSLMODE` | Yes | API, worker | `disable` locally, usually `require` or platform-specific in managed DB setups |
| `DB_MAX_OPEN_CONNS` | No | API, worker | Connection pool tuning |
| `DB_MAX_IDLE_CONNS` | No | API, worker | Connection pool tuning |
| `DB_CONN_MAX_LIFETIME_MIN` | No | API, worker | Connection pool tuning |
| `REDIS_ADDR` | Yes | API, worker | `host:port` |
| `REDIS_PORT` | No | Docker Compose | Local published port |
| `REDIS_PASSWORD` | Yes | API, worker, Compose | Redis password |
| `REDIS_DB` | No | API, worker | Usually `0` |
| `JWT_SECRET` | Yes | API | Must be random in non-dev environments |
| `JWT_ACCESS_TTL_MIN` | No | API | Default `60` |
| `JWT_REFRESH_TTL_HOURS` | No | API | Default `168` |
| `AUTH_ALLOW_MAGIC_LINK_PREVIEW` | Dev only | API auth routes | Set `true` only for local preview-code login during development |
| `OAUTH_GOOGLE_CLIENT_ID` | Optional | API auth routes | Provider client ID |
| `OAUTH_GOOGLE_CLIENT_SECRET` | Required for real Google OAuth | API auth routes | Needed for server-side Google authorization-code exchange |
| `OAUTH_APPLE_CLIENT_ID` | Optional | API auth routes | Provider client ID |
| `OAUTH_FACEBOOK_CLIENT_ID` | Optional | API auth routes | Provider client ID |
| `OAUTH_X_CLIENT_ID` | Optional | API auth routes | Provider client ID |
| `OAUTH_GITHUB_CLIENT_ID` | Optional | API auth routes | Provider client ID |
| `OAUTH_LINE_CLIENT_ID` | Optional | API auth routes | Provider client ID |
| `OAUTH_KAKAO_CLIENT_ID` | Optional | API auth routes | Provider client ID |
| `OAUTH_WECHAT_CLIENT_ID` | Optional | API auth routes | Provider client ID |
| `OAUTH_TRIPADVISOR_CLIENT_ID` | Optional | API auth routes | Provider client ID |
| `OAUTH_BOOKING_CLIENT_ID` | Optional | API auth routes | Provider client ID |
| `WEB_PORT` | No | Docker Compose | Frontend nginx published port |
| `MIGRATE_DATABASE_URL` | Recommended for external DB | Docker Compose migrate/manual migration | Use a full Postgres URL when your DB password contains special chars |

### Frontend Vite variables

| Variable | Required | Used by | Notes |
| --- | --- | --- | --- |
| `VITE_API_BASE_URL` | Optional | `apps/web` | Needed when Vite dev server should call a backend origin directly, for example `http://localhost:8080` |
| `VITE_ENABLE_MAGIC_LINK_AUTH` | Dev only | `apps/web` | Defaults to enabled in `npm run dev`, should stay `false` in production builds |
| `VITE_OAUTH_PROVIDERS` | Recommended | `apps/web` | Comma-separated provider ids to show in the frontend, for example `google` |

### GitHub Actions secrets

| Secret | Required | Used by | Notes |
| --- | --- | --- | --- |
| `ORACLE_SSH_KEY` | Yes for deploy | `deploy.yml` | Private SSH key for server login |
| `APP_ENV_FILE` | Yes for deploy | `deploy.yml` | Full production root `.env` contents |
| `MIGRATE_DATABASE_URL` | Yes for external production DB | `deploy.yml` | Full Postgres migration URL used by GitHub Actions before deploy |

### Terraform inputs

These are variables, not runtime app env values. The project currently defines:

- `project_id`
- `region`
- `environment`
- `db_tier`
- `db_disk_size_gb`
- `db_name`
- `redis_memory_size_gb`
- `redis_tier`
- `api_image`
- `worker_image`
- `api_min_instances`
- `api_max_instances`

If you prefer env-style input, export them as:

```bash
export TF_VAR_project_id="your-gcp-project-id"
export TF_VAR_region="asia-east1"
export TF_VAR_environment="prod"
```

## 3. Recommended local setups

### Option A: Run the full stack with Docker Compose

1. Copy `.env.example` to `.env`.
2. Update at least:
   - `DB_PASSWORD`
   - `REDIS_PASSWORD`
   - `JWT_SECRET`
   - `FRONTEND_BASE_URL`
3. Start services:

```bash
docker compose up -d postgres redis
docker compose --profile tools run --rm migrate
docker compose up -d api worker web
```

4. Open:
   - frontend: `http://localhost`
   - backend health: `http://localhost/healthz`

In this mode you usually do not need `apps/web/.env.local`, because nginx proxies `/api` on the same origin.

### Option B: Run Vite frontend + backend directly

1. Copy `.env.example` to `.env`.
2. Copy `apps/web/.env.example` to `apps/web/.env.local`.
3. Set:
   - root `.env`:
     - `FRONTEND_BASE_URL=http://localhost:5173`
     - `CORS_ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173`
   - `apps/web/.env.local`:
     - `VITE_API_BASE_URL=http://localhost:8080`
4. Run backend and frontend separately.

## 4. How to obtain each secret/value

### Values you generate yourself

These do not need vendor signup. Generate them yourself:

- `DB_PASSWORD`
- `REDIS_PASSWORD`
- `JWT_SECRET`

Example:

```bash
openssl rand -base64 32
```

Recommended:

- `DB_PASSWORD`: 24+ random chars
- `REDIS_PASSWORD`: 24+ random chars
- `JWT_SECRET`: 32 to 64 random chars minimum

### OAuth client IDs

Only configure providers you actually plan to expose in the UI. In production, this backend only permits real Google OAuth when both `OAUTH_GOOGLE_CLIENT_ID` and `OAUTH_GOOGLE_CLIENT_SECRET` are set. Other providers remain development-only placeholders unless you implement their token exchange.

General process for every provider:

1. Create a developer account for that provider.
2. Create an OAuth app.
3. Add an authorized redirect URI.
4. Copy the provider's client ID into the matching `OAUTH_*_CLIENT_ID` variable.
5. For Google, also copy the client secret into `OAUTH_GOOGLE_CLIENT_SECRET`.

Redirect URI pattern:

- local backend direct call: `http://localhost:8080/api/v1/auth/oauth/<provider>/callback`
- docker compose same-origin proxy: `http://localhost/api/v1/auth/oauth/<provider>/callback`
- production: `https://<your-domain>/api/v1/auth/oauth/<provider>/callback`

Examples:

- `OAUTH_GOOGLE_CLIENT_ID` and `OAUTH_GOOGLE_CLIENT_SECRET` for Google sign-in
- `OAUTH_GITHUB_CLIENT_ID` for future GitHub sign-in work
- `OAUTH_LINE_CLIENT_ID` for future LINE sign-in work
- You still need the redirect URI registered exactly as your browser-facing API URL.
- Set `FRONTEND_BASE_URL` to the frontend origin you want the OAuth callback to redirect back to after login.

### GitHub Actions deploy secrets

#### `ORACLE_SSH_KEY`

Use a deploy-only SSH key pair.

1. Generate a key pair:

```bash
ssh-keygen -t ed25519 -C "github-actions-deploy"
```

2. Put the public key on your server user's `~/.ssh/authorized_keys`.
3. Put the private key contents into the GitHub repository secret `ORACLE_SSH_KEY`.

#### `APP_ENV_FILE`

This should be the full contents of your production root `.env`.

Recommended flow:

1. Start from `.env.example`.
2. Replace all development passwords and `JWT_SECRET`.
3. Set real production hosts/origins, for example:
   - `APP_ENV=prod`
   - `FRONTEND_BASE_URL=https://your-domain`
   - `CORS_ALLOWED_ORIGINS=https://your-domain`
   - managed DB / Redis connection values
4. Copy the full file contents into the GitHub repository secret `APP_ENV_FILE`.

#### `MIGRATE_DATABASE_URL`

Use a full connection string for migrations. This is safer than rebuilding the URL from pieces when the password contains special characters.

Supabase pooler example:

```text
postgresql://postgres.<project-ref>:<URL_ENCODED_PASSWORD>@aws-1-ap-northeast-1.pooler.supabase.com:6543/postgres?sslmode=require
```

Notes:

- Replace `<project-ref>` with your Supabase project ref.
- If your Supabase password contains characters such as `@`, `:`, `/`, or `?`, URL-encode it before putting it into the URL.
- Store the final full URL in the GitHub repository secret `MIGRATE_DATABASE_URL`.

## 5. Terraform setup

If you use the Terraform folder, create `infra/terraform/terraform.tfvars` or export `TF_VAR_*`.

Example `terraform.tfvars`:

```hcl
project_id        = "your-gcp-project-id"
region            = "asia-east1"
environment       = "prod"
db_name           = "travel_planner"
api_image         = "gcr.io/your-project/travel-api:latest"
worker_image      = "gcr.io/your-project/travel-worker:latest"
api_min_instances = 1
api_max_instances = 10
```

Terraform currently provisions secrets for:

- DB password
- JWT secret
- LLM encryption key

Note that the current backend runtime only consumes `DB_PASSWORD` and `JWT_SECRET`. The LLM encryption key is provisioned in Terraform, but I did not find a corresponding runtime read path in the current application code.

## 6. Practical minimum required values

### Minimum for local Docker Compose

- `TRIPS_STORE=postgres`
- `DB_PASSWORD`
- `REDIS_PASSWORD`
- `JWT_SECRET`
- `FRONTEND_BASE_URL`

### Minimum for Vite frontend development

- `apps/web/.env.local` with `VITE_API_BASE_URL`

### Minimum for deployment workflow

- GitHub secret `ORACLE_SSH_KEY`
- GitHub secret `APP_ENV_FILE`
- GitHub secret `MIGRATE_DATABASE_URL` if production DB is external/Supabase

## 7. Files scanned

Primary sources scanned for this guide:

- `backend/internal/platform/config/config.go`
- `backend/internal/auth/routes.go`
- `apps/web/src/lib/api.ts`
- `docker-compose.yml`
- `.github/workflows/deploy.yml`
- `.github/workflows/ci.yml`
- `infra/terraform/*.tf`
