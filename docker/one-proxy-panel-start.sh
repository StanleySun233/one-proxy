#!/usr/bin/env sh
set -eu

ENV_FILE="${ENV_FILE_PATH:-./data/.env}"
if [ -f "$ENV_FILE" ]; then
    while IFS= read -r line || [ -n "$line" ]; do
        case "$line" in
            ''|\#*) continue ;;
        esac
        export "$line"
    done < "$ENV_FILE"
fi
if [ ! -f "$ENV_FILE" ] && [ -n "${MYSQL_DSN:-}" ]; then
    {
        echo "MYSQL_DSN=${MYSQL_DSN}"
        [ -n "${JWT_SIGNING_KEY:-}" ] && echo "JWT_SIGNING_KEY=${JWT_SIGNING_KEY}"
    } > "$ENV_FILE"
fi

HTTP_ADDR="${HTTP_ADDR:-127.0.0.1:2887}"
PUBLIC_PORT="${PORT:-2886}"
WEB_PORT="${WEB_PORT:-2885}"
EDGE_ADDR="${EDGE_ADDR:-:${PUBLIC_PORT}}"
EDGE_WEB_URL="${EDGE_WEB_URL:-http://127.0.0.1:${WEB_PORT}}"
EDGE_API_URL="${EDGE_API_URL:-http://127.0.0.1:2887}"
GUACD_ADDR="${GUACD_ADDR:-guacd:4822}"
HOSTNAME="0.0.0.0"
CONTROL_PLANE_URL="${CONTROL_PLANE_URL:-http://127.0.0.1:2887}"
TZ="${TZ:-Asia/Shanghai}"

export HTTP_ADDR
export EDGE_ADDR
export EDGE_WEB_URL
export EDGE_API_URL
export GUACD_ADDR
export HOSTNAME
export CONTROL_PLANE_URL
export TZ

guacd_pid=""
embedded_guacd=0
case "${EMBEDDED_GUACD:-auto}" in
  1|true|yes)
    embedded_guacd=1
    ;;
  auto)
    case "$GUACD_ADDR" in
      127.0.0.1:4822|localhost:4822)
        if command -v guacd >/dev/null 2>&1; then
          embedded_guacd=1
        fi
        ;;
    esac
    ;;
esac
if [ "$embedded_guacd" = "1" ]; then
  guacd_bin="$(command -v guacd || true)"
  if [ -z "$guacd_bin" ]; then
    echo "embedded_guacd_unavailable" >&2
    exit 1
  fi
  "$guacd_bin" -b 127.0.0.1 -l 4822 -f &
  guacd_pid="$!"
fi
/app/bin/one-proxy-panel &
backend_pid="$!"
frontend_pid=""
edge_pid=""

cleanup() {
  [ -n "$guacd_pid" ] && kill "$guacd_pid" 2>/dev/null || true
  [ -n "$backend_pid" ] && kill "$backend_pid" 2>/dev/null || true
  [ -n "$frontend_pid" ] && kill "$frontend_pid" 2>/dev/null || true
  [ -n "$edge_pid" ] && kill "$edge_pid" 2>/dev/null || true
}

capture_wait() {
  set +e
  wait "$1"
  wait_status="$?"
  set -e
}

trap cleanup INT TERM EXIT

cd /app
PORT="$WEB_PORT"
export PORT
node server.js &
frontend_pid="$!"
/app/bin/one-proxy-panel-edge &
edge_pid="$!"

while :; do
  if [ -n "$guacd_pid" ] && ! kill -0 "$guacd_pid" 2>/dev/null; then
    capture_wait "$guacd_pid"
    status="$wait_status"
    cleanup
    exit "$status"
  fi
  if ! kill -0 "$backend_pid" 2>/dev/null; then
    capture_wait "$backend_pid"
    status="$wait_status"
    cleanup
    exit "$status"
  fi
  if ! kill -0 "$frontend_pid" 2>/dev/null; then
    capture_wait "$frontend_pid"
    status="$wait_status"
    cleanup
    exit "$status"
  fi
  if ! kill -0 "$edge_pid" 2>/dev/null; then
    capture_wait "$edge_pid"
    status="$wait_status"
    cleanup
    exit "$status"
  fi
  sleep 1
done
