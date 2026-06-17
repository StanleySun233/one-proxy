#!/usr/bin/env bash
set -euo pipefail

mode="${1:-check}"
tag="${2:-${ONEPROXY_IMAGE_TAG:-}}"
ssh_host="${CAMELBOT_SSH_HOST:-camelbot}"
node_repo="${ONEPROXY_NODE_IMAGE_REPO:-ghcr.io/stanleysun233/oneproxy-node}"
panel_container="${ONEPROXY_PANEL_CONTAINER:-one-proxy-panel}"
panel_port="${ONEPROXY_PANEL_PORT:-2886}"
panel_network="${ONEPROXY_PANEL_NETWORK:-one-proxy-net}"
mysql_container="${ONEPROXY_MYSQL_CONTAINER:-one-proxy-mysql8}"
remote_node_container="${ONEPROXY_CAMELBOT_NODE_CONTAINER:-one-proxy-node}"
local_node_container="${ONEPROXY_LOCAL_NODE_CONTAINER:-one-proxy-node}"
remote_node_volume="${ONEPROXY_POST_SETUP_CAMELBOT_NODE_VOLUME:-one-proxy-node-runtime-v210-final}"
local_node_volume="${ONEPROXY_POST_SETUP_LOCAL_NODE_VOLUME:-one-proxy-node-runtime-v210-final}"
remote_node_name="${ONEPROXY_POST_SETUP_CAMELBOT_NODE_NAME:-sg-astar-58}"
local_node_name="${ONEPROXY_POST_SETUP_LOCAL_NODE_NAME:-hk-public-node}"
remote_node_parent_url="${ONEPROXY_POST_SETUP_CAMELBOT_NODE_PARENT_URL:-http://${panel_container}:2886}"
local_node_parent_url="${ONEPROXY_POST_SETUP_LOCAL_NODE_PARENT_URL:-}"
remote_node_public_host="${ONEPROXY_POST_SETUP_CAMELBOT_NODE_PUBLIC_HOST:-127.0.0.1}"
local_node_public_host="${ONEPROXY_POST_SETUP_LOCAL_NODE_PUBLIC_HOST:-127.0.0.1}"
remote_node_http_port="${ONEPROXY_POST_SETUP_CAMELBOT_NODE_HTTP_PORT:-2988}"
local_node_http_port="${ONEPROXY_POST_SETUP_LOCAL_NODE_HTTP_PORT:-2988}"
tenant_name="${ONEPROXY_POST_SETUP_TENANT_NAME:-OneProxy v2.1.0}"
scope_name="${ONEPROXY_POST_SETUP_SCOPE_NAME:-v2.1.0 final scope}"
confirm="${ONEPROXY_POST_SETUP_CONFIRM:-}"

case "$mode" in
  check|dry-run|run|verify)
    ;;
  *)
    echo "usage: $0 [check|dry-run|run|verify] <immutable_tag>" >&2
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

node_image() {
  printf "%s:%s" "$node_repo" "$tag"
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

print_plan() {
  parent="$local_node_parent_url"
  if [ -z "$parent" ]; then
    parent="$(infer_local_node_parent_url)"
  fi
  echo "mode=${mode}"
  echo "tag=${tag:-<required-for-dry-run-run>}"
  if [ -n "$tag" ]; then
    echo "node_image=$(node_image)"
  else
    echo "node_image=<required-for-dry-run-run>"
  fi
  echo "panel=http://${ssh_host}:${panel_port}"
  echo "remote_node=${remote_node_name}"
  echo "local_node=${local_node_name}"
  echo "remote_node_parent_url=${remote_node_parent_url}"
  echo "local_node_parent_url=${parent:-<required-for-run>}"
}

remote_panel_value() {
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

check_remote_state() {
  ssh -T "$ssh_host" 'bash -s' -- "$panel_container" "$panel_port" "$mysql_container" "$remote_node_container" <<'REMOTE'
set -euo pipefail
panel_container="$1"
panel_port="$2"
mysql_container="$3"
remote_node_container="$4"
echo "target=camelbot-panel"
docker inspect "$panel_container" --format 'current={{.Config.Image}} status={{.State.Status}} started={{.State.StartedAt}}' 2>/dev/null || echo "current=missing"
setup_status="$(curl -fsS "http://127.0.0.1:${panel_port}/api/setup/status" 2>/dev/null || true)"
health="$(curl -fsS "http://127.0.0.1:${panel_port}/healthz" 2>/dev/null || true)"
echo "setup_status=${setup_status:-unavailable}"
echo "health=${health:-unavailable}"
configured="$(printf "%s" "$setup_status" | python3 -c 'import json,sys
try:
    print(str(bool(json.load(sys.stdin).get("data", {}).get("configured"))).lower())
except Exception:
    print("unknown")
' 2>/dev/null || echo unknown)"
echo "configured=${configured}"
db_name="$(docker exec "$panel_container" sh -lc 'if [ -f /app/data/.env ]; then awk -F= '\''$1=="MYSQL_DSN"{sub(/^[^=]*=/,"",$0); print; exit}'\'' /app/data/.env; fi' 2>/dev/null | sed -n 's#.*)/\\([^?]*\\).*#\\1#p' || true)"
if [ -n "$db_name" ] && docker inspect "$mysql_container" >/dev/null 2>&1; then
  tables="$(docker exec -e MYSQL_QUERY="SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '${db_name}';" "$mysql_container" sh -lc 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -N -B -e "$MYSQL_QUERY"' 2>/dev/null || echo unknown)"
  echo "db=${db_name} tables=${tables}"
else
  echo "db=<unconfigured>"
fi
echo "target=camelbot-node"
docker inspect "$remote_node_container" --format 'current={{.Config.Image}} status={{.State.Status}} started={{.State.StartedAt}}' 2>/dev/null || echo "current=missing"
REMOTE
}

check_local_state() {
  echo "target=local-node"
  docker inspect "$local_node_container" --format 'current={{.Config.Image}} status={{.State.Status}} started={{.State.StartedAt}}' 2>/dev/null || echo "current=missing"
}

require_run_env() {
  require_tag
  if [ "$confirm" != "bootstrap-nodes" ]; then
    echo "run requires ONEPROXY_POST_SETUP_CONFIRM=bootstrap-nodes" >&2
    exit 2
  fi
  if [ -z "$local_node_parent_url" ]; then
    local_node_parent_url="$(infer_local_node_parent_url)"
  fi
  if [ -z "$local_node_parent_url" ]; then
    echo "run requires ONEPROXY_POST_SETUP_LOCAL_NODE_PARENT_URL or a reachable camelbot panel URL" >&2
    exit 2
  fi
}

deploy_remote_node_and_tokens() {
  ssh -T "$ssh_host" 'bash -s' -- \
    "$(node_image)" "$panel_container" "$panel_port" "$panel_network" "$mysql_container" "$remote_node_container" "$remote_node_volume" \
    "$remote_node_name" "$remote_node_parent_url" "$remote_node_public_host" "$remote_node_http_port" "$local_node_name" "$local_node_public_host" \
    "$local_node_http_port" "$tenant_name" "$scope_name" <<'REMOTE'
set -Eeuo pipefail
node_image="$1"
panel_container="$2"
panel_port="$3"
panel_network="$4"
mysql_container="$5"
remote_node_container="$6"
remote_node_volume="$7"
remote_node_name="$8"
remote_node_parent_url="$9"
remote_node_public_host="${10}"
remote_node_http_port="${11}"
local_node_name="${12}"
local_node_public_host="${13}"
local_node_http_port="${14}"
tenant_name="${15}"
scope_name="${16}"
node_env="$(mktemp)"
remote_node_backup="${remote_node_container}-prev-$(date +%Y%m%d%H%M%S)"
renamed=0
completed=0
cleanup() {
  rm -f "$node_env"
}
rollback() {
  status=$?
  if [ "$completed" -eq 0 ] && [ "$renamed" -eq 1 ]; then
    docker rm -f "$remote_node_container" >/dev/null 2>&1 || true
    if docker ps -a --format '{{.Names}}' | grep -Fxq "$remote_node_backup"; then
      docker rename "$remote_node_backup" "$remote_node_container" >/dev/null 2>&1 || true
      docker start "$remote_node_container" >/dev/null 2>&1 || true
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

admin_password="$(docker exec "$panel_container" sh -lc 'awk -F= '\''$1=="ADMIN_PASSWORD"{sub(/^[^=]*=/,"",$0); print; exit}'\'' /app/data/.env' 2>/dev/null || true)"
if [ -z "$admin_password" ]; then
  echo "panel admin password is unavailable from configured panel" >&2
  exit 2
fi
configured="$(curl -fsS "http://127.0.0.1:${panel_port}/api/setup/status" | python3 -c 'import json,sys; print(str(bool(json.load(sys.stdin).get("data", {}).get("configured"))).lower())')"
if [ "$configured" != "true" ]; then
  echo "panel is not configured" >&2
  exit 2
fi

login_response="$(api_request POST /auth/login "" "" "{\"account\":\"admin\",\"password\":\"${admin_password}\"}")"
access_token="$(printf "%s" "$login_response" | json_get data.accessToken)"
account_id="$(printf "%s" "$login_response" | json_get data.account.id)"
run_id="$(date +%Y%m%d%H%M%S)"
tenant_response="$(api_request POST /tenants "$access_token" "" "{\"name\":\"${tenant_name} ${run_id}\",\"initialAdminAccountId\":\"${account_id}\"}")"
tenant_id="$(printf "%s" "$tenant_response" | json_get data.tenant.id)"
scope_response="$(api_request POST /proxy/scopes "$access_token" "$tenant_id" "{\"name\":\"${scope_name}\",\"description\":\"post-setup final release\"}")"
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
  echo "NODE_UDP_ACCESS_LISTEN_ADDR=:2991"
  echo "NODE_DIRECT_LISTEN_ADDR=:2992"
  echo "NODE_DIRECT_STUN_SERVERS="
  echo "NODE_DIRECT_REFRESH_INTERVAL=5s"
  echo "NODE_HEARTBEAT_INTERVAL=2s"
  echo "NODE_POLICY_STATE_PATH=runtime/node-policy-state.json"
  echo "NODE_RUNTIME_CONFIG_PATH=runtime/node-runtime.json"
} > "$node_env"

docker pull "$node_image" >/dev/null
if docker inspect "$remote_node_container" >/dev/null 2>&1; then
  docker rename "$remote_node_container" "$remote_node_backup"
  renamed=1
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
completed=1
trap - ERR HUP INT TERM
echo "tenant_id=${tenant_id}"
echo "scope_id=${scope_id}"
echo "remote_node_id=${remote_node_id}"
echo "remote_node_token=${remote_node_token}"
echo "local_bootstrap_token=${local_bootstrap_token}"
echo "remote_node_bound=ok"
REMOTE
}

deploy_local_node() {
  local_bootstrap_token="$1"
  image="$(node_image)"
  env_file="$(mktemp)"
  backup="${local_node_container}-prev-$(date +%Y%m%d%H%M%S)"
  renamed=0
  completed=0
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
    echo "NODE_UDP_ACCESS_LISTEN_ADDR=:2991"
    echo "NODE_DIRECT_LISTEN_ADDR=:2992"
    echo "NODE_DIRECT_STUN_SERVERS="
    echo "NODE_DIRECT_REFRESH_INTERVAL=5s"
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
    --env-file "$env_file" \
    -v "${local_node_volume}:/app/runtime" \
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

finish_policy_and_evidence() {
  tenant_id="$1"
  scope_id="$2"
  remote_node_id="$3"
  remote_node_token="$4"
  ssh -T "$ssh_host" 'bash -s' -- "$panel_container" "$panel_port" "$mysql_container" "$tenant_id" "$scope_id" "$remote_node_id" "$remote_node_token" "$local_node_name" "$local_node_http_port" "$remote_node_http_port" <<'REMOTE'
set -Eeuo pipefail
panel_container="$1"
panel_port="$2"
mysql_container="$3"
tenant_id="$4"
scope_id="$5"
remote_node_id="$6"
remote_node_token="$7"
local_node_name="$8"
local_node_http_port="$9"
remote_node_http_port="${10}"

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
  db_name="$(docker exec "$panel_container" sh -lc 'awk -F= '\''$1=="MYSQL_DSN"{sub(/^[^=]*=/,"",$0); print; exit}'\'' /app/data/.env' 2>/dev/null | sed -n 's#.*)/\([^?]*\).*#\1#p')"
  docker exec -e MYSQL_QUERY="$1" -e MYSQL_DATABASE="$db_name" "$mysql_container" sh -lc 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" "$MYSQL_DATABASE" -N -B -e "$MYSQL_QUERY"' 2>/dev/null || true
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

admin_password="$(docker exec "$panel_container" sh -lc 'awk -F= '\''$1=="ADMIN_PASSWORD"{sub(/^[^=]*=/,"",$0); print; exit}'\'' /app/data/.env' 2>/dev/null)"
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

group_response="$(api_request POST /proxy/route-groups "$access_token" "$tenant_id" "{\"name\":\"v2.1.0 final routes\",\"description\":\"post-setup final release\"}")"
route_group_id="$(printf "%s" "$group_response" | json_get data.id)"
local_route_response="$(api_request POST /proxy/routes "$access_token" "$tenant_id" "{\"groupId\":\"${route_group_id}\",\"priority\":10,\"matchType\":\"domain_suffix\",\"matchValue\":\".local.example.com\",\"actionType\":\"chain\",\"chainId\":\"${local_chain_id}\",\"destinationScope\":\"\"}")"
local_route_id="$(printf "%s" "$local_route_response" | json_get data.id)"
remote_route_response="$(api_request POST /proxy/routes "$access_token" "$tenant_id" "{\"groupId\":\"${route_group_id}\",\"priority\":20,\"matchType\":\"domain_suffix\",\"matchValue\":\".remote.example.com\",\"actionType\":\"chain\",\"chainId\":\"${remote_chain_id}\",\"destinationScope\":\"\"}")"
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
mysql_query "SELECT id, status, enabled, public_host, public_port FROM nodes ORDER BY id;"
mysql_query "SELECT node_id, transport_type, direction, address, status FROM node_transports ORDER BY node_id, transport_type;"
mysql_query "SELECT id, chain_id, entry_node_id, target_node_id, listen_port, target_host, target_port, enabled FROM node_access_paths ORDER BY id;"
mysql_query "SELECT id, priority, match_type, match_value, action_type, chain_id, destination_scope, enabled FROM route_rules ORDER BY priority, id;"
mysql_query "SELECT id, access_token_hash REGEXP '^[0-9a-f]{64}$' AS access_hash_shape, refresh_token_hash REGEXP '^[0-9a-f]{64}$' AS refresh_hash_shape FROM sessions ORDER BY id;"
mysql_query "SELECT id, token_hash REGEXP '^[0-9a-f]{64}$' AS token_hash_shape FROM node_api_tokens ORDER BY node_id, id;"
mysql_query "SELECT id, token_hash REGEXP '^[0-9a-f]{64}$' AS token_hash_shape, consumed_at IS NOT NULL AS consumed FROM bootstrap_tokens ORDER BY id;"
REMOTE
}

run_post_setup() {
  require_run_env
  remote_output="$(deploy_remote_node_and_tokens)"
  printf "%s\n" "$remote_output" | grep -Ev '(^|_)token=' || true
  tenant_id="$(printf "%s" "$remote_output" | sed -n 's/^tenant_id=//p' | tail -n 1)"
  scope_id="$(printf "%s" "$remote_output" | sed -n 's/^scope_id=//p' | tail -n 1)"
  remote_node_id="$(printf "%s" "$remote_output" | sed -n 's/^remote_node_id=//p' | tail -n 1)"
  remote_node_token="$(printf "%s" "$remote_output" | sed -n 's/^remote_node_token=//p' | tail -n 1)"
  local_bootstrap_token="$(printf "%s" "$remote_output" | sed -n 's/^local_bootstrap_token=//p' | tail -n 1)"
  if [ -z "$tenant_id" ] || [ -z "$scope_id" ] || [ -z "$remote_node_id" ] || [ -z "$remote_node_token" ] || [ -z "$local_bootstrap_token" ]; then
    echo "post-setup output missing required identifiers" >&2
    exit 1
  fi
  deploy_local_node "$local_bootstrap_token"
  finish_policy_and_evidence "$tenant_id" "$scope_id" "$remote_node_id" "$remote_node_token"
}

verify_post_setup() {
  require_tag
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
  if ! ssh -T "$ssh_host" 'bash -s' -- "$panel_container" "$panel_port" "$mysql_container" "$remote_node_container" "$expected_node_image" <<'REMOTE'
set -euo pipefail
panel_container="$1"
panel_port="$2"
mysql_container="$3"
remote_node_container="$4"
expected_node_image="$5"
status=0
query() {
  db_name="$(docker exec "$panel_container" sh -lc 'awk -F= '\''$1=="MYSQL_DSN"{sub(/^[^=]*=/,"",$0); print; exit}'\'' /app/data/.env' 2>/dev/null | sed -n 's#.*)/\([^?]*\).*#\1#p')"
  docker exec -e MYSQL_QUERY="$1" -e MYSQL_DATABASE="$db_name" "$mysql_container" sh -lc 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" "$MYSQL_DATABASE" -N -B -e "$MYSQL_QUERY"' 2>/dev/null || true
}
echo "target=camelbot-panel"
setup_status="$(curl -fsS "http://127.0.0.1:${panel_port}/api/setup/status" 2>/dev/null || true)"
health="$(curl -fsS "http://127.0.0.1:${panel_port}/healthz" 2>/dev/null || true)"
echo "setup_status=${setup_status:-unavailable}"
echo "health=${health:-unavailable}"
echo "target=camelbot-node"
remote_image="$(docker inspect "$remote_node_container" --format '{{.Config.Image}}' 2>/dev/null || true)"
remote_health="$(curl -fsS http://127.0.0.1:2988/healthz 2>/dev/null || true)"
echo "image=${remote_image:-missing}"
echo "health=${remote_health:-unavailable}"
if [ "$remote_image" != "$expected_node_image" ] || ! printf "%s" "$remote_health" | grep -q '"controlPlaneBound":true'; then
  status=1
fi
nodes="$(query "SELECT COUNT(*) FROM nodes;")"
access_paths="$(query "SELECT COUNT(*) FROM node_access_paths;")"
routes="$(query "SELECT COUNT(*) FROM route_rules;")"
policies="$(query "SELECT COUNT(*) FROM policy_revisions;")"
goose="$(query "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'goose_db_version';")"
echo "db_counts=nodes:${nodes:-unknown} access_paths:${access_paths:-unknown} routes:${routes:-unknown} policies:${policies:-unknown} goose_tables:${goose:-unknown}"
if [ "${nodes:-0}" -lt 2 ] || [ "${access_paths:-0}" -lt 2 ] || [ "${routes:-0}" -lt 2 ] || [ "${policies:-0}" -lt 1 ] || [ "${goose:-1}" != "0" ]; then
  status=1
fi
exit "$status"
REMOTE
  then
    status=1
  fi
  return "$status"
}

case "$mode" in
  check)
    print_plan
    check_remote_state
    check_local_state
    ;;
  dry-run)
    require_tag
    print_plan
    ;;
  run)
    run_post_setup
    ;;
  verify)
    verify_post_setup
    ;;
esac
