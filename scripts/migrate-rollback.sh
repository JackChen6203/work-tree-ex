#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

# Roll back all applied migrations for emergency recovery scenarios.
"$ROOT_DIR/scripts/migrate.sh" down-all

