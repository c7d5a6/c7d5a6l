#!/usr/bin/env bash
# Allow this Node binary to bind :443 without full sudo (needs sudo once).
set -euo pipefail

NODE_BIN="$(command -v node)"
NODE_REAL="$(readlink -f "$NODE_BIN")"

if [[ ! -x "$NODE_REAL" ]]; then
  echo "node not found"
  exit 1
fi

echo "Granting cap_net_bind_service to:"
echo "  $NODE_REAL"
echo
sudo setcap 'cap_net_bind_service=+ep' "$NODE_REAL"
getcap "$NODE_REAL"
echo
echo "OK. Re-run: npm run dev:https"
echo "Note: after nvm install/upgrade of Node, run this script again."
