#!/usr/bin/env bash
# Load server-local .env (never deployed) and exec the API binary.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"
if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi
exec "$ROOT/server"
