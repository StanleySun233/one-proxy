#!/usr/bin/env sh
set -eu

ENV_FILE="${ENV_FILE_PATH:-./.env}"
if [ -f "$ENV_FILE" ]; then
    export $(grep -v '^\s*#' "$ENV_FILE" | grep -v '^\s*$' | xargs)
fi
if [ ! -f "$ENV_FILE" ] && [ -n "${MYSQL_DSN:-}" ]; then
    {
        echo "MYSQL_DSN=${MYSQL_DSN}"
        [ -n "${JWT_SIGNING_KEY:-}" ] && echo "JWT_SIGNING_KEY=${JWT_SIGNING_KEY}"
    } > "$ENV_FILE"
fi

HTTP_ADDR="${HTTP_ADDR:-127.0.0.1:2887}"
PORT="${PORT:-2886}"
HOSTNAME="${HOSTNAME:-0.0.0.0}"
CONTROL_PLANE_URL="${CONTROL_PLANE_URL:-http://127.0.0.1:2887}"
TZ="${TZ:-Asia/Shanghai}"

export HTTP_ADDR
export PORT
export HOSTNAME
export CONTROL_PLANE_URL
export TZ

/app/bin/one-proxy-panel &
backend_pid="$!"

cleanup() {
  kill "$backend_pid" 2>/dev/null || true
}

trap cleanup INT TERM EXIT

cd /app
exec node server.js
