#!/usr/bin/env bash
# One-time local HTTPS for Telegram Login Widget on https://c7d5a6l.lo
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CERT_DIR="$ROOT/certs"
HOST="c7d5a6l.lo"

echo "==> Hosts file"
if grep -qE '^[[:space:]]*127\.0\.0\.1[[:space:]]+.*\b'"$HOST"'\b' /etc/hosts 2>/dev/null; then
  echo "    $HOST already maps in /etc/hosts"
else
  echo "    Add this line (needs sudo):"
  echo "      127.0.0.1  $HOST"
  echo "    Example:"
  echo "      echo '127.0.0.1  $HOST' | sudo tee -a /etc/hosts"
  exit 1
fi

echo "==> mkcert"
if ! command -v mkcert >/dev/null 2>&1; then
  echo "    Install mkcert first, e.g.:"
  echo "      # Debian/Ubuntu (example)"
  echo "      sudo apt install libnss3-tools"
  echo "      # then install mkcert from https://github.com/FiloSottile/mkcert"
  echo "      mkcert -install"
  exit 1
fi

mkdir -p "$CERT_DIR"
if [[ ! -f "$CERT_DIR/$HOST.pem" || ! -f "$CERT_DIR/$HOST-key.pem" ]]; then
  echo "    Generating certs in $CERT_DIR"
  (cd "$CERT_DIR" && mkcert "$HOST")
else
  echo "    Certs already present in $CERT_DIR"
fi

echo "==> Port 443"
NODE_REAL="$(readlink -f "$(command -v node)")"
if [[ "$(id -u)" -eq 0 ]]; then
  echo "    Running as root — fine for binding :443"
elif getcap "$NODE_REAL" 2>/dev/null | grep -q 'cap_net_bind_service'; then
  echo "    Node already has cap_net_bind_service ($NODE_REAL)"
else
  echo "    EACCES on :443 is expected until you allow the Node binary:"
  echo "      npm run allow:https"
  echo "    (uses sudo setcap on $NODE_REAL — re-run after nvm upgrades Node)"
fi

echo
echo "Done. Next:"
echo "  1. BotFather → /setdomain → $HOST"
echo "  2. Restart backend"
echo "  3. npm run allow:https   # once, if :443 is denied"
echo "  4. npm run dev:https"
echo "  5. Open https://$HOST/"
