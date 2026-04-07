#!/usr/bin/env bash
set -euo pipefail

STRATEGY="${1:-rolling}"
CANARY_MINUTES="${2:-10}"
PROJECT_ROOT="${3:-$(pwd)}"
HEALTH_URL_PRIMARY="${HEALTH_URL_PRIMARY:-}"
HEALTH_URL_CANARY="${HEALTH_URL_CANARY:-}"
CANARY_PROJECT_NAME="${CANARY_PROJECT_NAME:-travel-canary}"
CANARY_WEB_PORT="${CANARY_WEB_PORT:-18080}"
CANARY_REDIS_PORT="${CANARY_REDIS_PORT:-16379}"

require_integer() {
  local value="$1"
  local name="$2"
  if ! [[ "$value" =~ ^[0-9]+$ ]]; then
    echo "${name} must be an integer: ${value}" >&2
    exit 1
  fi
}

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
  local project_name="${1:-}"
  local published=""
  if [[ -n "$project_name" ]]; then
    published="$(docker compose -p "$project_name" port web 80 2>/dev/null | tail -n 1 || true)"
  else
    published="$(docker compose port web 80 2>/dev/null | tail -n 1 || true)"
  fi

  if [[ -n "$published" ]]; then
    printf 'http://127.0.0.1:%s/healthz\n' "${published##*:}"
    return 0
  fi

  printf 'http://127.0.0.1/healthz\n'
}

deploy_rolling() {
  docker compose pull || true
  docker compose up -d --build redis api worker web
  local health_url="${HEALTH_URL_PRIMARY:-$(compose_health_url)}"
  check_health "$health_url" 30 3
}

deploy_canary_stack() {
  WEB_PORT="$CANARY_WEB_PORT" REDIS_PORT="$CANARY_REDIS_PORT" \
    docker compose -p "$CANARY_PROJECT_NAME" up -d --build redis api worker web
  local health_url="${HEALTH_URL_CANARY:-$(compose_health_url "$CANARY_PROJECT_NAME")}"
  check_health "$health_url" 30 3
}

cleanup_canary_stack() {
  WEB_PORT="$CANARY_WEB_PORT" REDIS_PORT="$CANARY_REDIS_PORT" \
    docker compose -p "$CANARY_PROJECT_NAME" down --remove-orphans || true
}

record_release_meta() {
  mkdir -p .deploy
  local previous=""
  previous="$(cat .deploy/last_successful_sha 2>/dev/null || true)"
  if [[ -n "$previous" ]]; then
    printf '%s\n' "$previous" > .deploy/previous_successful_sha
  fi
  git rev-parse HEAD > .deploy/last_successful_sha
}

require_integer "$CANARY_MINUTES" "CANARY_MINUTES"

cd "$PROJECT_ROOT"

case "$STRATEGY" in
  rolling)
    deploy_rolling
    ;;
  blue-green)
    # Single-host safe blue-green:
    # deploy candidate stack on alternate port, validate, then promote.
    deploy_canary_stack
    deploy_rolling
    cleanup_canary_stack
    ;;
  canary)
    # Single-host safe canary:
    # keep candidate stack on alternate port and run synthetic probes first.
    deploy_canary_stack
    canary_health_url="${HEALTH_URL_CANARY:-$(compose_health_url "$CANARY_PROJECT_NAME")}"
    end_time=$((SECONDS + CANARY_MINUTES * 60))
    while (( SECONDS < end_time )); do
      if ! check_health "$canary_health_url" 1 0; then
        cleanup_canary_stack
        echo "canary health check failed" >&2
        exit 1
      fi
      sleep 15
    done
    deploy_rolling
    cleanup_canary_stack
    ;;
  *)
    echo "unsupported strategy: ${STRATEGY}" >&2
    echo "expected one of: rolling, blue-green, canary" >&2
    exit 1
    ;;
esac

record_release_meta
echo "deploy completed with strategy=${STRATEGY}"
