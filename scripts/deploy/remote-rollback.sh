#!/usr/bin/env bash
set -euo pipefail

TARGET_REF="${1:-}"
PROJECT_ROOT="${2:-$(pwd)}"
HEALTH_URL_PRIMARY="${HEALTH_URL_PRIMARY:-}"

check_health() {
  local url="$1"
  local attempts="${2:-20}"
  local sleep_seconds="${3:-3}"
  local i=1
  local status=""
  while (( i <= attempts )); do
    status="$(curl -sS -o /dev/null -w "%{http_code}" --max-time 5 "$url" 2>/dev/null || true)"
    if [[ "$status" == "200" ]]; then
      return 0
    fi
    sleep "$sleep_seconds"
    i=$((i + 1))
  done
  echo "health check failed for ${url} with last status ${status:-curl_failed}" >&2
  return 1
}

compose_health_url() {
  local published=""
  published="$(docker compose port web 80 2>/dev/null | tail -n 1 || true)"
  if [[ -n "$published" ]]; then
    printf 'http://127.0.0.1:%s/healthz\n' "${published##*:}"
    return 0
  fi

  printf 'http://127.0.0.1/healthz\n'
}

cd "$PROJECT_ROOT"
mkdir -p .deploy

if [[ -z "$TARGET_REF" ]]; then
  TARGET_REF="$(cat .deploy/previous_successful_sha 2>/dev/null || true)"
fi

if [[ -z "$TARGET_REF" ]]; then
  echo "rollback ref is required (or provide .deploy/previous_successful_sha)" >&2
  exit 1
fi

git fetch --all --prune
git checkout -B main origin/main
git reset --hard "$TARGET_REF"

docker compose pull || true
docker compose up -d --build redis api worker web
check_health "${HEALTH_URL_PRIMARY:-$(compose_health_url)}" 30 3

git rev-parse HEAD > .deploy/last_successful_sha
echo "rollback completed to $(git rev-parse --short HEAD)"
