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

case "$mode" in
  check|dry-run|deploy)
    ;;
  *)
    echo "usage: $0 [check|dry-run|deploy] [all|local-node|camelbot-node|camelbot-panel] <immutable_tag>" >&2
    exit 2
    ;;
esac

case "$target" in
  all|local-node|camelbot-node|camelbot-panel)
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
  curl -fsS http://127.0.0.1:2988/healthz >/dev/null
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

check_camelbot_panel() {
  echo "target=camelbot-panel"
  echo "image=$(panel_image)"
  ssh -T "$camelbot_ssh_host" 'bash -s' -- "$camelbot_panel_container" "$camelbot_panel_port" <<'REMOTE'
set -euo pipefail
container="$1"
port="$2"
docker inspect "$container" --format 'current={{.Config.Image}} status={{.State.Status}} started={{.State.StartedAt}}' 2>/dev/null || echo "current=missing"
curl -fsS "http://127.0.0.1:${port}/healthz" >/dev/null && echo "health=ok"
REMOTE
}

deploy_camelbot_panel() {
  image="$(panel_image)"
  ssh -T "$camelbot_ssh_host" 'bash -s' -- "$image" "$camelbot_panel_container" "$camelbot_panel_network" "$camelbot_panel_volume" "$camelbot_panel_port" <<'REMOTE'
set -Eeuo pipefail
image="$1"
container="$2"
network="$3"
volume="$4"
port="$5"
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

docker pull "$image" >/dev/null
docker inspect "$container" --format '{{range .Config.Env}}{{println .}}{{end}}' | grep -Ev '^(PATH|NODE_VERSION|YARN_VERSION|GOLANG_VERSION|GOTOOLCHAIN|GOPATH)=' > "$env_file"
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
until curl -fsS "http://127.0.0.1:${port}/healthz" >/dev/null; do
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
  run_target local-node
  run_target camelbot-node
  run_target camelbot-panel
else
  run_target "$target"
fi
