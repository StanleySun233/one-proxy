#!/bin/sh
set -eu

ROOT="$(cd "$(dirname "$0")" && pwd)"
PID_FILE="$ROOT/runtime/proxy.pid"
LOG_FILE="$ROOT/runtime/proxy.log"
PYTHON_BIN="${PYTHON_BIN:-python3}"

mkdir -p "$ROOT/runtime"

if [ -f "$PID_FILE" ]; then
  old_pid="$(cat "$PID_FILE" 2>/dev/null || true)"
  if [ -n "$old_pid" ] && kill -0 "$old_pid" 2>/dev/null; then
    echo "proxy already running: $old_pid"
    exit 0
  fi
fi

cd "$ROOT"
nohup "$PYTHON_BIN" src/rootless_proxy_server.py > "$LOG_FILE" 2>&1 &
echo $! > "$PID_FILE"
echo "started: $(cat "$PID_FILE")"
