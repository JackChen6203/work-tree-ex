#!/usr/bin/env bash
set -euo pipefail

STRATEGY="${1:-rolling}"
CANARY_MINUTES="${2:-10}"
PROJECT_ROOT="${3:-$(pwd)}"
HEALTH_URL_PRIMARY="${HEALTH_URL_PRIMARY:-http://127.0.0.1/healthz}"
HEALTH_URL_CANARY="${HEALTH_URL_CANARY:-http://127.0.0.1:18080/healthz}"
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
  while (( i <= attempts )); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep "$sleep_seconds"
    i=$((i + 1))
  done
  return 1
}

deploy_rolling() {
  docker compose pull || true
  docker compose up -d --build redis api worker web
  check_health "$HEALTH_URL_PRIMARY" 30 3
}

deploy_canary_stack() {
  WEB_PORT="$CANARY_WEB_PORT" REDIS_PORT="$CANARY_REDIS_PORT" \
    docker compose -p "$CANARY_PROJECT_NAME" up -d --build redis api worker web
  check_health "$HEALTH_URL_CANARY" 30 3
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
    end_time=$((SECONDS + CANARY_MINUTES * 60))
    while (( SECONDS < end_time )); do
      if ! curl -fsS "$HEALTH_URL_CANARY" >/dev/null 2>&1; then
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
