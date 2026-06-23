#!/usr/bin/env bash
set -euo pipefail

mode="${1:-check}"
tag="${2:-${ONEPROXY_IMAGE_TAG:-}}"
ssh_host="${CAMELBOT_SSH_HOST:-camelbot}"
panel_repo="${ONEPROXY_PANEL_IMAGE_REPO:-ghcr.io/stanleysun233/oneproxy-panel}"
node_repo="${ONEPROXY_NODE_IMAGE_REPO:-ghcr.io/stanleysun233/oneproxy-node}"
panel_container="${ONEPROXY_PANEL_CONTAINER:-one-proxy-panel}"
panel_network="${ONEPROXY_PANEL_NETWORK:-one-proxy-net}"
panel_port="${ONEPROXY_PANEL_PORT:-2886}"
mysql_container="${ONEPROXY_MYSQL_CONTAINER:-one-proxy-mysql8}"
final_db="${ONEPROXY_FINAL_PANEL_DB_NAME:-one_proxy_v210_final}"
final_panel_volume="${ONEPROXY_FINAL_PANEL_DATA_VOLUME:-one-proxy-panel-data-v210-final}"
final_remote_node_volume="${ONEPROXY_FINAL_CAMELBOT_NODE_VOLUME:-one-proxy-node-runtime-v210-final}"
final_local_node_volume="${ONEPROXY_FINAL_LOCAL_NODE_VOLUME:-one-proxy-node-runtime-v210-final}"
remote_node_container="${ONEPROXY_CAMELBOT_NODE_CONTAINER:-one-proxy-node}"
local_node_container="${ONEPROXY_LOCAL_NODE_CONTAINER:-one-proxy-node}"
local_node_network="${ONEPROXY_LOCAL_NODE_NETWORK:-}"
admin_password="${ONEPROXY_FINAL_PANEL_ADMIN_PASSWORD:-}"
jwt_signing_key="${ONEPROXY_FINAL_PANEL_JWT_SIGNING_KEY:-}"
confirm="${ONEPROXY_FINAL_SCHEMA_CONFIRM:-}"
tenant_name="${ONEPROXY_FINAL_TENANT_NAME:-OneProxy v2.1.0}"
scope_name="${ONEPROXY_FINAL_SCOPE_NAME:-v2.1.0 final scope}"
local_node_name="${ONEPROXY_FINAL_LOCAL_NODE_NAME:-hk-public-node}"
remote_node_name="${ONEPROXY_FINAL_CAMELBOT_NODE_NAME:-sg-astar-58}"
local_node_parent_url="${ONEPROXY_FINAL_LOCAL_NODE_PARENT_URL:-}"
remote_node_parent_url="${ONEPROXY_FINAL_CAMELBOT_NODE_PARENT_URL:-http://${panel_container}:2886}"
local_node_public_host="${ONEPROXY_FINAL_LOCAL_NODE_PUBLIC_HOST:-127.0.0.1}"
remote_node_public_host="${ONEPROXY_FINAL_CAMELBOT_NODE_PUBLIC_HOST:-127.0.0.1}"
local_node_http_port="${ONEPROXY_FINAL_LOCAL_NODE_HTTP_PORT:-2988}"
remote_node_http_port="${ONEPROXY_FINAL_CAMELBOT_NODE_HTTP_PORT:-2988}"
local_node_direct_port="${ONEPROXY_FINAL_LOCAL_NODE_DIRECT_PORT:-2992}"
remote_node_direct_port="${ONEPROXY_FINAL_CAMELBOT_NODE_DIRECT_PORT:-2992}"

case "$mode" in
  check|dry-run|verify|run)
    ;;
  *)
    echo "usage: $0 [check|dry-run|verify|run] <immutable_tag>" >&2
    exit 2
    ;;
esac

require_tag() {
  if [ -z "$tag" ]; then
    echo "immutable image tag is required" >&2
    exit 2
  fi
  case "$tag" in
    latest|main|master|dev|nightly|stable)
      echo "mutable image tag is not allowed: $tag" >&2
      exit 2
      ;;
  esac
}

panel_image() {
  printf "%s:%s" "$panel_repo" "$tag"
}

node_image() {
  printf "%s:%s" "$node_repo" "$tag"
}

current_panel_value() {
  key="$1"
  ssh -T "$ssh_host" 'bash -s' -- "$panel_container" "$key" <<'REMOTE'
set -euo pipefail
container="$1"
key="$2"
value="$(docker exec -e LOOKUP_KEY="$key" "$container" sh -lc 'if [ -f /app/data/.env ]; then awk -F= -v key="$LOOKUP_KEY" '\''$1 == key {sub(/^[^=]*=/, "", $0); print; exit}'\'' /app/data/.env; fi' 2>/dev/null || true)"
if [ -z "$value" ]; then
  value="$(docker inspect "$container" --format '{{range .Config.Env}}{{println .}}{{end}}' 2>/dev/null | awk -F= -v key="$key" '$1 == key {sub(/^[^=]*=/, "", $0); print; exit}')"
fi
printf "%s" "$value"
REMOTE
}

infer_local_node_parent_url() {
  host="$(ssh -G "$ssh_host" 2>/dev/null | awk '/^hostname /{print $2; exit}')"
  if [ -z "$host" ]; then
    return 0
  fi
  candidate="http://${host}:${panel_port}"
  if curl -fsS "${candidate}/healthz" >/dev/null 2>&1; then
    printf "%s" "$candidate"
  fi
}

resolve_run_env() {
  if [ -z "$admin_password" ]; then
    admin_password="$(current_panel_value ADMIN_PASSWORD)"
  fi
  if [ -z "$jwt_signing_key" ]; then
    jwt_signing_key="$(current_panel_value JWT_SIGNING_KEY)"
  fi
  if [ -z "$local_node_parent_url" ]; then
    local_node_parent_url="$(infer_local_node_parent_url)"
  fi
}

require_run_env() {
  require_tag
  if [ "$confirm" != "deploy-final-schema" ]; then
    echo "run requires ONEPROXY_FINAL_SCHEMA_CONFIRM=deploy-final-schema" >&2
    exit 2
  fi
  resolve_run_env
  if [ -z "$admin_password" ] || [ -z "$jwt_signing_key" ] || [ -z "$local_node_parent_url" ]; then
    echo "run requires final panel secrets and ONEPROXY_FINAL_LOCAL_NODE_PARENT_URL, or reusable existing panel secrets plus a reachable camelbot panel URL" >&2
    exit 2
  fi
}

print_plan() {
  echo "mode=${mode}"
  echo "tag=${tag:-<required-for-dry-run-run>}"
  if [ -n "$tag" ]; then
    echo "panel_image=$(panel_image)"
    echo "node_image=$(node_image)"
  else
    echo "panel_image=<required-for-dry-run-run>"
    echo "node_image=<required-for-dry-run-run>"
  fi
  echo "camelbot=${ssh_host}"
  echo "final_db=${final_db}"
  echo "final_panel_volume=${final_panel_volume}"
  echo "final_remote_node_volume=${final_remote_node_volume}"
  echo "final_local_node_volume=${final_local_node_volume}"
  echo "tenant_name=${tenant_name}"
  echo "local_node=${local_node_name}"
  echo "remote_node=${remote_node_name}"
  display_local_parent_url="$local_node_parent_url"
  if [ -z "$display_local_parent_url" ]; then
    display_local_parent_url="$(infer_local_node_parent_url)"
  fi
  echo "local_node_parent_url=${display_local_parent_url:-<required-for-run>}"
  echo "remote_node_parent_url=${remote_node_parent_url}"
}

check_local() {
  echo "target=local-node"
  if command -v docker >/dev/null 2>&1; then
    docker inspect "$local_node_container" --format 'current={{.Config.Image}} status={{.State.Status}} started={{.State.StartedAt}}' 2>/dev/null || echo "current=missing"
  else
    echo "docker=missing"
  fi
}

check_remote() {
  ssh -T "$ssh_host" 'bash -s' -- "$panel_container" "$panel_port" "$mysql_container" "$final_db" "$remote_node_container" <<'REMOTE'
set -euo pipefail
panel_container="$1"
panel_port="$2"
mysql_container="$3"
final_db="$4"
remote_node_container="$5"
echo "target=camelbot-panel"
docker inspect "$panel_container" --format 'current={{.Config.Image}} status={{.State.Status}} started={{.State.StartedAt}}' 2>/dev/null || echo "current=missing"
curl -fsS "http://127.0.0.1:${panel_port}/healthz" >/dev/null 2>&1 && echo "health=ok" || echo "health=unavailable"
echo "target=camelbot-node"
docker inspect "$remote_node_container" --format 'current={{.Config.Image}} status={{.State.Status}} started={{.State.StartedAt}}' 2>/dev/null || echo "current=missing"
if docker inspect "$mysql_container" >/dev/null 2>&1; then
  final_db_literal="$(printf "%s" "$final_db" | sed "s/'/''/g")"
  tables="$(docker exec -e MYSQL_QUERY="SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '${final_db_literal}';" "$mysql_container" sh -lc 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -N -B -e "$MYSQL_QUERY"' 2>/dev/null || echo unknown)"
  echo "final_db=${final_db} tables=${tables}"
else
  echo "mysql=missing"
fi
REMOTE
}

verify_final_state() {
  require_tag
  expected_panel_image="$(panel_image)"
  expected_node_image="$(node_image)"
  status=0
  echo "target=local-node"
  local_image="$(docker inspect "$local_node_container" --format '{{.Config.Image}}' 2>/dev/null || true)"
  local_health="$(curl -fsS http://127.0.0.1:2988/healthz 2>/dev/null || true)"
  echo "image=${local_image:-missing}"
  echo "health=${local_health:-unavailable}"
  if [ "$local_image" != "$expected_node_image" ] || ! printf "%s" "$local_health" | grep -q '"controlPlaneBound":true'; then
    status=1
  fi
  if ! ssh -T "$ssh_host" 'bash -s' -- "$panel_container" "$remote_node_container" "$panel_port" "$mysql_container" "$final_db" "$expected_panel_image" "$expected_node_image" <<'REMOTE'
set -euo pipefail
panel_container="$1"
remote_node_container="$2"
panel_port="$3"
mysql_container="$4"
final_db="$5"
expected_panel_image="$6"
expected_node_image="$7"
status=0
query() {
  docker exec -e MYSQL_QUERY="$1" -e MYSQL_DATABASE="$final_db" "$mysql_container" sh -lc 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" "$MYSQL_DATABASE" -N -B -e "$MYSQL_QUERY"' 2>/dev/null || true
}
echo "target=camelbot-panel"
panel_image="$(docker inspect "$panel_container" --format '{{.Config.Image}}' 2>/dev/null || true)"
panel_health="$(curl -fsS "http://127.0.0.1:${panel_port}/healthz" 2>/dev/null || true)"
echo "image=${panel_image:-missing}"
echo "health=${panel_health:-unavailable}"
if [ "$panel_image" != "$expected_panel_image" ] || [ -z "$panel_health" ]; then
  status=1
fi
echo "target=camelbot-node"
node_image="$(docker inspect "$remote_node_container" --format '{{.Config.Image}}' 2>/dev/null || true)"
node_health="$(curl -fsS http://127.0.0.1:2988/healthz 2>/dev/null || true)"
echo "image=${node_image:-missing}"
echo "health=${node_health:-unavailable}"
if [ "$node_image" != "$expected_node_image" ] || ! printf "%s" "$node_health" | grep -q '"controlPlaneBound":true'; then
  status=1
fi
echo "target=final-db"
tables="$(query "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE();")"
goose="$(query "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'goose_db_version';")"
nodes="$(query "SELECT COUNT(*) FROM nodes;")"
access_paths="$(query "SELECT COUNT(*) FROM node_access_paths;")"
routes="$(query "SELECT COUNT(*) FROM route_rules;")"
policies="$(query "SELECT COUNT(*) FROM policy_revisions;")"
session_hashes="$(query "SELECT COUNT(*) FROM sessions WHERE access_token_hash REGEXP '^[0-9a-f]{64}$' AND refresh_token_hash REGEXP '^[0-9a-f]{64}$';")"
node_token_hashes="$(query "SELECT COUNT(*) FROM node_api_tokens WHERE token_hash REGEXP '^[0-9a-f]{64}$';")"
bootstrap_token_hashes="$(query "SELECT COUNT(*) FROM bootstrap_tokens WHERE token_hash REGEXP '^[0-9a-f]{64}$';")"
echo "db=${final_db} tables=${tables:-unknown} goose_tables=${goose:-unknown} nodes=${nodes:-unknown} access_paths=${access_paths:-unknown} routes=${routes:-unknown} policies=${policies:-unknown}"
echo "hash_shapes=sessions:${session_hashes:-unknown} node_tokens:${node_token_hashes:-unknown} bootstrap_tokens:${bootstrap_token_hashes:-unknown}"
query "SELECT id, status, enabled, public_host, public_port FROM nodes ORDER BY id;"
query "SELECT node_id, transport_type, direction, address, status FROM node_transports ORDER BY node_id, transport_type;"
query "SELECT id, chain_id, entry_node_id, target_node_id, listen_port, target_host, target_port, enabled FROM node_access_paths ORDER BY id;"
query "SELECT id, priority, match_type, match_value, action_type, chain_id, destination_scope, enabled FROM route_rules ORDER BY priority, id;"
if [ "${goose:-1}" != "0" ] || [ "${nodes:-0}" -lt 2 ] || [ "${access_paths:-0}" -lt 2 ] || [ "${routes:-0}" -lt 2 ] || [ "${policies:-0}" -lt 1 ]; then
  status=1
fi
exit "$status"
REMOTE
  then
    status=1
  fi
  return "$status"
}

json_get() {
  python3 -c 'import json,sys
path=sys.argv[1].split(".")
obj=json.load(sys.stdin)
for part in path:
    if part == "":
        continue
    if isinstance(obj, list):
        obj = obj[int(part)] if part.isdigit() and int(part) < len(obj) else None
    elif isinstance(obj, dict):
        obj = obj.get(part)
    else:
        obj = None
    if obj is None:
        break
if isinstance(obj, (dict, list)):
    print(json.dumps(obj, separators=(",", ":")))
elif obj is not None:
    print(obj)
' "$1"
}

deploy_panel_and_remote_node() {
  ssh -T "$ssh_host" 'bash -s' -- \
    "$(panel_image)" "$(node_image)" "$panel_container" "$panel_network" "$panel_port" "$mysql_container" "$final_db" "$final_panel_volume" \
    "$admin_password" "$jwt_signing_key" "$remote_node_container" "$final_remote_node_volume" "$remote_node_name" "$remote_node_parent_url" \
    "$remote_node_public_host" "$remote_node_http_port" "$remote_node_direct_port" "$tenant_name" "$scope_name" "$local_node_name" "$local_node_public_host" "$local_node_http_port" <<'REMOTE'
set -Eeuo pipefail
panel_image="$1"
node_image="$2"
panel_container="$3"
panel_network="$4"
panel_port="$5"
mysql_container="$6"
final_db="$7"
panel_volume="$8"
admin_password="$9"
jwt_signing_key="${10}"
remote_node_container="${11}"
remote_node_volume="${12}"
remote_node_name="${13}"
remote_node_parent_url="${14}"
remote_node_public_host="${15}"
remote_node_http_port="${16}"
remote_node_direct_port="${17}"
tenant_name="${18}"
scope_name="${19}"
local_node_name="${20}"
local_node_public_host="${21}"
local_node_http_port="${22}"
panel_backup="${panel_container}-prev-$(date +%Y%m%d%H%M%S)"
remote_node_backup="${remote_node_container}-prev-$(date +%Y%m%d%H%M%S)"
panel_env="$(mktemp)"
node_env="$(mktemp)"
panel_renamed=0
remote_node_renamed=0
completed=0
cleanup() {
  rm -f "$panel_env" "$node_env"
}
rollback() {
  status=$?
  if [ "$completed" -eq 0 ]; then
    if [ "$remote_node_renamed" -eq 1 ]; then
      docker rm -f "$remote_node_container" >/dev/null 2>&1 || true
      if docker ps -a --format '{{.Names}}' | grep -Fxq "$remote_node_backup"; then
        docker rename "$remote_node_backup" "$remote_node_container" >/dev/null 2>&1 || true
        docker start "$remote_node_container" >/dev/null 2>&1 || true
      fi
    fi
    if [ "$panel_renamed" -eq 1 ]; then
      docker rm -f "$panel_container" >/dev/null 2>&1 || true
      if docker ps -a --format '{{.Names}}' | grep -Fxq "$panel_backup"; then
        docker rename "$panel_backup" "$panel_container" >/dev/null 2>&1 || true
        docker start "$panel_container" >/dev/null 2>&1 || true
      fi
    fi
  fi
  cleanup
  exit "$status"
}
trap cleanup EXIT
trap rollback ERR HUP INT TERM

json_get() {
  python3 -c 'import json,sys
path=sys.argv[1].split(".")
obj=json.load(sys.stdin)
for part in path:
    if part == "":
        continue
    if isinstance(obj, list):
        obj = obj[int(part)] if part.isdigit() and int(part) < len(obj) else None
    elif isinstance(obj, dict):
        obj = obj.get(part)
    else:
        obj = None
    if obj is None:
        break
if isinstance(obj, (dict, list)):
    print(json.dumps(obj, separators=(",", ":")))
elif obj is not None:
    print(obj)
' "$1"
}

node_id_by_name() {
  python3 -c 'import json,sys
name=sys.argv[1]
payload=json.load(sys.stdin).get("data") or []
for item in payload:
    if str(item.get("name", "")) == name:
        print(item.get("id", ""))
        break
' "$1"
}

mysql_admin() {
  docker exec -e MYSQL_QUERY="$1" "$mysql_container" sh -lc 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -N -B -e "$MYSQL_QUERY"'
}

api_request() {
  method="$1"
  path="$2"
  access_token="${3:-}"
  tenant_id="${4:-}"
  body="${5:-}"
  headers=(-H "Content-Type: application/json")
  if [ -n "$access_token" ]; then
    headers+=(-H "X-One-Proxy-Access-Token: ${access_token}")
  fi
  if [ -n "$tenant_id" ]; then
    headers+=(-H "X-One-Proxy-Tenant-ID: ${tenant_id}")
  fi
  if [ -n "$body" ]; then
    curl -fsS -X "$method" "http://127.0.0.1:${panel_port}/api${path}" "${headers[@]}" -d "$body"
  else
    curl -fsS -X "$method" "http://127.0.0.1:${panel_port}/api${path}" "${headers[@]}"
  fi
}

wait_for_http() {
  url="$1"
  label="$2"
  tries=0
  until curl -fsS "$url" >/dev/null 2>&1; do
    tries=$((tries + 1))
    if [ "$tries" -ge 60 ]; then
      echo "${label}=failed" >&2
      exit 1
    fi
    sleep 2
  done
  echo "${label}=ok"
}

wait_pending_node() {
  name="$1"
  tries=0
  while :; do
    response="$(api_request GET /nodes/pending "$access_token" "$tenant_id")"
    found="$(printf "%s" "$response" | node_id_by_name "$name")"
    if [ -n "$found" ]; then
      printf "%s" "$found"
      return 0
    fi
    tries=$((tries + 1))
    if [ "$tries" -ge 90 ]; then
      echo "pending_node=${name}=failed" >&2
      exit 1
    fi
    sleep 2
  done
}

escaped_db="$(printf "%s" "$final_db" | sed 's/`/``/g')"
escaped_db_literal="$(printf "%s" "$final_db" | sed "s/'/''/g")"
mysql_admin "CREATE DATABASE IF NOT EXISTS \`${escaped_db}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
table_count="$(mysql_admin "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '${escaped_db_literal}';")"
if [ "$table_count" != "0" ]; then
  echo "final schema database is not empty: ${final_db} tables=${table_count}" >&2
  exit 2
fi
mysql_root_password="$(docker exec "$mysql_container" sh -lc 'printf "%s" "$MYSQL_ROOT_PASSWORD"')"
{
  echo "MYSQL_DSN=root:${mysql_root_password}@tcp(${mysql_container}:3306)/${final_db}?charset=utf8mb4&parseTime=true&loc=UTC"
  echo "ADMIN_PASSWORD=${admin_password}"
  echo "JWT_SIGNING_KEY=${jwt_signing_key}"
  echo "CONTROL_PLANE_URL=http://127.0.0.1:2887"
} > "$panel_env"

docker pull "$panel_image" >/dev/null
if docker inspect "$panel_container" >/dev/null 2>&1; then
  docker rename "$panel_container" "$panel_backup"
  panel_renamed=1
  docker stop "$panel_backup" >/dev/null
  echo "panel_backup=${panel_backup}"
fi
docker run -d \
  --name "$panel_container" \
  --restart unless-stopped \
  --network "$panel_network" \
  --env-file "$panel_env" \
  -v "${panel_volume}:/app/data" \
  -p "${panel_port}:2886" \
  "$panel_image" >/dev/null
wait_for_http "http://127.0.0.1:${panel_port}/healthz" "panel_health"

login_response="$(api_request POST /auth/login "" "" "{\"account\":\"admin\",\"password\":\"${admin_password}\"}")"
access_token="$(printf "%s" "$login_response" | json_get data.accessToken)"
account_id="$(printf "%s" "$login_response" | json_get data.account.id)"
tenant_response="$(api_request POST /tenants "$access_token" "" "{\"name\":\"${tenant_name}\",\"initialAdminAccountId\":\"${account_id}\"}")"
tenant_id="$(printf "%s" "$tenant_response" | json_get data.tenant.id)"
scope_response="$(api_request POST /proxy/scopes "$access_token" "$tenant_id" "{\"name\":\"${scope_name}\",\"description\":\"final-schema standing cutover\"}")"
scope_id="$(printf "%s" "$scope_response" | json_get data.id)"
remote_bootstrap_response="$(api_request POST /nodes/bootstrap/token "$access_token" "$tenant_id" "{\"targetType\":\"node\",\"nodeName\":\"${remote_node_name}\",\"nodeMode\":\"edge\",\"scopeKey\":\"${scope_id}\",\"publicHost\":\"${remote_node_public_host}\",\"publicPort\":${remote_node_http_port}}")"
remote_bootstrap_token="$(printf "%s" "$remote_bootstrap_response" | json_get data.token)"
local_bootstrap_response="$(api_request POST /nodes/bootstrap/token "$access_token" "$tenant_id" "{\"targetType\":\"node\",\"nodeName\":\"${local_node_name}\",\"nodeMode\":\"edge\",\"scopeKey\":\"${scope_id}\",\"publicHost\":\"${local_node_public_host}\",\"publicPort\":${local_node_http_port}}")"
local_bootstrap_token="$(printf "%s" "$local_bootstrap_response" | json_get data.token)"

{
  echo "NODE_BOOTSTRAP_TOKEN=${remote_bootstrap_token}"
  echo "NODE_PARENT_URL=${remote_node_parent_url}"
  echo "NODE_NAME=${remote_node_name}"
  echo "NODE_MODE=edge"
  echo "NODE_PUBLIC_HOST=${remote_node_public_host}"
  echo "NODE_LISTEN_ADDR=:2988"
  echo "NODE_HTTPS_LISTEN_ADDR=:2989"
  echo "NODE_TCP_ACCESS_LISTEN_ADDR=:2990"
  echo "NODE_TCP_ACCESS_MAX_SESSIONS=4096"
  echo "NODE_UDP_ACCESS_LISTEN_ADDR=:2991"
  echo "NODE_UDP_ACCESS_MAX_IN_FLIGHT=1024"
  echo "NODE_UDP_ACCESS_TIMEOUT=15s"
  echo "NODE_DIRECT_LISTEN_ADDR=:2992"
  echo "NODE_DIRECT_STUN_SERVERS="
  echo "NODE_DIRECT_REFRESH_INTERVAL=5s"
  echo "NODE_FORWARD_RETRY_BODY_MAX_BYTES=8mb"
  echo "NODE_HEARTBEAT_INTERVAL=2s"
  echo "NODE_POLICY_STATE_PATH=runtime/node-policy-state.json"
  echo "NODE_RUNTIME_CONFIG_PATH=runtime/node-runtime.json"
} > "$node_env"

docker pull "$node_image" >/dev/null
if docker inspect "$remote_node_container" >/dev/null 2>&1; then
  docker rename "$remote_node_container" "$remote_node_backup"
  remote_node_renamed=1
  docker stop "$remote_node_backup" >/dev/null
  echo "remote_node_backup=${remote_node_backup}"
fi
docker run -d \
  --name "$remote_node_container" \
  --restart unless-stopped \
  --network "$panel_network" \
  --env-file "$node_env" \
  -v "${remote_node_volume}:/app/runtime" \
  -p 2988:2988 \
  -p 2989:2989 \
  -p 2990:2990 \
  -p 2991:2991/udp \
  -p 2992:2992/udp \
  "$node_image" >/dev/null
remote_node_id="$(wait_pending_node "$remote_node_name")"
remote_approve_response="$(api_request POST "/nodes/${remote_node_id}/approve" "$access_token" "$tenant_id" "{}")"
remote_node_token="$(printf "%s" "$remote_approve_response" | json_get data.accessToken)"
tries=0
until curl -fsS http://127.0.0.1:2988/healthz 2>/dev/null | grep -q '"controlPlaneBound":true'; do
  tries=$((tries + 1))
  if [ "$tries" -ge 60 ]; then
    echo "remote_node_bound=failed" >&2
    exit 1
  fi
  sleep 2
done
echo "remote_node_bound=ok"
completed=1
trap - ERR HUP INT TERM
echo "tenant_id=${tenant_id}"
echo "scope_id=${scope_id}"
echo "remote_node_id=${remote_node_id}"
echo "remote_node_token=${remote_node_token}"
echo "local_bootstrap_token=${local_bootstrap_token}"
REMOTE
}

deploy_local_node() {
  local_bootstrap_token="$1"
  image="$(node_image)"
  env_file="$(mktemp)"
  backup="${local_node_container}-prev-$(date +%Y%m%d%H%M%S)"
  renamed=0
  completed=0
  network="$local_node_network"
  if [ -z "$network" ] && docker inspect "$local_node_container" >/dev/null 2>&1; then
    network="$(docker inspect "$local_node_container" --format '{{range $name, $_ := .NetworkSettings.Networks}}{{println $name}}{{end}}' | head -n 1)"
  fi
  network_args=()
  if [ -n "$network" ]; then
    network_args=(--network "$network")
  fi
  cleanup() {
    rm -f "$env_file"
  }
  rollback() {
    status=$?
    if [ "$completed" -eq 0 ] && [ "$renamed" -eq 1 ]; then
      docker rm -f "$local_node_container" >/dev/null 2>&1 || true
      if docker ps -a --format '{{.Names}}' | grep -Fxq "$backup"; then
        docker rename "$backup" "$local_node_container" >/dev/null 2>&1 || true
        docker start "$local_node_container" >/dev/null 2>&1 || true
      fi
    fi
    cleanup
    exit "$status"
  }
  trap cleanup EXIT
  trap rollback ERR HUP INT TERM
  {
    echo "NODE_BOOTSTRAP_TOKEN=${local_bootstrap_token}"
    echo "NODE_PARENT_URL=${local_node_parent_url}"
    echo "NODE_NAME=${local_node_name}"
    echo "NODE_MODE=edge"
    echo "NODE_PUBLIC_HOST=${local_node_public_host}"
    echo "NODE_LISTEN_ADDR=:2988"
    echo "NODE_HTTPS_LISTEN_ADDR=:2989"
    echo "NODE_TCP_ACCESS_LISTEN_ADDR=:2990"
    echo "NODE_TCP_ACCESS_MAX_SESSIONS=4096"
    echo "NODE_UDP_ACCESS_LISTEN_ADDR=:2991"
    echo "NODE_UDP_ACCESS_MAX_IN_FLIGHT=1024"
    echo "NODE_UDP_ACCESS_TIMEOUT=15s"
    echo "NODE_DIRECT_LISTEN_ADDR=:2992"
    echo "NODE_DIRECT_STUN_SERVERS="
    echo "NODE_DIRECT_REFRESH_INTERVAL=5s"
    echo "NODE_FORWARD_RETRY_BODY_MAX_BYTES=8mb"
    echo "NODE_HEARTBEAT_INTERVAL=2s"
    echo "NODE_POLICY_STATE_PATH=runtime/node-policy-state.json"
    echo "NODE_RUNTIME_CONFIG_PATH=runtime/node-runtime.json"
  } > "$env_file"
  docker pull "$image" >/dev/null
  if docker inspect "$local_node_container" >/dev/null 2>&1; then
    docker rename "$local_node_container" "$backup"
    renamed=1
    docker stop "$backup" >/dev/null
    echo "local_node_backup=${backup}"
  fi
  docker run -d \
    --name "$local_node_container" \
    --restart unless-stopped \
    "${network_args[@]}" \
    --env-file "$env_file" \
    -v "${final_local_node_volume}:/app/runtime" \
    -p 2988:2988 \
    -p 2989:2989 \
    -p 2990:2990 \
    -p 2991:2991/udp \
    -p 2992:2992/udp \
    "$image" >/dev/null
  completed=1
  trap - ERR HUP INT TERM
  trap - EXIT
  cleanup
}

restore_remote_backups() {
  panel_backup="$1"
  remote_node_backup="$2"
  ssh -T "$ssh_host" 'bash -s' -- "$panel_container" "$panel_backup" "$remote_node_container" "$remote_node_backup" <<'REMOTE'
set -euo pipefail
panel_container="$1"
panel_backup="$2"
remote_node_container="$3"
remote_node_backup="$4"
restore_container() {
  current="$1"
  backup="$2"
  if [ -z "$backup" ]; then
    return 0
  fi
  if docker ps -a --format '{{.Names}}' | grep -Fxq "$backup"; then
    docker rm -f "$current" >/dev/null 2>&1 || true
    docker rename "$backup" "$current" >/dev/null
    docker start "$current" >/dev/null
    echo "restored=${current}"
  fi
}
restore_container "$remote_node_container" "$remote_node_backup"
restore_container "$panel_container" "$panel_backup"
REMOTE
}

restore_local_backup() {
  local_backup="$1"
  if [ -z "$local_backup" ]; then
    return 0
  fi
  if docker ps -a --format '{{.Names}}' | grep -Fxq "$local_backup"; then
    docker rm -f "$local_node_container" >/dev/null 2>&1 || true
    docker rename "$local_backup" "$local_node_container" >/dev/null
    docker start "$local_node_container" >/dev/null
    echo "restored=${local_node_container}"
  fi
}

finish_control_plane_state() {
  tenant_id="$1"
  scope_id="$2"
  remote_node_id="$3"
  remote_node_token="$4"
  ssh -T "$ssh_host" 'bash -s' -- "$panel_port" "$admin_password" "$tenant_id" "$scope_id" "$local_node_name" "$remote_node_id" "$remote_node_token" "$local_node_http_port" "$remote_node_http_port" "$mysql_container" "$final_db" <<'REMOTE'
set -Eeuo pipefail
panel_port="$1"
admin_password="$2"
tenant_id="$3"
scope_id="$4"
local_node_name="$5"
remote_node_id="$6"
remote_node_token="$7"
local_node_http_port="$8"
remote_node_http_port="$9"
mysql_container="${10}"
final_db="${11}"

json_get() {
  python3 -c 'import json,sys
path=sys.argv[1].split(".")
obj=json.load(sys.stdin)
for part in path:
    if part == "":
        continue
    if isinstance(obj, list):
        obj = obj[int(part)] if part.isdigit() and int(part) < len(obj) else None
    elif isinstance(obj, dict):
        obj = obj.get(part)
    else:
        obj = None
    if obj is None:
        break
if isinstance(obj, (dict, list)):
    print(json.dumps(obj, separators=(",", ":")))
elif obj is not None:
    print(obj)
' "$1"
}

node_id_by_name() {
  python3 -c 'import json,sys
name=sys.argv[1]
payload=json.load(sys.stdin).get("data") or []
for item in payload:
    if str(item.get("name", "")) == name:
        print(item.get("id", ""))
        break
' "$1"
}

sha256_hex() {
  python3 -c 'import hashlib,sys; print(hashlib.sha256(sys.argv[1].encode()).hexdigest())' "$1"
}

api_request() {
  method="$1"
  path="$2"
  access_token="${3:-}"
  tenant_id_header="${4:-}"
  body="${5:-}"
  headers=(-H "Content-Type: application/json")
  if [ -n "$access_token" ]; then
    headers+=(-H "X-One-Proxy-Access-Token: ${access_token}")
  fi
  if [ -n "$tenant_id_header" ]; then
    headers+=(-H "X-One-Proxy-Tenant-ID: ${tenant_id_header}")
  fi
  if [ -n "$body" ]; then
    curl -fsS -X "$method" "http://127.0.0.1:${panel_port}/api${path}" "${headers[@]}" -d "$body"
  else
    curl -fsS -X "$method" "http://127.0.0.1:${panel_port}/api${path}" "${headers[@]}"
  fi
}

node_api_request() {
  method="$1"
  path="$2"
  node_token="$3"
  body="${4:-}"
  if [ -n "$body" ]; then
    curl -fsS -X "$method" "http://127.0.0.1:${panel_port}/api${path}" -H "Content-Type: application/json" -H "X-One-Proxy-Node-Token: ${node_token}" -d "$body"
  else
    curl -fsS -X "$method" "http://127.0.0.1:${panel_port}/api${path}" -H "X-One-Proxy-Node-Token: ${node_token}"
  fi
}

mysql_query() {
  docker exec -e MYSQL_QUERY="$1" -e MYSQL_DATABASE="$final_db" "$mysql_container" sh -lc 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" "$MYSQL_DATABASE" -N -B -e "$MYSQL_QUERY"' 2>/dev/null
}

wait_pending_node() {
  name="$1"
  access_token="$2"
  tries=0
  while :; do
    response="$(api_request GET /nodes/pending "$access_token" "$tenant_id")"
    found="$(printf "%s" "$response" | node_id_by_name "$name")"
    if [ -n "$found" ]; then
      printf "%s" "$found"
      return 0
    fi
    tries=$((tries + 1))
    if [ "$tries" -ge 90 ]; then
      echo "pending_node=${name}=failed" >&2
      exit 1
    fi
    sleep 2
  done
}

login_response="$(api_request POST /auth/login "" "" "{\"account\":\"admin\",\"password\":\"${admin_password}\"}")"
access_token="$(printf "%s" "$login_response" | json_get data.accessToken)"
local_node_id="$(wait_pending_node "$local_node_name" "$access_token")"
local_approve_response="$(api_request POST "/nodes/${local_node_id}/approve" "$access_token" "$tenant_id" "{}")"
local_node_token="$(printf "%s" "$local_approve_response" | json_get data.accessToken)"

local_chain_response="$(api_request POST /proxy "$access_token" "$tenant_id" "{\"name\":\"v2.1.0 local chain\",\"destinationScope\":\"${scope_id}\",\"hops\":[\"${local_node_id}\"]}")"
local_chain_id="$(printf "%s" "$local_chain_response" | json_get data.id)"
remote_chain_response="$(api_request POST /proxy "$access_token" "$tenant_id" "{\"name\":\"v2.1.0 camelbot chain\",\"destinationScope\":\"${scope_id}\",\"hops\":[\"${remote_node_id}\"]}")"
remote_chain_id="$(printf "%s" "$remote_chain_response" | json_get data.id)"

local_path_body="{\"chainId\":\"${local_chain_id}\",\"name\":\"v2.1.0 local forward path\",\"mode\":\"forward\",\"protocol\":\"http\",\"serviceType\":\"http_forward_proxy\",\"targetNodeId\":\"${local_node_id}\",\"entryNodeId\":\"${local_node_id}\",\"relayNodeIds\":[],\"listenHost\":\"127.0.0.1\",\"listenPort\":${local_node_http_port},\"targetProtocol\":\"http\",\"targetHost\":\"example.com\",\"targetPort\":80,\"targetSni\":\"\",\"tlsMode\":\"\",\"authMode\":\"proxy_token\",\"options\":{}}"
local_path_response="$(api_request POST /proxy/paths "$access_token" "$tenant_id" "$local_path_body")"
local_access_path_id="$(printf "%s" "$local_path_response" | json_get data.id)"
remote_path_body="{\"chainId\":\"${remote_chain_id}\",\"name\":\"v2.1.0 camelbot forward path\",\"mode\":\"forward\",\"protocol\":\"http\",\"serviceType\":\"http_forward_proxy\",\"targetNodeId\":\"${remote_node_id}\",\"entryNodeId\":\"${remote_node_id}\",\"relayNodeIds\":[],\"listenHost\":\"127.0.0.1\",\"listenPort\":${remote_node_http_port},\"targetProtocol\":\"http\",\"targetHost\":\"example.com\",\"targetPort\":80,\"targetSni\":\"\",\"tlsMode\":\"\",\"authMode\":\"proxy_token\",\"options\":{}}"
remote_path_response="$(api_request POST /proxy/paths "$access_token" "$tenant_id" "$remote_path_body")"
remote_access_path_id="$(printf "%s" "$remote_path_response" | json_get data.id)"

group_response="$(api_request POST /proxy/route-groups "$access_token" "$tenant_id" "{\"name\":\"v2.1.0 final routes\",\"description\":\"final-schema standing cutover\"}")"
route_group_id="$(printf "%s" "$group_response" | json_get data.id)"
local_route_body="{\"groupId\":\"${route_group_id}\",\"priority\":10,\"matchType\":\"domain_suffix\",\"matchValue\":\".local.example.com\",\"actionType\":\"chain\",\"chainId\":\"${local_chain_id}\",\"destinationScope\":\"\"}"
local_route_response="$(api_request POST /proxy/routes "$access_token" "$tenant_id" "$local_route_body")"
local_route_id="$(printf "%s" "$local_route_response" | json_get data.id)"
remote_route_body="{\"groupId\":\"${route_group_id}\",\"priority\":20,\"matchType\":\"domain_suffix\",\"matchValue\":\".remote.example.com\",\"actionType\":\"chain\",\"chainId\":\"${remote_chain_id}\",\"destinationScope\":\"\"}"
remote_route_response="$(api_request POST /proxy/routes "$access_token" "$tenant_id" "$remote_route_body")"
remote_route_id="$(printf "%s" "$remote_route_response" | json_get data.id)"

api_request POST /policies/publish "$access_token" "$tenant_id" "{}" >/dev/null
bootstrap_response="$(api_request GET /proxy/extension/bootstrap "$access_token" "$tenant_id")"
printf "%s" "$bootstrap_response" | python3 -c 'import json,sys
payload=json.load(sys.stdin)["data"]
assert payload.get("schemaVersion") == "v2.1.0", payload
assert "groups" not in payload, payload.keys()
assert len(payload.get("nodes") or []) >= 2, payload
assert len(payload.get("accessPaths") or []) >= 2, payload
assert len(payload.get("routes") or []) >= 2, payload
print("bootstrap_schema=v2.1.0 nodes=%d access_paths=%d routes=%d" % (len(payload["nodes"]), len(payload["accessPaths"]), len(payload["routes"])))
'
proxy_token="$(printf "%s" "$bootstrap_response" | json_get data.proxyToken)"
token_hash="$(sha256_hex "$proxy_token")"
node_api_request POST /node/agent/proxy/token/validate "$local_node_token" "{\"tokenHash\":\"${token_hash}\",\"accessPathId\":\"${local_access_path_id}\",\"targetHost\":\"www.local.example.com\",\"targetPort\":80,\"protocol\":\"http\",\"routeId\":\"${local_route_id}\"}" >/dev/null
node_api_request POST /node/agent/proxy/token/validate "$remote_node_token" "{\"tokenHash\":\"${token_hash}\",\"accessPathId\":\"${remote_access_path_id}\",\"targetHost\":\"www.remote.example.com\",\"targetPort\":80,\"protocol\":\"http\",\"routeId\":\"${remote_route_id}\"}" >/dev/null
echo "local_node_approved=${local_node_id}"
echo "chains=${local_chain_id},${remote_chain_id}"
echo "access_paths=${local_access_path_id},${remote_access_path_id}"
echo "routes=${local_route_id},${remote_route_id}"
echo "proxy_token_validation=ok"
echo "db_tables=$(mysql_query "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE();")"
echo "db_goose_tables=$(mysql_query "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'goose_db_version';")"
mysql_query "SELECT id, status, enabled, public_host, public_port FROM nodes ORDER BY id;"
mysql_query "SELECT node_id, transport_type, direction, address, status FROM node_transports ORDER BY node_id, transport_type;"
mysql_query "SELECT id, chain_id, entry_node_id, target_node_id, listen_port, target_host, target_port, enabled FROM node_access_paths ORDER BY id;"
mysql_query "SELECT id, priority, match_type, match_value, action_type, chain_id, destination_scope, enabled FROM route_rules ORDER BY priority, id;"
mysql_query "SELECT id, access_token_hash REGEXP '^[0-9a-f]{64}$' AS access_hash_shape, refresh_token_hash REGEXP '^[0-9a-f]{64}$' AS refresh_hash_shape FROM sessions ORDER BY id;"
mysql_query "SELECT id, token_hash REGEXP '^[0-9a-f]{64}$' AS token_hash_shape FROM node_api_tokens ORDER BY node_id, id;"
mysql_query "SELECT id, token_hash REGEXP '^[0-9a-f]{64}$' AS token_hash_shape, consumed_at IS NOT NULL AS consumed FROM bootstrap_tokens ORDER BY id;"
REMOTE
}

wait_local_bound() {
  tries=0
  while :; do
    body="$(curl -fsS http://127.0.0.1:2988/healthz 2>/dev/null || true)"
    if printf "%s" "$body" | grep -q '"controlPlaneBound":true'; then
      echo "local_node_bound=ok"
      return 0
    fi
    tries=$((tries + 1))
    if [ "$tries" -ge 60 ]; then
      echo "local_node_bound=failed" >&2
      return 1
    fi
    sleep 2
  done
}

run_cutover() {
  require_run_env
  remote_output="$(deploy_panel_and_remote_node)"
  printf "%s\n" "$remote_output" | grep -Ev '(^|_)token=' || true
  panel_backup="$(printf "%s" "$remote_output" | sed -n 's/^panel_backup=//p' | tail -n 1)"
  remote_node_backup="$(printf "%s" "$remote_output" | sed -n 's/^remote_node_backup=//p' | tail -n 1)"
  tenant_id="$(printf "%s" "$remote_output" | sed -n 's/^tenant_id=//p' | tail -n 1)"
  scope_id="$(printf "%s" "$remote_output" | sed -n 's/^scope_id=//p' | tail -n 1)"
  remote_node_id="$(printf "%s" "$remote_output" | sed -n 's/^remote_node_id=//p' | tail -n 1)"
  remote_node_token="$(printf "%s" "$remote_output" | sed -n 's/^remote_node_token=//p' | tail -n 1)"
  local_bootstrap_token="$(printf "%s" "$remote_output" | sed -n 's/^local_bootstrap_token=//p' | tail -n 1)"
  if [ -z "$tenant_id" ] || [ -z "$scope_id" ] || [ -z "$remote_node_id" ] || [ -z "$remote_node_token" ] || [ -z "$local_bootstrap_token" ]; then
    echo "cutover output missing required identifiers" >&2
    exit 1
  fi
  if ! local_output="$(deploy_local_node "$local_bootstrap_token")"; then
    restore_remote_backups "$panel_backup" "$remote_node_backup"
    exit 1
  fi
  printf "%s\n" "$local_output"
  local_backup="$(printf "%s" "$local_output" | sed -n 's/^local_node_backup=//p' | tail -n 1)"
  if ! finish_output="$(finish_control_plane_state "$tenant_id" "$scope_id" "$remote_node_id" "$remote_node_token")"; then
    restore_local_backup "$local_backup"
    restore_remote_backups "$panel_backup" "$remote_node_backup"
    exit 1
  fi
  printf "%s\n" "$finish_output"
  wait_local_bound
}

case "$mode" in
  check)
    print_plan
    check_local
    check_remote
    ;;
  dry-run)
    require_tag
    print_plan
    ;;
  verify)
    verify_final_state
    ;;
  run)
    run_cutover
    ;;
esac
