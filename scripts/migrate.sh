#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
MIGRATE_BIN="${MIGRATE_BIN:-migrate}"
MIGRATE_PATH="${MIGRATE_PATH:-$ROOT_DIR/backend/migrations}"
MIGRATE_DATABASE_URL="${MIGRATE_DATABASE_URL:-}"

if [[ -z "$MIGRATE_DATABASE_URL" ]]; then
  echo "MIGRATE_DATABASE_URL is required" >&2
  exit 1
fi

if [[ ! -d "$MIGRATE_PATH" ]]; then
  echo "migration path does not exist: $MIGRATE_PATH" >&2
  exit 1
fi

run_migrate() {
  set +e
  local output
  output="$("$MIGRATE_BIN" -path "$MIGRATE_PATH" -database "$MIGRATE_DATABASE_URL" "$@" 2>&1)"
  local status=$?
  set -e

  printf '%s\n' "$output"

  if [[ "$status" -ne 0 && "$output" != *"no change"* ]]; then
    return "$status"
  fi
  return 0
}

command="${1:-up}"
case "$command" in
  up)
    run_migrate up
    ;;
  down)
    steps="${2:-1}"
    run_migrate down "$steps"
    ;;
  down-all)
    run_migrate down -all
    ;;
  version)
    "$MIGRATE_BIN" -path "$MIGRATE_PATH" -database "$MIGRATE_DATABASE_URL" version
    ;;
  *)
    echo "unsupported command: $command" >&2
    echo "usage: $0 [up|down <steps>|down-all|version]" >&2
    exit 1
    ;;
esac

