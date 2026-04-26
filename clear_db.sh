#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
RUNTIME_DIR="$ROOT_DIR/apps/control-plane-api/runtime"

mkdir -p "$RUNTIME_DIR"

find "$RUNTIME_DIR" -maxdepth 1 -type f \( -name '*.db' -o -name '*.sqlite' -o -name '*.sqlite3' -o -name '*.json' \) -delete

printf 'cleared: %s\n' "$RUNTIME_DIR"
