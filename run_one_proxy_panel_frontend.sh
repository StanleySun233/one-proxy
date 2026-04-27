#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="$ROOT_DIR/apps/one-proxy-panel"
HOST="${HOST:-0.0.0.0}"
PORT="${PORT:-2886}"

cd "$APP_DIR"

export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
if [ -s "$NVM_DIR/nvm.sh" ]; then
  . "$NVM_DIR/nvm.sh"
else
  echo "missing nvm init script: $NVM_DIR/nvm.sh" >&2
  exit 1
fi

if ! nvm use 22 >/dev/null 2>&1; then
  nvm use default >/dev/null 2>&1
fi

if ! command -v npm >/dev/null 2>&1; then
  echo "npm not found after loading nvm" >&2
  exit 1
fi

if [ ! -d node_modules ]; then
  npm install
fi

rm -rf .next

export WATCHPACK_POLLING=true
export CHOKIDAR_USEPOLLING=1

exec npx next dev --hostname "$HOST" --port "$PORT"
