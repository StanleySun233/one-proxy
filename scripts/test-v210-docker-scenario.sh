#!/usr/bin/env bash
set -euo pipefail

mode="${1:-check}"
release="v2.1.0"
project="${ONEPROXY_V210_PROJECT:-oneproxy-v210-local}"
panel_repo="${ONEPROXY_PANEL_IMAGE_REPO:-ghcr.io/stanleysun233/oneproxy-panel}"
node_repo="${ONEPROXY_NODE_IMAGE_REPO:-ghcr.io/stanleysun233/oneproxy-node}"
tag="${ONEPROXY_IMAGE_TAG:-v2.1.0-rc.local}"
mysql_image="${ONEPROXY_MYSQL_IMAGE:-mysql:8.4}"
redis_image="${ONEPROXY_REDIS_IMAGE:-redis:7}"
mysql_db="${ONEPROXY_V210_DB:-oneproxy_v210}"
mysql_root_password="${ONEPROXY_V210_MYSQL_ROOT_PASSWORD:-oneproxy-v210-root}"
jwt_key="${ONEPROXY_V210_JWT_SIGNING_KEY:-oneproxy-v210-jwt}"
panel_host_port="${ONEPROXY_V210_PANEL_PORT:-12886}"
node_http_host_port="${ONEPROXY_V210_NODE_HTTP_PORT:-12988}"
node_https_host_port="${ONEPROXY_V210_NODE_HTTPS_PORT:-12989}"
node_tcp_host_port="${ONEPROXY_V210_NODE_TCP_PORT:-12990}"
node_udp_host_port="${ONEPROXY_V210_NODE_UDP_PORT:-12991}"
node_direct_host_port="${ONEPROXY_V210_NODE_DIRECT_PORT:-12992}"
mysql_host_port="${ONEPROXY_V210_MYSQL_PORT:-13306}"
redis_host_port="${ONEPROXY_V210_REDIS_PORT:-16379}"
compose_file=""

case "$mode" in
  check|build|run|clean)
    ;;
  *)
    echo "usage: $0 [check|build|run|clean]" >&2
    exit 2
    ;;
esac

cleanup() {
  if [ -n "$compose_file" ]; then
    rm -f "$compose_file"
  fi
}
trap cleanup EXIT

ensure_docker_compose() {
  if docker compose version >/dev/null 2>&1; then
    return 0
  fi
  if command -v docker-compose >/dev/null 2>&1; then
    return 0
  fi
  echo "docker compose is required" >&2
  exit 2
}

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -p "$project" -f "$compose_file" "$@"
  else
    docker-compose -p "$project" -f "$compose_file" "$@"
  fi
}

write_compose() {
  compose_file="$(mktemp)"
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
      NODE_NAME: "oneproxy-v210-local-node"
      NODE_MODE: "relay"
      NODE_SCOPE_KEY: "oneproxy-v210-local"
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
  echo "release=${release}"
  echo "mode=${mode}"
  echo "project=${project}"
  echo "services=mysql,redis,panel,node"
  echo "panel_image=${panel_repo}:${tag}"
  echo "node_image=${node_repo}:${tag}"
  echo "mysql_image=${mysql_image}"
  echo "redis_image=${redis_image}"
  echo "host_ports=panel:${panel_host_port},node_http:${node_http_host_port},node_https:${node_https_host_port},node_tcp:${node_tcp_host_port},node_udp:${node_udp_host_port},node_direct:${node_direct_host_port},mysql:${mysql_host_port},redis:${redis_host_port}"
  echo "destructive_modes=build,run,clean"
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

run_db_evidence() {
  compose exec -T mysql mysql -uroot -p"${mysql_root_password}" "${mysql_db}" -N -B -e "SELECT id, status, enabled, public_host, public_port FROM nodes ORDER BY id;" || true
  compose exec -T mysql mysql -uroot -p"${mysql_root_password}" "${mysql_db}" -N -B -e "SELECT node_id, transport_type, direction, address, status FROM node_transports ORDER BY node_id, transport_type;" || true
  compose exec -T mysql mysql -uroot -p"${mysql_root_password}" "${mysql_db}" -N -B -e "SELECT id, chain_id, entry_node_id, target_node_id, listen_port, target_host, target_port, enabled FROM node_access_paths ORDER BY id;" || true
  compose exec -T mysql mysql -uroot -p"${mysql_root_password}" "${mysql_db}" -N -B -e "SELECT id, priority, match_type, match_value, action_type, chain_id, destination_scope, enabled FROM route_rules ORDER BY priority, id;" || true
}

case "$mode" in
  check)
    print_plan
    if command -v docker >/dev/null 2>&1; then
      echo "docker=found"
    else
      echo "docker=missing"
    fi
    if docker compose version >/dev/null 2>&1 || command -v docker-compose >/dev/null 2>&1; then
      echo "docker_compose=found"
    else
      echo "docker_compose=missing"
    fi
    if command -v curl >/dev/null 2>&1; then
      echo "curl=found"
    else
      echo "curl=missing"
    fi
    ;;
  build)
    ensure_docker_compose
    print_plan
    docker build -f docker/one-proxy-panel-base.Dockerfile -t oneproxy-panel-base:"$tag" .
    docker build -f docker/one-proxy-panel.Dockerfile --build-arg PANEL_BASE_IMAGE=oneproxy-panel-base:"$tag" -t "${panel_repo}:${tag}" .
    docker build -f docker/one-proxy-node-base.Dockerfile -t oneproxy-node-base:"$tag" .
    docker build -f docker/one-proxy-node.Dockerfile --build-arg NODE_BASE_IMAGE=oneproxy-node-base:"$tag" -t "${node_repo}:${tag}" .
    ;;
  run)
    ensure_docker_compose
    command -v curl >/dev/null 2>&1 || {
      echo "curl is required for run mode" >&2
      exit 2
    }
    write_compose
    print_plan
    compose up -d
    wait_for_http "http://127.0.0.1:${panel_host_port}/healthz" "panel_health"
    wait_for_http "http://127.0.0.1:${node_http_host_port}/healthz" "node_health"
    run_db_evidence
    ;;
  clean)
    ensure_docker_compose
    write_compose
    print_plan
    compose down --volumes --remove-orphans
    ;;
esac
