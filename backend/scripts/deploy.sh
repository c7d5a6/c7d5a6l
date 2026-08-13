#!/usr/bin/env bash
# Cross-compile the API and rsync it to foundry, then restart via PM2.
#
# Working tree must be clean. Remote .env / data / logs are never overwritten.
#
# Host:  foundry@foundry.owlbeardm.com:/home/foundry/c7d5a6l
# Nginx: api.league.c7d5a6.com → 127.0.0.1:18765
#
# Env overrides:
#   C7D5A6L_DEPLOY_HOST   default foundry@foundry.owlbeardm.com
#   C7D5A6L_DEPLOY_DIR    default /home/foundry/c7d5a6l
#   C7D5A6L_GOARCH        default: detected from remote uname -m
set -euo pipefail

DEPLOY_HOST="${C7D5A6L_DEPLOY_HOST:-foundry@foundry.owlbeardm.com}"
REMOTE_DIR="${C7D5A6L_DEPLOY_DIR:-/home/foundry/c7d5a6l}"
APP_NAME="c7d5a6l"

BACKEND="$(cd "$(dirname "$0")/.." && pwd)"
REPO="$(git -C "$BACKEND" rev-parse --show-toplevel)"
cd "$REPO"

if [[ -n "$(git status --porcelain)" ]]; then
  echo "error: uncommitted content. Commit or stash before deploying." >&2
  git status --porcelain >&2
  exit 1
fi

if [[ "$(git rev-parse --abbrev-ref HEAD)" == "gh-pages" ]]; then
  echo "error: run this from a source branch, not gh-pages." >&2
  exit 1
fi

for cmd in go rsync ssh; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "error: $cmd is required." >&2
    exit 1
  fi
done

remote() {
  ssh "$DEPLOY_HOST" "$@"
}

echo "==> Probe ${DEPLOY_HOST}"
REMOTE_UNAME="$(remote uname -m)"
if [[ -n "${C7D5A6L_GOARCH:-}" ]]; then
  GOARCH="$C7D5A6L_GOARCH"
else
  case "$REMOTE_UNAME" in
    x86_64) GOARCH=amd64 ;;
    aarch64|arm64) GOARCH=arm64 ;;
    *)
      echo "error: unsupported remote arch ${REMOTE_UNAME} (set C7D5A6L_GOARCH)." >&2
      exit 1
      ;;
  esac
fi

STAGE="$(mktemp -d)"
cleanup() { rm -rf "$STAGE"; }
trap cleanup EXIT

echo "==> Build linux/${GOARCH} → ${STAGE}/server"
cd "$BACKEND"
CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" go build -trimpath -ldflags='-s -w' -o "$STAGE/server" ./cmd/server
cp "$BACKEND/deploy/run.sh" "$STAGE/run.sh"
cp "$BACKEND/deploy/ecosystem.config.cjs" "$STAGE/ecosystem.config.cjs"
cp "$BACKEND/.env.example" "$STAGE/.env.example"
chmod +x "$STAGE/server" "$STAGE/run.sh"

echo "==> Ensure remote dirs"
remote "mkdir -p $(printf '%q' "$REMOTE_DIR/data") $(printf '%q' "$REMOTE_DIR/logs")"

if remote "test -f $(printf '%q' "$REMOTE_DIR/.env")"; then
  echo "==> Keep existing ${REMOTE_DIR}/.env"
  SEEDED_ENV=0
else
  echo "==> Seed ${REMOTE_DIR}/.env from example (edit secrets on the server)"
  rsync -a "$STAGE/.env.example" "${DEPLOY_HOST}:${REMOTE_DIR}/.env"
  SEEDED_ENV=1
fi

echo "==> Rsync (does not touch .env, data/, logs/)"
# Trailing slashes copy CONTENTS into REMOTE_DIR (not a nested folder).
# Ship the binary to server.next then mv over the live name — scp/rsync
# into a running executable often fails with ETXTBSY ("new files never arrive").
rsync -az "$STAGE/server" "${DEPLOY_HOST}:${REMOTE_DIR}/server.next"
remote "mv -f $(printf '%q' "$REMOTE_DIR/server.next") $(printf '%q' "$REMOTE_DIR/server") && chmod +x $(printf '%q' "$REMOTE_DIR/server")"

rsync -az --delete --delay-updates \
  --exclude 'server' \
  --exclude 'server.next' \
  --exclude '.env' \
  --exclude 'data/' \
  --exclude 'logs/' \
  --exclude '*.sqlite' \
  --exclude '*.sqlite-*' \
  "${STAGE}/" "${DEPLOY_HOST}:${REMOTE_DIR}/"
remote "chmod +x $(printf '%q' "$REMOTE_DIR/run.sh")"

if [[ "$SEEDED_ENV" -eq 1 ]]; then
  echo "error: created ${REMOTE_DIR}/.env — fill bot token, jwt secret, admin ids, then re-run." >&2
  echo "  ssh ${DEPLOY_HOST}  # edit ${REMOTE_DIR}/.env" >&2
  exit 1
fi

echo "==> PM2 restart ${APP_NAME}"
# Non-interactive SSH does not load nvm from .bashrc (interactive-only guard).
# Source nvm / PM2_HOME explicitly so this talks to the same daemon as a login shell.
remote "REMOTE_DIR=$(printf '%q' "$REMOTE_DIR") APP_NAME=$(printf '%q' "$APP_NAME") bash -s" <<'REMOTE'
set -euo pipefail
export PM2_HOME="${PM2_HOME:-$HOME/.pm2}"
export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
if [[ -s "$NVM_DIR/nvm.sh" ]]; then
  # shellcheck disable=SC1091
  . "$NVM_DIR/nvm.sh"
fi
export PATH="${HOME}/.local/bin:/usr/local/bin:${PATH}"
if ! command -v pm2 >/dev/null 2>&1; then
  echo "error: pm2 not found on remote (PATH=${PATH})" >&2
  exit 1
fi
cd "$REMOTE_DIR"
if pm2 describe "$APP_NAME" >/dev/null 2>&1; then
  pm2 restart "$APP_NAME" --update-env
else
  pm2 start ecosystem.config.cjs
  pm2 save
fi
pm2 status "$APP_NAME"
sleep 1
if command -v curl >/dev/null 2>&1; then
  curl -sf --max-time 5 http://127.0.0.1:18765/health || {
    echo "error: health check failed" >&2
    pm2 logs "$APP_NAME" --lines 40 --nostream >&2 || true
    exit 1
  }
fi
REMOTE

echo
echo "Done. ${APP_NAME} @ ${DEPLOY_HOST}:${REMOTE_DIR} (http://127.0.0.1:18765)"
echo "  Public API: https://api.league.c7d5a6.com"
