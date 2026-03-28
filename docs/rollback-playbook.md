# Rollback Playbook

## Scope

This document covers production rollback for the single-host deployment pipeline in `.github/workflows/deploy.yml`.

## Trigger rollback from GitHub Actions

1. Open Actions → `Deploy` workflow.
2. Click `Run workflow`.
3. Set:
   - `target=rollback`
   - `rollback_ref=<commit/tag/sha>` (optional)
4. Run workflow.

## Rollback behavior

Workflow executes:

- `scripts/deploy/remote-rollback.sh`

Server-side steps:

1. `git fetch --all --prune`
2. Resolve rollback target:
   - use input `rollback_ref`, or
   - fallback `.deploy/previous_successful_sha`
3. `git reset --hard <target>`
4. `docker compose up -d --build redis api worker web`
5. wait for `/healthz` success
6. record `.deploy/last_successful_sha`

## Emergency manual rollback (SSH)

```bash
cd /home/opc/apps/work-tree-ex
./scripts/deploy/remote-rollback.sh <commit-or-tag> /home/opc/apps/work-tree-ex
```

If you want automatic fallback to previous successful release:

```bash
cd /home/opc/apps/work-tree-ex
./scripts/deploy/remote-rollback.sh "" /home/opc/apps/work-tree-ex
```

## Validation checklist after rollback

1. `curl -f http://127.0.0.1/healthz`
2. Verify web homepage loads.
3. Verify key API flows (auth/session refresh, trip read/list).
4. Check worker health endpoint.
5. Confirm error rate recovers in logs/monitoring.
