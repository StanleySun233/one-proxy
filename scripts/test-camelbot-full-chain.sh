#!/usr/bin/env bash
set -euo pipefail

mode="${1:-run}"
tag="${2:-${ONEPROXY_IMAGE_TAG:-}}"
ssh_host="${CAMELBOT_SSH_HOST:-camelbot}"
panel_repo="${ONEPROXY_PANEL_IMAGE_REPO:-ghcr.io/stanleysun233/oneproxy-panel}"
node_repo="${ONEPROXY_NODE_IMAGE_REPO:-ghcr.io/stanleysun233/oneproxy-node}"
mysql_container="${ONEPROXY_MYSQL_CONTAINER:-one-proxy-mysql8}"
network="${ONEPROXY_CHAIN_TEST_NETWORK:-one-proxy-chain-test-net}"
panel_container="${ONEPROXY_CHAIN_TEST_PANEL_CONTAINER:-one-proxy-chain-test-panel}"
node_container="${ONEPROXY_CHAIN_TEST_NODE_CONTAINER:-one-proxy-chain-test-node}"
origin_container="${ONEPROXY_CHAIN_TEST_ORIGIN_CONTAINER:-one-proxy-chain-test-origin}"
panel_port="${ONEPROXY_CHAIN_TEST_PANEL_PORT:-3886}"
node_port="${ONEPROXY_CHAIN_TEST_NODE_PORT:-3988}"
db_name="${ONEPROXY_CHAIN_TEST_DB:-one_proxy_chain_test}"

case "$mode" in
  run|cleanup|prune)
    ;;
  *)
    echo "usage: $0 [run|cleanup|prune] <immutable_tag>" >&2
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

ssh -T "$ssh_host" 'bash -s' -- "$mode" "${tag:-}" "$(panel_image 2>/dev/null || true)" "$(node_image 2>/dev/null || true)" "$mysql_container" "$network" "$panel_container" "$node_container" "$origin_container" "$panel_port" "$node_port" "$db_name" <<'REMOTE'
set -Eeuo pipefail

mode="$1"
tag="$2"
panel_image="$3"
node_image="$4"
mysql_container="$5"
network="$6"
panel_container="$7"
node_container="$8"
origin_container="$9"
panel_port="${10}"
node_port="${11}"
db_name="${12}"

docker_prune_safe() {
  echo "docker_df_before"
  docker system df || true
  docker container prune -f >/dev/null || true
  docker image prune -af --filter "until=24h" >/dev/null || true
  docker builder prune -af >/dev/null || true
  docker network prune -f >/dev/null || true
  echo "docker_df_after"
  docker system df || true
}

cleanup_test() {
  docker rm -f "$node_container" "$panel_container" "$origin_container" >/dev/null 2>&1 || true
  docker network disconnect "$network" "$mysql_container" >/dev/null 2>&1 || true
  docker network rm "$network" >/dev/null 2>&1 || true
  if docker inspect "$mysql_container" >/dev/null 2>&1; then
    escaped="$(printf "%s" "$db_name" | sed 's/`/``/g')"
    docker exec -e MYSQL_QUERY="DROP DATABASE IF EXISTS \`${escaped}\`;" "$mysql_container" sh -lc 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -N -B -e "$MYSQL_QUERY"' >/dev/null 2>&1 || true
  fi
}

if [ "$mode" = "cleanup" ]; then
  cleanup_test
  exit 0
fi

if [ "$mode" = "prune" ]; then
  docker_prune_safe
  exit 0
fi

if [ -z "$tag" ] || [ "$tag" = "latest" ] || [ "$tag" = "main" ] || [ "$tag" = "master" ] || [ "$tag" = "dev" ] || [ "$tag" = "nightly" ] || [ "$tag" = "stable" ]; then
  echo "immutable image tag is required" >&2
  exit 2
fi

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
    item_name = item.get("name") or item.get("nodeName") or ""
    if str(item_name) == name:
        print(item.get("id", ""))
        break
' "$1"
}

sql_quote() {
  printf "%s" "$1" | sed "s/'/''/g"
}

mysql_query() {
  docker exec -e MYSQL_QUERY="$1" -e MYSQL_DATABASE="$db_name" "$mysql_container" sh -lc 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" "$MYSQL_DATABASE" -N -B -e "$MYSQL_QUERY"' 2>/dev/null
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
  access_token="$2"
  tenant_id="$3"
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

wait_proxy_ok() {
  proxy_token="$1"
  tries=0
  while :; do
    body="$(curl -fsS -x "http://127.0.0.1:${node_port}" -H "Proxy-Authorization: Bearer ${proxy_token}" "http://${origin_container}:18080/chain-test" 2>/dev/null || true)"
    if [ "$body" = "oneproxy-chain-test-ok" ]; then
      echo "proxy_chain_traffic=ok"
      return 0
    fi
    tries=$((tries + 1))
    if [ "$tries" -ge 60 ]; then
      echo "proxy_chain_traffic=failed body=${body}" >&2
      exit 1
    fi
    sleep 2
  done
}

trap cleanup_test EXIT
cleanup_test
docker_prune_safe

docker network create "$network" >/dev/null
if ! docker inspect "$mysql_container" --format '{{json .NetworkSettings.Networks}}' | grep -q "\"${network}\""; then
  docker network connect "$network" "$mysql_container" >/dev/null 2>&1 || true
fi

escaped_db="$(printf "%s" "$db_name" | sed 's/`/``/g')"
mysql_root_password="$(docker exec "$mysql_container" sh -lc 'printf "%s" "$MYSQL_ROOT_PASSWORD"')"
docker exec -e MYSQL_QUERY="CREATE DATABASE IF NOT EXISTS \`${escaped_db}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" "$mysql_container" sh -lc 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -N -B -e "$MYSQL_QUERY"' >/dev/null

admin_password="ChainTest-${tag}-Admin-$(date +%s)"
jwt_signing_key="chain-test-${tag}-jwt-$(date +%s%N)"
panel_env="$(mktemp)"
node_env="$(mktemp)"
cleanup_files() {
  rm -f "$panel_env" "$node_env"
}
trap 'cleanup_files; cleanup_test' EXIT

{
  echo "MYSQL_DSN=root:${mysql_root_password}@tcp(${mysql_container}:3306)/${db_name}?charset=utf8mb4&parseTime=true&loc=UTC"
  echo "ADMIN_PASSWORD=${admin_password}"
  echo "JWT_SIGNING_KEY=${jwt_signing_key}"
  echo "CONTROL_PLANE_URL=http://127.0.0.1:2887"
} > "$panel_env"

docker pull "$panel_image" >/dev/null
docker run -d \
  --name "$panel_container" \
  --network "$network" \
  --env-file "$panel_env" \
  -p "${panel_port}:2886" \
  "$panel_image" >/dev/null
wait_for_http "http://127.0.0.1:${panel_port}/healthz" "panel_health"

docker pull python:3.12-alpine >/dev/null
docker run -d \
  --name "$origin_container" \
  --network "$network" \
  python:3.12-alpine \
  python -c 'from http.server import BaseHTTPRequestHandler,HTTPServer
class H(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b"oneproxy-chain-test-ok")
    def log_message(self, *args):
        pass
HTTPServer(("0.0.0.0",18080),H).serve_forever()' >/dev/null

login_response="$(api_request POST /auth/login "" "" "{\"account\":\"admin\",\"password\":\"${admin_password}\"}")"
access_token="$(printf "%s" "$login_response" | json_get data.accessToken)"
account_id="$(printf "%s" "$login_response" | json_get data.account.id)"
tenant_response="$(api_request POST /tenants "$access_token" "" "{\"name\":\"Chain Test ${tag}\",\"initialAdminAccountId\":\"${account_id}\"}")"
tenant_id="$(printf "%s" "$tenant_response" | json_get data.tenant.id)"
scope_response="$(api_request POST /proxy/scopes "$access_token" "$tenant_id" "{\"name\":\"chain-test-scope\",\"description\":\"camelbot full chain test\"}")"
scope_id="$(printf "%s" "$scope_response" | json_get data.id)"
unused_bootstrap_response="$(api_request POST /nodes/bootstrap/token "$access_token" "$tenant_id" "{\"targetType\":\"node\",\"nodeName\":\"chain-test-unused-node\",\"nodeMode\":\"relay\",\"scopeKey\":\"${scope_id}\"}")"
unused_bootstrap_id="$(printf "%s" "$unused_bootstrap_response" | json_get data.id)"
unused_node_id="$(printf "%s" "$unused_bootstrap_response" | json_get data.targetId)"
pending_before="$(api_request GET /nodes/pending "$access_token" "$tenant_id")"
if [ -n "$(printf "%s" "$pending_before" | node_id_by_name chain-test-unused-node)" ]; then
  echo "bootstrap_lifecycle_contract=failed pending_before_connect" >&2
  exit 1
fi
unconsumed_before="$(api_request GET /nodes/bootstrap/tokens/unconsumed "$access_token" "$tenant_id")"
if [ -z "$(printf "%s" "$unconsumed_before" | node_id_by_name chain-test-unused-node)" ]; then
  echo "bootstrap_lifecycle_contract=failed missing_unconsumed_token" >&2
  exit 1
fi
pre_approve_code="$(curl -sS -o /tmp/oneproxy-chain-test-preapprove -w '%{http_code}' -X POST "http://127.0.0.1:${panel_port}/api/nodes/${unused_node_id}/approve" -H "Content-Type: application/json" -H "X-One-Proxy-Access-Token: ${access_token}" -H "X-One-Proxy-Tenant-ID: ${tenant_id}" -d "{}" || true)"
if [ "$pre_approve_code" != "400" ]; then
  echo "bootstrap_lifecycle_contract=failed pre_approve_code=${pre_approve_code}" >&2
  exit 1
fi
api_request DELETE "/nodes/bootstrap/tokens/${unused_bootstrap_id}" "$access_token" "$tenant_id" "" >/dev/null
quoted_unused_node_id="$(sql_quote "$unused_node_id")"
remaining_unused_nodes="$(mysql_query "SELECT COUNT(*) FROM nodes WHERE id='${quoted_unused_node_id}';")"
if [ "${remaining_unused_nodes:-0}" != "0" ]; then
  echo "bootstrap_lifecycle_contract=failed placeholder_node_remaining=${remaining_unused_nodes}" >&2
  exit 1
fi
echo "bootstrap_lifecycle_contract=ok"
bootstrap_response="$(api_request POST /nodes/bootstrap/token "$access_token" "$tenant_id" "{\"targetType\":\"node\",\"nodeName\":\"chain-test-node\",\"nodeMode\":\"edge\",\"scopeKey\":\"${scope_id}\",\"publicHost\":\"127.0.0.1\",\"publicPort\":2988}")"
bootstrap_token="$(printf "%s" "$bootstrap_response" | json_get data.token)"

{
  echo "NODE_BOOTSTRAP_TOKEN=${bootstrap_token}"
  echo "NODE_PARENT_URL=http://${panel_container}:2886"
  echo "NODE_NAME=chain-test-node"
  echo "NODE_MODE=edge"
  echo "NODE_PUBLIC_HOST=127.0.0.1"
  echo "NODE_LISTEN_ADDR=:2988"
  echo "NODE_HTTPS_LISTEN_ADDR=:2989"
  echo "NODE_TCP_ACCESS_LISTEN_ADDR=:2990"
  echo "NODE_TCP_ACCESS_MAX_SESSIONS=64"
  echo "NODE_UDP_ACCESS_LISTEN_ADDR=:2991"
  echo "NODE_UDP_ACCESS_MAX_IN_FLIGHT=64"
  echo "NODE_UDP_ACCESS_TIMEOUT=15s"
  echo "NODE_DIRECT_LISTEN_ADDR="
  echo "NODE_HEARTBEAT_INTERVAL=2s"
  echo "NODE_FORWARD_RETRY_BODY_MAX_BYTES=8mb"
  echo "NODE_POLICY_STATE_PATH=runtime/node-policy-state.json"
  echo "NODE_RUNTIME_CONFIG_PATH=runtime/node-runtime.json"
} > "$node_env"

docker pull "$node_image" >/dev/null
docker run -d \
  --name "$node_container" \
  --network "$network" \
  --env-file "$node_env" \
  -p "${node_port}:2988" \
  "$node_image" >/dev/null

node_id="$(wait_pending_node chain-test-node "$access_token" "$tenant_id")"
approve_response="$(api_request POST "/nodes/${node_id}/approve" "$access_token" "$tenant_id" "{}")"
node_token="$(printf "%s" "$approve_response" | json_get data.accessToken)"
tries=0
until curl -fsS "http://127.0.0.1:${node_port}/healthz" 2>/dev/null | grep -q '"controlPlaneBound":true'; do
  tries=$((tries + 1))
  if [ "$tries" -ge 60 ]; then
    echo "node_bound=failed" >&2
    exit 1
  fi
  sleep 2
done
echo "node_bound=ok"

chain_response="$(api_request POST /proxy "$access_token" "$tenant_id" "{\"name\":\"chain-test-chain\",\"destinationScope\":\"${scope_id}\",\"hops\":[\"${node_id}\"]}")"
chain_id="$(printf "%s" "$chain_response" | json_get data.id)"
path_body="{\"chainId\":\"${chain_id}\",\"name\":\"chain-test-forward-path\",\"mode\":\"forward\",\"protocol\":\"http\",\"serviceType\":\"http_forward_proxy\",\"targetNodeId\":\"${node_id}\",\"entryNodeId\":\"${node_id}\",\"relayNodeIds\":[],\"listenHost\":\"127.0.0.1\",\"listenPort\":2988,\"targetProtocol\":\"http\",\"targetHost\":\"${origin_container}\",\"targetPort\":18080,\"targetSni\":\"\",\"tlsMode\":\"\",\"authMode\":\"proxy_token\",\"options\":{}}"
path_response="$(api_request POST /proxy/paths "$access_token" "$tenant_id" "$path_body")"
access_path_id="$(printf "%s" "$path_response" | json_get data.id)"
group_response="$(api_request POST /proxy/route-groups "$access_token" "$tenant_id" "{\"name\":\"chain-test-routes\",\"description\":\"camelbot full chain test\"}")"
route_group_id="$(printf "%s" "$group_response" | json_get data.id)"
chain_route_response="$(api_request POST /proxy/routes "$access_token" "$tenant_id" "{\"groupId\":\"${route_group_id}\",\"priority\":10,\"matchType\":\"domain\",\"matchValue\":\"${origin_container}\",\"actionType\":\"chain\",\"chainId\":\"${chain_id}\",\"destinationScope\":\"\"}")"
chain_route_id="$(printf "%s" "$chain_route_response" | json_get data.id)"
direct_route_response="$(api_request POST /proxy/routes "$access_token" "$tenant_id" "{\"groupId\":\"${route_group_id}\",\"priority\":20,\"matchType\":\"domain\",\"matchValue\":\"direct.chain-test.invalid\",\"actionType\":\"direct\",\"chainId\":\"\",\"destinationScope\":\"${scope_id}\"}")"
direct_route_id="$(printf "%s" "$direct_route_response" | json_get data.id)"

if curl -fsS -X POST "http://127.0.0.1:${panel_port}/api/proxy/routes" -H "Content-Type: application/json" -H "X-One-Proxy-Access-Token: ${access_token}" -H "X-One-Proxy-Tenant-ID: ${tenant_id}" -d "{\"groupId\":\"${route_group_id}\",\"priority\":30,\"matchType\":\"url_regex\",\"matchValue\":\".*\",\"actionType\":\"direct\",\"chainId\":\"\",\"destinationScope\":\"${scope_id}\"}" >/dev/null 2>&1; then
  echo "unsupported_match_type_rejected=failed" >&2
  exit 1
fi
echo "unsupported_match_type_rejected=ok"

api_request POST /policies/publish "$access_token" "$tenant_id" "{}" >/dev/null
bootstrap_payload="$(api_request GET /proxy/extension/bootstrap "$access_token" "$tenant_id")"
proxy_token="$(printf "%s" "$bootstrap_payload" | json_get data.proxyToken)"
token_hash="$(sha256_hex "$proxy_token")"

node_api_request POST /node/agent/proxy/token/authenticate "$node_token" "{\"tokenHash\":\"${token_hash}\"}" >/dev/null
node_api_request POST /node/agent/proxy/token/validate "$node_token" "{\"tokenHash\":\"${token_hash}\",\"accessPathId\":\"${access_path_id}\",\"targetHost\":\"${origin_container}\",\"targetPort\":18080,\"protocol\":\"http\",\"routeId\":\"${chain_route_id}\"}" >/dev/null
node_api_request POST /node/agent/proxy/token/validate "$node_token" "{\"tokenHash\":\"${token_hash}\",\"accessPathId\":\"\",\"targetHost\":\"direct.chain-test.invalid\",\"targetPort\":80,\"protocol\":\"http\",\"routeId\":\"${direct_route_id}\"}" >/dev/null
node_api_request POST /node/agent/proxy/token/validate "$node_token" "{\"tokenHash\":\"${token_hash}\",\"accessPathId\":\"${access_path_id}\",\"targetHost\":\"${origin_container}\",\"targetPort\":18080,\"protocol\":\"http\",\"routeId\":\"\"}" >/dev/null
echo "token_validate_contract=ok"

wait_proxy_ok "$proxy_token"
no_match_code="$(curl -sS -o /tmp/oneproxy-chain-test-no-match -w '%{http_code}' -x "http://127.0.0.1:${node_port}" -H "Proxy-Authorization: Bearer ${proxy_token}" "http://no-match.chain-test.invalid/" || true)"
if [ "$no_match_code" != "403" ]; then
  echo "no_match_deny=failed code=${no_match_code}" >&2
  exit 1
fi
echo "no_match_deny=ok"

printf "%s" "$bootstrap_payload" | python3 -c 'import json,sys
payload=json.load(sys.stdin)["data"]
assert payload.get("schemaVersion") == "v2.1.0", payload
assert "groups" not in payload, payload.keys()
assert len(payload.get("nodes") or []) >= 1, payload
assert len(payload.get("accessPaths") or []) >= 1, payload
assert len(payload.get("routes") or []) >= 2, payload
assert "url_regex" not in payload.get("routeEvaluation", {}).get("supportedMatchTypes", []), payload
print("extension_bootstrap=ok")
'

echo "camelbot_full_chain_test=ok"
REMOTE
