#!/usr/bin/env bash
set -euo pipefail

mode="${1:-check}"
target="${2:-all}"
tag="${3:-${ONEPROXY_IMAGE_TAG:-}}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
node_repo="${ONEPROXY_NODE_IMAGE_REPO:-ghcr.io/stanleysun233/oneproxy-node}"
panel_repo="${ONEPROXY_PANEL_IMAGE_REPO:-ghcr.io/stanleysun233/oneproxy-panel}"
local_node_container="${ONEPROXY_LOCAL_NODE_CONTAINER:-one-proxy-node}"
local_node_network="${ONEPROXY_LOCAL_NODE_NETWORK:-}"
local_node_volume="${ONEPROXY_LOCAL_NODE_VOLUME:-one-proxy-node-runtime}"
camelbot_ssh_host="${CAMELBOT_SSH_HOST:-camelbot}"
camelbot_panel_container="${ONEPROXY_PANEL_CONTAINER:-one-proxy-panel}"
camelbot_panel_network="${ONEPROXY_PANEL_NETWORK:-one-proxy-net}"
camelbot_panel_volume="${ONEPROXY_PANEL_DATA_VOLUME:-one-proxy-panel-data}"
camelbot_panel_port="${ONEPROXY_PANEL_PORT:-2886}"
camelbot_mysql_container="${ONEPROXY_MYSQL_CONTAINER:-one-proxy-mysql8}"
camelbot_panel_db_name="${ONEPROXY_DB_NAME:-one_proxy}"
camelbot_final_panel_db_name="${ONEPROXY_FINAL_PANEL_DB_NAME:-}"
camelbot_final_panel_volume="${ONEPROXY_FINAL_PANEL_DATA_VOLUME:-one-proxy-panel-data-v210-final}"
camelbot_final_admin_password="${ONEPROXY_FINAL_PANEL_ADMIN_PASSWORD:-}"
camelbot_final_jwt_signing_key="${ONEPROXY_FINAL_PANEL_JWT_SIGNING_KEY:-}"
camelbot_final_schema_confirm="${ONEPROXY_FINAL_SCHEMA_CONFIRM:-}"

case "$mode" in
  check|dry-run|test|deploy)
    ;;
  *)
    echo "usage: $0 [check|dry-run|test|deploy] [all|local-node|camelbot-node|camelbot-panel|camelbot-full-chain] <immutable_tag>" >&2
    exit 2
    ;;
esac

case "$target" in
  all|local-node|camelbot-node|camelbot-panel|camelbot-full-chain)
    ;;
  *)
    echo "unknown target: $target" >&2
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

panel_image() {
  printf "%s:%s" "$panel_repo" "$tag"
}

prune_camelbot_docker() {
  ssh -T "$camelbot_ssh_host" 'bash -s' <<'REMOTE'
set -euo pipefail
echo "docker_df_before"
docker system df || true
docker container prune -f >/dev/null || true
docker image prune -af --filter "until=24h" >/dev/null || true
docker builder prune -af >/dev/null || true
docker network prune -f >/dev/null || true
echo "docker_df_after"
docker system df || true
REMOTE
}

append_node_env_defaults() {
  env_file="$1"
  grep -q '^NODE_FORWARD_RETRY_BODY_MAX_BYTES=' "$env_file" || echo "NODE_FORWARD_RETRY_BODY_MAX_BYTES=8mb" >> "$env_file"
  grep -q '^NODE_TCP_ACCESS_MAX_SESSIONS=' "$env_file" || echo "NODE_TCP_ACCESS_MAX_SESSIONS=4096" >> "$env_file"
  grep -q '^NODE_UDP_ACCESS_MAX_IN_FLIGHT=' "$env_file" || echo "NODE_UDP_ACCESS_MAX_IN_FLIGHT=1024" >> "$env_file"
  grep -q '^NODE_UDP_ACCESS_TIMEOUT=' "$env_file" || echo "NODE_UDP_ACCESS_TIMEOUT=15s" >> "$env_file"
  reverse_target="$(awk -F= '$1 == "NODE_REVERSE_TARGET_URL" {sub(/^[^=]*=/, "", $0); print; exit}' "$env_file")"
  reverse_path="$(awk -F= '$1 == "NODE_REVERSE_ACCESS_PATH_ID" {sub(/^[^=]*=/, "", $0); print; exit}' "$env_file")"
  if [ -n "$reverse_target" ] && [ -z "$reverse_path" ]; then
    echo "NODE_REVERSE_ACCESS_PATH_ID is required when NODE_REVERSE_TARGET_URL is set" >&2
    exit 2
  fi
}

check_local_node() {
  echo "target=local-node"
  echo "image=$(node_image)"
  if ! command -v docker >/dev/null 2>&1; then
    echo "docker=missing"
    return 0
  fi
  docker inspect "$local_node_container" --format 'current={{.Config.Image}} status={{.State.Status}} started={{.State.StartedAt}}' 2>/dev/null || echo "current=missing"
}

deploy_local_node() {
  image="$(node_image)"
  backup="${local_node_container}-prev-$(date +%Y%m%d%H%M%S)"
  env_file="$(mktemp)"
  network="$local_node_network"
  if [ -z "$network" ]; then
    network="$(docker inspect "$local_node_container" --format '{{range $name, $_ := .NetworkSettings.Networks}}{{println $name}}{{end}}' | head -n 1)"
  fi
  network_args=()
  if [ -n "$network" ]; then
    network_args=(--network "$network")
  fi
  renamed=0
  completed=0
  rollback() {
    status=$?
    rm -f "$env_file"
    if [ "$completed" -eq 0 ] && [ "$renamed" -eq 1 ]; then
      docker rm -f "$local_node_container" >/dev/null 2>&1 || true
      if docker ps -a --format '{{.Names}}' | grep -Fxq "$backup"; then
        docker rename "$backup" "$local_node_container" >/dev/null 2>&1 || true
        docker start "$local_node_container" >/dev/null 2>&1 || true
      fi
    fi
    exit "$status"
  }
  trap rollback ERR HUP INT TERM

  docker pull "$image" >/dev/null
  docker inspect "$local_node_container" --format '{{range .Config.Env}}{{println .}}{{end}}' | grep -Ev '^(PATH|GOLANG_VERSION|GOTOOLCHAIN|GOPATH|ZONEINFO)=' > "$env_file"
  append_node_env_defaults "$env_file"
  docker rename "$local_node_container" "$backup"
  renamed=1
  docker stop "$backup" >/dev/null
  docker run -d \
    --name "$local_node_container" \
    --restart unless-stopped \
    "${network_args[@]}" \
    --env-file "$env_file" \
    -v "${local_node_volume}:/app/runtime" \
    -p 2988:2988 \
    -p 2989:2989 \
    -p 2990:2990 \
    -p 2991:2991/udp \
    -p 2992:2992/udp \
    "$image" >/dev/null
  tries=0
  until curl -fsS http://127.0.0.1:2988/healthz >/dev/null 2>&1; do
    tries=$((tries + 1))
    if [ "$tries" -ge 45 ]; then
      echo "local node health failed" >&2
      exit 1
    fi
    sleep 2
  done
  completed=1
  rm -f "$env_file"
  trap - ERR HUP INT TERM
  echo "target=local-node"
  echo "backup=${backup}"
  docker inspect "$local_node_container" --format 'current={{.Config.Image}} status={{.State.Status}} started={{.State.StartedAt}}'
}

check_camelbot_node() {
  echo "target=camelbot-node"
  echo "image=$(node_image)"
  ONEPROXY_NODE_IMAGE_REPO="$node_repo" "$script_dir/deploy-camelbot-node.sh" check
}

deploy_camelbot_node() {
  echo "target=camelbot-node"
  ONEPROXY_NODE_IMAGE_REPO="$node_repo" "$script_dir/deploy-camelbot-node.sh" deploy "$tag"
}

test_camelbot_full_chain() {
  echo "target=camelbot-full-chain"
  ONEPROXY_PANEL_IMAGE_REPO="$panel_repo" ONEPROXY_NODE_IMAGE_REPO="$node_repo" "$script_dir/test-camelbot-full-chain.sh" run "$tag"
}

check_camelbot_panel() {
  echo "target=camelbot-panel"
  echo "image=$(panel_image)"
  ssh -T "$camelbot_ssh_host" 'bash -s' -- "$camelbot_panel_container" "$camelbot_panel_port" "$camelbot_mysql_container" "$camelbot_panel_db_name" <<'REMOTE'
set -euo pipefail
container="$1"
port="$2"
mysql_container="$3"
db_name="$4"
docker inspect "$container" --format 'current={{.Config.Image}} status={{.State.Status}} started={{.State.StartedAt}}' 2>/dev/null || echo "current=missing"
curl -fsS "http://127.0.0.1:${port}/healthz" >/dev/null && echo "health=ok"
mysql_query() {
  docker exec -e MYSQL_QUERY="$1" -e MYSQL_DATABASE="$db_name" "$mysql_container" sh -lc 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" "$MYSQL_DATABASE" -N -B -e "$MYSQL_QUERY"' 2>/dev/null
}
tables="$(mysql_query "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE();" || echo unknown)"
goose="$(mysql_query "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'goose_db_version';" || echo unknown)"
access_paths="$(mysql_query "SELECT COUNT(*) FROM node_access_paths;" || echo unknown)"
echo "db=${db_name} tables=${tables} goose_tables=${goose} access_paths=${access_paths}"
REMOTE
}

deploy_camelbot_panel() {
  image="$(panel_image)"
  if [ -z "$camelbot_final_panel_db_name" ]; then
    echo "camelbot panel final-schema deploy requires ONEPROXY_FINAL_PANEL_DB_NAME" >&2
    exit 2
  fi
  if [ "$camelbot_final_schema_confirm" != "deploy-final-schema" ]; then
    echo "camelbot panel final-schema deploy requires ONEPROXY_FINAL_SCHEMA_CONFIRM=deploy-final-schema" >&2
    exit 2
  fi
  if [ -z "$camelbot_final_admin_password" ] || [ -z "$camelbot_final_jwt_signing_key" ]; then
    echo "camelbot panel final-schema deploy requires ONEPROXY_FINAL_PANEL_ADMIN_PASSWORD and ONEPROXY_FINAL_PANEL_JWT_SIGNING_KEY" >&2
    exit 2
  fi
  ssh -T "$camelbot_ssh_host" 'bash -s' -- "$image" "$camelbot_panel_container" "$camelbot_panel_network" "$camelbot_final_panel_volume" "$camelbot_panel_port" "$camelbot_mysql_container" "$camelbot_final_panel_db_name" "$camelbot_final_admin_password" "$camelbot_final_jwt_signing_key" <<'REMOTE'
set -Eeuo pipefail
image="$1"
container="$2"
network="$3"
volume="$4"
port="$5"
mysql_container="$6"
db_name="$7"
admin_password="$8"
jwt_signing_key="$9"
backup="${container}-prev-$(date +%Y%m%d%H%M%S)"
env_file="$(mktemp)"
renamed=0
completed=0
rollback() {
  status=$?
  rm -f "$env_file"
  if [ "$completed" -eq 0 ] && [ "$renamed" -eq 1 ]; then
    docker rm -f "$container" >/dev/null 2>&1 || true
    if docker ps -a --format '{{.Names}}' | grep -Fxq "$backup"; then
      docker rename "$backup" "$container" >/dev/null 2>&1 || true
      docker start "$container" >/dev/null 2>&1 || true
    fi
  fi
  exit "$status"
}
trap rollback ERR HUP INT TERM

sql_identifier() {
  printf "%s" "$1" | sed 's/`/``/g'
}

sql_literal() {
  printf "%s" "$1" | sed "s/'/''/g"
}

mysql_admin() {
  docker exec -e MYSQL_QUERY="$1" "$mysql_container" sh -lc 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -N -B -e "$MYSQL_QUERY"'
}

escaped_db="$(sql_identifier "$db_name")"
escaped_db_literal="$(sql_literal "$db_name")"
mysql_admin "CREATE DATABASE IF NOT EXISTS \`${escaped_db}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
table_count="$(docker exec -e MYSQL_QUERY="SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '${escaped_db_literal}';" "$mysql_container" sh -lc 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -N -B -e "$MYSQL_QUERY"')"
if [ "${table_count}" != "0" ]; then
  echo "final schema database is not empty: ${db_name} tables=${table_count}" >&2
  exit 2
fi
mysql_root_password="$(docker exec "$mysql_container" sh -lc 'printf "%s" "$MYSQL_ROOT_PASSWORD"')"
{
  echo "MYSQL_DSN=root:${mysql_root_password}@tcp(${mysql_container}:3306)/${db_name}?charset=utf8mb4&parseTime=true&loc=UTC"
  echo "ADMIN_PASSWORD=${admin_password}"
  echo "JWT_SIGNING_KEY=${jwt_signing_key}"
  echo "CONTROL_PLANE_URL=http://127.0.0.1:2887"
} > "$env_file"

docker pull "$image" >/dev/null
docker rename "$container" "$backup"
renamed=1
docker stop "$backup" >/dev/null
docker run -d \
  --name "$container" \
  --restart unless-stopped \
  --network "$network" \
  --env-file "$env_file" \
  -v "${volume}:/app/data" \
  -p "${port}:2886" \
  "$image" >/dev/null
tries=0
until curl -fsS "http://127.0.0.1:${port}/healthz" >/dev/null 2>&1; do
  tries=$((tries + 1))
  if [ "$tries" -ge 45 ]; then
    echo "panel health failed" >&2
    exit 1
  fi
  sleep 2
done
completed=1
rm -f "$env_file"
trap - ERR HUP INT TERM
echo "target=camelbot-panel"
echo "backup=${backup}"
echo "db=${db_name}"
echo "volume=${volume}"
docker inspect "$container" --format 'current={{.Config.Image}} status={{.State.Status}} started={{.State.StartedAt}}'
REMOTE
}

run_target() {
  item="$1"
  case "$mode:$item" in
    dry-run:local-node)
      echo "would_deploy=local-node image=$(node_image)"
      ;;
    dry-run:camelbot-node)
      echo "would_deploy=camelbot-node image=$(node_image)"
      ;;
    dry-run:camelbot-panel)
      echo "would_deploy=camelbot-panel image=$(panel_image)"
      echo "final_db=${camelbot_final_panel_db_name:-<required>}"
      echo "final_volume=${camelbot_final_panel_volume}"
      ;;
    dry-run:camelbot-full-chain)
      echo "would_test=camelbot-full-chain panel_image=$(panel_image) node_image=$(node_image)"
      ;;
    check:local-node)
      check_local_node
      ;;
    check:camelbot-node)
      check_camelbot_node
      ;;
    check:camelbot-panel)
      check_camelbot_panel
      ;;
    check:camelbot-full-chain)
      echo "target=camelbot-full-chain"
      echo "script=${script_dir}/test-camelbot-full-chain.sh"
      ;;
    test:camelbot-full-chain)
      test_camelbot_full_chain
      ;;
    test:all)
      test_camelbot_full_chain
      ;;
    deploy:local-node)
      deploy_local_node
      ;;
    deploy:camelbot-node)
      deploy_camelbot_node
      ;;
    deploy:camelbot-panel)
      deploy_camelbot_panel
      ;;
  esac
}

require_tag

if [ "$target" = "all" ]; then
  if [ "$mode" = "deploy" ]; then
    prune_camelbot_docker
    test_camelbot_full_chain
    prune_camelbot_docker
    run_target camelbot-panel
    run_target camelbot-node
    run_target local-node
  elif [ "$mode" = "test" ]; then
    test_camelbot_full_chain
  else
    run_target camelbot-panel
    run_target camelbot-node
    run_target local-node
  fi
else
  run_target "$target"
fi
