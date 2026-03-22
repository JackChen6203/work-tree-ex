# SRE Runbook

## Core Health Checks
- `GET /healthz`
- backend CI test job
- migration smoke test job
- frontend build and test job
- deploy workflow on `main`

## Expected Pipelines
1. push to `main`
2. CI validates frontend + backend + migrations
3. deploy workflow syncs repository to VPS
4. server runs migrations and starts docker compose stack

## Operational Checks
- verify GitHub Actions CI status is green
- confirm deploy workflow reached VPS successfully
- confirm migration step completed before app restart
- validate `/healthz` after deploy
- smoke check auth, trips list, notifications list, sync bootstrap

## Rollback
- reset VPS checkout to previous green commit
- rerun docker compose with previous revision
- if migration introduced incompatible change, execute matching down migration or restore DB backup

## Known Local Environment Limits
This workspace may not have Go and Docker installed, so GitHub Actions is the source of truth for backend integration validation.
