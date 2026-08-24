#!/usr/bin/env bash
# Build the frontend and commit dist/ onto origin/gh-pages (incremental diff),
# then tag HEAD as v$VERSION.
#
# Working tree must be clean. Version is read from frontend/package.json.
# Unchanged blobs are retained; only added/removed/changed files appear in the commit.
#
# Production:
#   https://league.c7d5a6.com          (GitHub Pages custom domain)
#   https://api.league.c7d5a6.com      (API origin baked into the build)
#
# Env overrides:
#   C7D5A6L_PAGES_BASE  Vite base path (default: /)
#   VITE_API_BASE       API origin, no trailing slash
set -euo pipefail

PROD_PAGES_HOST="league.c7d5a6.com"
PROD_API_ORIGIN="https://api.league.c7d5a6.com"

FRONTEND="$(cd "$(dirname "$0")/.." && pwd)"
REPO="$(git -C "$FRONTEND" rev-parse --show-toplevel)"
cd "$REPO"

if [[ -n "$(git status --porcelain)" ]]; then
  echo "error: uncommitted content. Commit or stash before deploying." >&2
  git status --porcelain >&2
  exit 1
fi

BRANCH="$(git rev-parse --abbrev-ref HEAD)"
if [[ "$BRANCH" == "gh-pages" ]]; then
  echo "error: run this from a source branch, not gh-pages." >&2
  exit 1
fi

if ! git remote get-url origin >/dev/null 2>&1; then
  echo "error: no git remote named origin." >&2
  exit 1
fi

VERSION="$(node -p "require('$FRONTEND/package.json').version")"
if [[ -z "$VERSION" || "$VERSION" == "undefined" ]]; then
  echo "error: frontend/package.json has no version." >&2
  exit 1
fi
TAG="v${VERSION}"

echo "==> Fetch tags"
git fetch origin --tags

if git rev-parse -q --verify "refs/tags/${TAG}" >/dev/null; then
  echo "error: tag ${TAG} already exists locally." >&2
  exit 1
fi
if git ls-remote --exit-code --tags origin "refs/tags/${TAG}" >/dev/null 2>&1; then
  echo "error: tag ${TAG} already exists on origin." >&2
  exit 1
fi

BASE="${C7D5A6L_PAGES_BASE:-/}"
if [[ "$BASE" != */ ]]; then
  BASE="${BASE}/"
fi
export C7D5A6L_PAGES_BASE="$BASE"
export VITE_API_BASE="${VITE_API_BASE:-$PROD_API_ORIGIN}"

echo "==> Build frontend ${TAG} (https://${PROD_PAGES_HOST} api=${VITE_API_BASE})"
cd "$FRONTEND"
if [[ -f package-lock.json ]]; then
  npm ci
else
  npm install
fi
npm run build

cp dist/index.html dist/404.html
touch dist/.nojekyll
printf '%s\n' "$PROD_PAGES_HOST" > dist/CNAME

SHA="$(git -C "$REPO" rev-parse --short HEAD)"
TMP="$(mktemp -d)"
cleanup() {
  git -C "$REPO" worktree remove --force "$TMP" 2>/dev/null || true
  rm -rf "$TMP"
}
trap cleanup EXIT

echo "==> Fetch origin/gh-pages"
git fetch origin gh-pages:refs/remotes/origin/gh-pages 2>/dev/null || true

echo "==> Update gh-pages (incremental commit, retain unchanged files)"
if git show-ref --verify --quiet refs/remotes/origin/gh-pages; then
  git worktree add -B gh-pages "$TMP" origin/gh-pages
else
  # First publish: orphan branch, then later deploys commit diffs on top.
  git worktree add --detach "$TMP" HEAD
  git -C "$TMP" checkout --orphan gh-pages
  git -C "$TMP" rm -rf . >/dev/null 2>&1 || true
fi

# Sync build into the worktree. --delete drops stale hashed assets;
# git still keeps blob history for files whose content did not change.
rsync -a --delete \
  --exclude '.git' \
  "$FRONTEND/dist/" "$TMP/"

git -C "$TMP" add -A
if git -C "$TMP" diff --cached --quiet; then
  echo "==> No site content changes; skip gh-pages commit"
else
  git -C "$TMP" -c core.hooksPath=/dev/null -c commit.gpgsign=false \
    commit -m "Deploy frontend ${TAG} (${SHA})"
  git -C "$TMP" push origin HEAD:gh-pages
fi

echo "==> Tag ${TAG} on ${SHA}"
git -C "$REPO" tag -a "$TAG" -m "Frontend ${VERSION}"
git -C "$REPO" push origin "$TAG"

echo
echo "Done. ${TAG} → https://${PROD_PAGES_HOST} (origin/gh-pages)"
echo "  DNS: ${PROD_PAGES_HOST} CNAME → <user>.github.io"
echo "  GitHub → Settings → Pages → Custom domain ${PROD_PAGES_HOST}"
