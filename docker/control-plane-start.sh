#!/usr/bin/env sh
set -eu

HTTP_ADDR="${HTTP_ADDR:-127.0.0.1:2887}"
PORT="${PORT:-2886}"
HOSTNAME="${HOSTNAME:-0.0.0.0}"
CONTROL_PLANE_URL="${CONTROL_PLANE_URL:-http://127.0.0.1:2887}"

export HTTP_ADDR
export PORT
export HOSTNAME
export CONTROL_PLANE_URL

/app/bin/control-plane &
backend_pid="$!"

cleanup() {
  kill "$backend_pid" 2>/dev/null || true
}

trap cleanup INT TERM EXIT

cd /app
exec node server.js
