#!/usr/bin/env bash
set -euo pipefail

mode="${1:-check}"
ssh_host="${CAMELBOT_SSH_HOST:-camelbot}"
remote_dir="${CAMELBOT_V210_REMOTE_DIR:-oneproxy-v210-isolated}"
project="${ONEPROXY_CAMELBOT_V210_PROJECT:-oneproxy-v210-camelbot}"
tag="${ONEPROXY_IMAGE_TAG:-${2:-}}"
panel_repo="${ONEPROXY_PANEL_IMAGE_REPO:-ghcr.io/stanleysun233/oneproxy-panel}"
node_repo="${ONEPROXY_NODE_IMAGE_REPO:-ghcr.io/stanleysun233/oneproxy-node}"
mysql_image="${ONEPROXY_MYSQL_IMAGE:-mysql:8.4}"
redis_image="${ONEPROXY_REDIS_IMAGE:-redis:7}"
mysql_db="${ONEPROXY_CAMELBOT_V210_DB:-oneproxy_v210_camelbot}"
mysql_root_password="${ONEPROXY_CAMELBOT_V210_MYSQL_ROOT_PASSWORD:-oneproxy-v210-camelbot-root}"
jwt_key="${ONEPROXY_CAMELBOT_V210_JWT_SIGNING_KEY:-oneproxy-v210-camelbot-jwt}"
admin_password="${ONEPROXY_CAMELBOT_V210_ADMIN_PASSWORD:-oneproxy-v210-camelbot-admin}"
tenant_name="${ONEPROXY_CAMELBOT_V210_TENANT_NAME:-OneProxy v2.1.0 Camelbot}"
node_name="${ONEPROXY_CAMELBOT_V210_NODE_NAME:-oneproxy-v210-camelbot-node}"
panel_host_port="${ONEPROXY_CAMELBOT_V210_PANEL_PORT:-13886}"
node_http_host_port="${ONEPROXY_CAMELBOT_V210_NODE_HTTP_PORT:-13988}"
node_https_host_port="${ONEPROXY_CAMELBOT_V210_NODE_HTTPS_PORT:-13989}"
node_tcp_host_port="${ONEPROXY_CAMELBOT_V210_NODE_TCP_PORT:-13990}"
node_udp_host_port="${ONEPROXY_CAMELBOT_V210_NODE_UDP_PORT:-13991}"
node_direct_host_port="${ONEPROXY_CAMELBOT_V210_NODE_DIRECT_PORT:-13992}"
mysql_host_port="${ONEPROXY_CAMELBOT_V210_MYSQL_PORT:-13316}"
redis_host_port="${ONEPROXY_CAMELBOT_V210_REDIS_PORT:-16389}"

case "$mode" in
  check|build|run|clean)
    ;;
  *)
    echo "usage: $0 [check|build|run|clean] [image_tag]" >&2
    exit 2
    ;;
esac

require_tag() {
  if [ -z "$tag" ] || [ "$tag" = "latest" ]; then
    echo "build and run require an immutable image tag" >&2
    exit 2
  fi
}

if [ "$mode" = "build" ] || [ "$mode" = "run" ]; then
  require_tag
fi
tag_arg="${tag:-__ONEPROXY_EMPTY_TAG__}"

control_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$control_dir"
}
trap cleanup EXIT

ssh_opts=(
  -o BatchMode=yes
  -o ConnectTimeout=15
  -o ConnectionAttempts=3
  -o ServerAliveInterval=10
  -o ServerAliveCountMax=3
  -o ControlMaster=auto
  -o ControlPersist=120s
  -o ControlPath="${control_dir}/%r@%h:%p"
)

remote_args=(
  "$mode" "$remote_dir" "$project" "$tag_arg" "$panel_repo" "$node_repo" "$mysql_image" "$redis_image"
  "$mysql_db" "$mysql_root_password" "$jwt_key" "$admin_password" "$tenant_name" "$node_name" "$panel_host_port" "$node_http_host_port"
  "$node_https_host_port" "$node_tcp_host_port" "$node_udp_host_port" "$node_direct_host_port"
  "$mysql_host_port" "$redis_host_port"
)
remote_cmd="bash -s --"
for arg in "${remote_args[@]}"; do
  remote_cmd+=" $(printf "%q" "$arg")"
done

ssh -T "${ssh_opts[@]}" "$ssh_host" "$remote_cmd" <<'REMOTE'
set -euo pipefail

mode="$1"
remote_dir="$2"
project="$3"
tag="$4"
if [ "$tag" = "__ONEPROXY_EMPTY_TAG__" ]; then
  tag=""
fi
panel_repo="$5"
node_repo="$6"
mysql_image="$7"
redis_image="$8"
mysql_db="$9"
mysql_root_password="${10}"
jwt_key="${11}"
admin_password="${12}"
tenant_name="${13}"
node_name="${14}"
panel_host_port="${15}"
node_http_host_port="${16}"
node_https_host_port="${17}"
node_tcp_host_port="${18}"
node_udp_host_port="${19}"
node_direct_host_port="${20}"
mysql_host_port="${21}"
redis_host_port="${22}"
compose_file="${remote_dir}/compose.yml"

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -p "$project" -f "$compose_file" "$@"
  elif command -v docker-compose >/dev/null 2>&1; then
    docker-compose -p "$project" -f "$compose_file" "$@"
  else
    echo "docker compose is required" >&2
    exit 2
  fi
}

write_compose() {
  node_bootstrap_token="${1:-}"
  mkdir -p "$remote_dir"
  cat > "$compose_file" <<COMPOSE
services:
  mysql:
    image: ${mysql_image}
    environment:
      MYSQL_ROOT_PASSWORD: "${mysql_root_password}"
      MYSQL_DATABASE: "${mysql_db}"
    ports:
      - "127.0.0.1:${mysql_host_port}:3306"
    volumes:
      - mysql-data:/var/lib/mysql
  redis:
    image: ${redis_image}
    ports:
      - "127.0.0.1:${redis_host_port}:6379"
    volumes:
      - redis-data:/data
  panel:
    image: ${panel_repo}:${tag}
    depends_on:
      - mysql
      - redis
    environment:
      MYSQL_DSN: "root:${mysql_root_password}@tcp(mysql:3306)/${mysql_db}?charset=utf8mb4&parseTime=true&loc=UTC"
      ADMIN_PASSWORD: "${admin_password}"
      JWT_SIGNING_KEY: "${jwt_key}"
      REDIS_URL: "redis://redis:6379/0"
      PORT: "2886"
      HTTP_ADDR: "0.0.0.0:2887"
      CONTROL_PLANE_URL: "http://panel:2887"
      PUBLIC_CERT_PROVIDER: "manual"
      TZ: "UTC"
    ports:
      - "127.0.0.1:${panel_host_port}:2886"
    volumes:
      - panel-data:/app/data
  node:
    image: ${node_repo}:${tag}
    depends_on:
      - panel
    environment:
      NODE_LISTEN_ADDR: ":2988"
      NODE_HTTPS_LISTEN_ADDR: ":2989"
      NODE_TCP_ACCESS_LISTEN_ADDR: ":2990"
      NODE_UDP_ACCESS_LISTEN_ADDR: ":2991"
      NODE_DIRECT_LISTEN_ADDR: ":2992"
      NODE_NAME: "${node_name}"
      NODE_MODE: "edge"
      NODE_BOOTSTRAP_TOKEN: "${node_bootstrap_token}"
      NODE_PARENT_URL: "http://panel:2887"
      NODE_PUBLIC_HOST: "127.0.0.1"
      NODE_HEARTBEAT_INTERVAL: "2s"
      NODE_DIRECT_STUN_SERVERS: ""
      NODE_DIRECT_REFRESH_INTERVAL: "5s"
      NODE_POLICY_STATE_PATH: "runtime/node-policy-state.json"
      NODE_RUNTIME_CONFIG_PATH: "runtime/node-runtime.json"
      PUBLIC_CERT_PROVIDER: "manual"
      TZ: "UTC"
    ports:
      - "127.0.0.1:${node_http_host_port}:2988"
      - "127.0.0.1:${node_https_host_port}:2989"
      - "127.0.0.1:${node_tcp_host_port}:2990"
      - "127.0.0.1:${node_udp_host_port}:2991/udp"
      - "127.0.0.1:${node_direct_host_port}:2992/udp"
    volumes:
      - node-runtime:/app/runtime
volumes:
  mysql-data:
  redis-data:
  panel-data:
  node-runtime:
COMPOSE
}

print_plan() {
  echo "mode=${mode}"
  echo "ssh_host=$(hostname)"
  echo "remote_dir=${remote_dir}"
  echo "project=${project}"
  echo "services=mysql,redis,panel,node"
  echo "panel_image=${panel_repo}:${tag:-<required-for-build-run>}"
  echo "node_image=${node_repo}:${tag:-<required-for-build-run>}"
  echo "tenant_name=${tenant_name}"
  echo "node_name=${node_name}"
  echo "host_ports=panel:${panel_host_port},node_http:${node_http_host_port},node_https:${node_https_host_port},node_tcp:${node_tcp_host_port},node_udp:${node_udp_host_port},node_direct:${node_direct_host_port},mysql:${mysql_host_port},redis:${redis_host_port}"
  echo "production_replacement=false"
  echo "destructive_modes=run,clean"
}

wait_for_http() {
  url="$1"
  label="$2"
  tries=0
  while :; do
    if curl -fsS "$url" >/dev/null 2>&1; then
      echo "${label}=ok"
      return 0
    fi
    tries=$((tries + 1))
    if [ "$tries" -ge 60 ]; then
      echo "${label}=failed" >&2
      return 1
    fi
    sleep 2
  done
}

wait_for_node_bound() {
  tries=0
  while :; do
    body="$(curl -fsS "http://127.0.0.1:${node_http_host_port}/healthz" 2>/dev/null || true)"
    if printf '%s' "$body" | grep -q '"controlPlaneBound":true'; then
      echo "node_bound=ok"
      return 0
    fi
    tries=$((tries + 1))
    if [ "$tries" -ge 60 ]; then
      echo "node_bound=failed body=${body}" >&2
      return 1
    fi
    sleep 2
  done
}

json_get() {
  path="$1"
  python3 -c 'import json,sys
obj=json.load(sys.stdin)
for part in sys.argv[1].split("."):
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
' "$path"
}

sha256_hex() {
  python3 -c 'import hashlib,sys; print(hashlib.sha256(sys.argv[1].encode()).hexdigest())' "$1"
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
    curl -fsS -X "$method" "http://127.0.0.1:${panel_host_port}/api${path}" "${headers[@]}" -d "$body"
  else
    curl -fsS -X "$method" "http://127.0.0.1:${panel_host_port}/api${path}" "${headers[@]}"
  fi
}

node_api_request() {
  method="$1"
  path="$2"
  node_token="$3"
  body="${4:-}"
  if [ -n "$body" ]; then
    curl -fsS -X "$method" "http://127.0.0.1:${panel_host_port}/api${path}" -H "Content-Type: application/json" -H "X-One-Proxy-Node-Token: ${node_token}" -d "$body"
  else
    curl -fsS -X "$method" "http://127.0.0.1:${panel_host_port}/api${path}" -H "X-One-Proxy-Node-Token: ${node_token}"
  fi
}

create_control_plane_state() {
  login_response="$(api_request POST /auth/login "" "" "{\"account\":\"admin\",\"password\":\"${admin_password}\"}")"
  access_token="$(printf '%s' "$login_response" | json_get data.accessToken)"
  account_id="$(printf '%s' "$login_response" | json_get data.account.id)"

  tenant_response="$(api_request POST /tenants "$access_token" "" "{\"name\":\"${tenant_name}\",\"initialAdminAccountId\":\"${account_id}\"}")"
  tenant_id="$(printf '%s' "$tenant_response" | json_get data.tenant.id)"

  scope_response="$(api_request POST /proxy/scopes "$access_token" "$tenant_id" "{\"name\":\"v2.1.0 camelbot scope\",\"description\":\"isolated release scenario\"}")"
  scope_id="$(printf '%s' "$scope_response" | json_get data.id)"

  bootstrap_response="$(api_request POST /nodes/bootstrap/token "$access_token" "$tenant_id" "{\"targetType\":\"node\",\"nodeName\":\"${node_name}\",\"nodeMode\":\"edge\",\"scopeKey\":\"${scope_id}\",\"publicHost\":\"127.0.0.1\",\"publicPort\":${node_http_host_port}}")"
  node_bootstrap_token="$(printf '%s' "$bootstrap_response" | json_get data.token)"
  echo "tenant_created=${tenant_id}"
  echo "scope_created=${scope_id}"
  echo "bootstrap_token_issued=yes"
}

wait_for_pending_node() {
  tries=0
  while :; do
    pending_response="$(api_request GET /nodes/pending "$access_token" "$tenant_id")"
    node_id="$(printf '%s' "$pending_response" | json_get data.0.id)"
    if [ -n "$node_id" ]; then
      echo "node_pending=${node_id}"
      return 0
    fi
    tries=$((tries + 1))
    if [ "$tries" -ge 60 ]; then
      echo "node_pending=failed" >&2
      return 1
    fi
    sleep 2
  done
}

create_access_path_state() {
  chain_response="$(api_request POST /proxy "$access_token" "$tenant_id" "{\"name\":\"v2.1.0 camelbot chain\",\"destinationScope\":\"${scope_id}\",\"hops\":[\"${node_id}\"]}")"
  chain_id="$(printf '%s' "$chain_response" | json_get data.id)"

  path_body="{\"chainId\":\"${chain_id}\",\"name\":\"v2.1.0 camelbot forward path\",\"mode\":\"forward\",\"protocol\":\"http\",\"serviceType\":\"http_forward_proxy\",\"targetNodeId\":\"${node_id}\",\"entryNodeId\":\"${node_id}\",\"relayNodeIds\":[],\"listenHost\":\"127.0.0.1\",\"listenPort\":${node_http_host_port},\"targetProtocol\":\"http\",\"targetHost\":\"example.test\",\"targetPort\":80,\"targetSni\":\"\",\"tlsMode\":\"\",\"authMode\":\"proxy_token\",\"options\":{}}"
  path_response="$(api_request POST /proxy/paths "$access_token" "$tenant_id" "$path_body")"
  access_path_id="$(printf '%s' "$path_response" | json_get data.id)"

  group_response="$(api_request POST /proxy/route-groups "$access_token" "$tenant_id" "{\"name\":\"v2.1.0 camelbot routes\",\"description\":\"isolated release scenario\"}")"
  route_group_id="$(printf '%s' "$group_response" | json_get data.id)"

  route_body="{\"groupId\":\"${route_group_id}\",\"priority\":10,\"matchType\":\"domain_suffix\",\"matchValue\":\".example.test\",\"actionType\":\"chain\",\"chainId\":\"${chain_id}\",\"destinationScope\":\"\"}"
  route_response="$(api_request POST /proxy/routes "$access_token" "$tenant_id" "$route_body")"
  route_id="$(printf '%s' "$route_response" | json_get data.id)"
  echo "chain_created=${chain_id}"
  echo "access_path_created=${access_path_id}"
  echo "route_created=${route_id}"
}

validate_latest_bootstrap() {
  bootstrap_response="$(api_request GET /proxy/extension/bootstrap "$access_token" "$tenant_id")"
  printf '%s' "$bootstrap_response" | python3 -c 'import json,sys
payload=json.load(sys.stdin)["data"]
assert payload.get("schemaVersion") == "v2.1.0", payload
assert "groups" not in payload, payload.keys()
assert payload.get("nodes"), payload
assert payload.get("accessPaths"), payload
assert payload.get("routes"), payload
print("bootstrap_schema=v2.1.0 access_paths=%d routes=%d" % (len(payload["accessPaths"]), len(payload["routes"])))
'
  proxy_token="$(printf '%s' "$bootstrap_response" | json_get data.proxyToken)"
  bootstrap_access_path_id="$(printf '%s' "$bootstrap_response" | json_get data.accessPaths.0.id)"
  bootstrap_route_id="$(printf '%s' "$bootstrap_response" | json_get data.routes.0.id)"
  token_hash="$(sha256_hex "$proxy_token")"
  validate_body="{\"tokenHash\":\"${token_hash}\",\"accessPathId\":\"${bootstrap_access_path_id}\",\"targetHost\":\"example.test\",\"targetPort\":80,\"protocol\":\"http\",\"routeId\":\"${bootstrap_route_id}\"}"
  node_api_request POST /node/agent/proxy/token/validate "$node_access_token" "$validate_body" >/dev/null
  echo "proxy_token_validation=ok"
}

run_db_evidence() {
  mysql_query() {
    compose exec -T mysql env MYSQL_PWD="${mysql_root_password}" mysql -uroot "${mysql_db}" -N -B -e "$1" || true
  }
  mysql_query "SELECT id, status, enabled, public_host, public_port FROM nodes ORDER BY id;"
  mysql_query "SELECT node_id, transport_type, direction, address, status FROM node_transports ORDER BY node_id, transport_type;"
  mysql_query "SELECT id, chain_id, entry_node_id, target_node_id, listen_port, target_host, target_port, enabled FROM node_access_paths ORDER BY id;"
  mysql_query "SELECT id, priority, match_type, match_value, action_type, chain_id, destination_scope, enabled FROM route_rules ORDER BY priority, id;"
  mysql_query "SELECT id, access_token_hash REGEXP '^[0-9a-f]{64}$' AS access_hash_shape, refresh_token_hash REGEXP '^[0-9a-f]{64}$' AS refresh_hash_shape FROM sessions ORDER BY id;"
  mysql_query "SELECT id, token_hash REGEXP '^[0-9a-f]{64}$' AS token_hash_shape FROM node_api_tokens ORDER BY node_id, id;"
  mysql_query "SELECT id, token_hash REGEXP '^[0-9a-f]{64}$' AS token_hash_shape, consumed_at IS NOT NULL AS consumed FROM bootstrap_tokens ORDER BY id;"
}

case "$mode" in
  check)
    print_plan
    command -v docker >/dev/null 2>&1 && echo "docker=found" || echo "docker=missing"
    if docker compose version >/dev/null 2>&1 || command -v docker-compose >/dev/null 2>&1; then
      echo "docker_compose=found"
    else
      echo "docker_compose=missing"
    fi
    command -v curl >/dev/null 2>&1 && echo "curl=found" || echo "curl=missing"
    command -v python3 >/dev/null 2>&1 && echo "python3=found" || echo "python3=missing"
    if command -v docker >/dev/null 2>&1; then
      docker ps -a --format '{{.Names}}' | grep -E "^${project}[-_]" || true
    fi
    ;;
  build)
    write_compose ""
    print_plan
    compose pull mysql redis panel node
    ;;
  run)
    command -v curl >/dev/null 2>&1 || {
      echo "curl is required for run mode" >&2
      exit 2
    }
    command -v python3 >/dev/null 2>&1 || {
      echo "python3 is required for run mode" >&2
      exit 2
    }
    write_compose ""
    print_plan
    compose down --volumes --remove-orphans
    compose up -d mysql redis panel
    wait_for_http "http://127.0.0.1:${panel_host_port}/healthz" "panel_health"
    create_control_plane_state
    write_compose "$node_bootstrap_token"
    compose up -d node
    wait_for_pending_node
    approve_response="$(api_request POST "/nodes/${node_id}/approve" "$access_token" "$tenant_id" "{}")"
    node_access_token="$(printf '%s' "$approve_response" | json_get data.accessToken)"
    echo "node_approved=${node_id}"
    wait_for_node_bound
    create_access_path_state
    validate_latest_bootstrap
    run_db_evidence
    ;;
  clean)
    write_compose ""
    print_plan
    compose down --volumes --remove-orphans
    ;;
esac
REMOTE
