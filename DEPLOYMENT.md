# Deployment

## Runtime model

- `web`: static frontend served by nginx, proxies `/api` to the API container
- `api`: Go Gin HTTP API
- `worker`: background worker
- `postgres`: Postgres / PostGIS
- `redis`: queue/cache support

## Local run

1. Copy `.env.example` to `.env`
2. Start backend stack:
   - `docker compose up -d postgres redis`
   - `docker compose --profile tools run --rm migrate`
   - `docker compose up -d api worker web`
3. Open:
   - Web: `http://localhost`
   - API health: `http://localhost/healthz`

## GitHub Actions secrets

Create these repository secrets before enabling deployment:

- `ORACLE_SSH_KEY`: private key content for the VPS
- `APP_ENV_FILE`: full production `.env` file contents

The deploy workflow pushes the repository over SSH and runs `docker compose --profile tools run --rm migrate` followed by `docker compose up -d --build`.
