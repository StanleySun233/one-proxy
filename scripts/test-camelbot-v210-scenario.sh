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

ssh -T "${ssh_opts[@]}" "$ssh_host" 'bash -s' -- \
  "$mode" "$remote_dir" "$project" "$tag" "$panel_repo" "$node_repo" "$mysql_image" "$redis_image" \
  "$mysql_db" "$mysql_root_password" "$jwt_key" "$panel_host_port" "$node_http_host_port" \
  "$node_https_host_port" "$node_tcp_host_port" "$node_udp_host_port" "$node_direct_host_port" \
  "$mysql_host_port" "$redis_host_port" <<'REMOTE'
set -euo pipefail

mode="$1"
remote_dir="$2"
project="$3"
tag="$4"
panel_repo="$5"
node_repo="$6"
mysql_image="$7"
redis_image="$8"
mysql_db="$9"
mysql_root_password="${10}"
jwt_key="${11}"
panel_host_port="${12}"
node_http_host_port="${13}"
node_https_host_port="${14}"
node_tcp_host_port="${15}"
node_udp_host_port="${16}"
node_direct_host_port="${17}"
mysql_host_port="${18}"
redis_host_port="${19}"
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
      JWT_SIGNING_KEY: "${jwt_key}"
      REDIS_URL: "redis://redis:6379/0"
      PORT: "2886"
      HTTP_ADDR: "127.0.0.1:2887"
      CONTROL_PLANE_URL: "http://127.0.0.1:2887"
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
      NODE_NAME: "oneproxy-v210-camelbot-node"
      NODE_MODE: "relay"
      NODE_SCOPE_KEY: "oneproxy-v210-camelbot"
      NODE_PUBLIC_HOST: "127.0.0.1"
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
  echo "host_ports=panel:${panel_host_port},node_http:${node_http_host_port},node_https:${node_https_host_port},node_tcp:${node_tcp_host_port},node_udp:${node_udp_host_port},node_direct:${node_direct_host_port},mysql:${mysql_host_port},redis:${redis_host_port}"
  echo "production_replacement=false"
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
    if command -v docker >/dev/null 2>&1; then
      docker ps -a --format '{{.Names}}' | grep -E "^${project}[-_]" || true
    fi
    ;;
  build)
    write_compose
    print_plan
    compose pull mysql redis panel node
    ;;
  run)
    write_compose
    print_plan
    compose up -d
    wait_for_http "http://127.0.0.1:${panel_host_port}/healthz" "panel_health"
    wait_for_http "http://127.0.0.1:${node_http_host_port}/healthz" "node_health"
    ;;
  clean)
    write_compose
    print_plan
    compose down --volumes --remove-orphans
    ;;
esac
REMOTE
