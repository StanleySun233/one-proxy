#!/usr/bin/env bash
set -euo pipefail

mode="${1:-check}"
tag="${2:-}"
ssh_host="${CAMELBOT_SSH_HOST:-camelbot}"
image_repo="${ONEPROXY_NODE_IMAGE_REPO:-ghcr.io/stanleysun233/oneproxy-node}"
container="${ONEPROXY_NODE_CONTAINER:-one-proxy-node}"
mysql_container="${ONEPROXY_MYSQL_CONTAINER:-one-proxy-mysql8}"
db_name="${ONEPROXY_DB_NAME:-one_proxy}"
expected_nodes="${ONEPROXY_EXPECTED_NODE_NAMES:-hk-public-node,sg-astar-58}"
reverse_node="${ONEPROXY_REVERSE_NODE_NAME:-sg-astar-58}"
direct_node="${ONEPROXY_DIRECT_NODE_NAME:-hk-public-node}"

case "$mode" in
  check)
    image="-"
    ;;
  deploy)
    if [ -z "$tag" ]; then
      echo "missing release tag" >&2
      exit 2
    fi
    if [ "$tag" = "latest" ]; then
      echo "deploy requires an immutable release tag" >&2
      exit 2
    fi
    image="${image_repo}:${tag}"
    ;;
  *)
    echo "usage: $0 check | deploy <tag>" >&2
    exit 2
    ;;
esac

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

ssh -T "${ssh_opts[@]}" "$ssh_host" 'bash -s' -- "$mode" "$image" "$container" "$mysql_container" "$db_name" "$expected_nodes" "$reverse_node" "$direct_node" <<'REMOTE'
set -Eeuo pipefail

mode="$1"
image="$2"
container="$3"
mysql_container="$4"
db_name="$5"
expected_nodes="$6"
reverse_node="$7"
direct_node="$8"

if [ "$image" = "-" ]; then
  image=""
fi

sql_quote() {
  printf "%s" "$1" | sed "s/'/''/g"
}

mysql_query() {
  docker exec -e MYSQL_QUERY="$1" -e MYSQL_DATABASE="$db_name" "$mysql_container" sh -lc 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" "$MYSQL_DATABASE" -N -B -e "$MYSQL_QUERY"' 2>/dev/null
}

container_running() {
  [ "$(docker inspect "$container" --format '{{.State.Running}}' 2>/dev/null || true)" = "true" ]
}

panel_healthy() {
  curl -fsS http://127.0.0.1:2886/healthz >/dev/null
}

node_auth_ready() {
  code="$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:2988/ || true)"
  [ "$code" = "401" ] || [ "$code" = "407" ]
}

nodes_healthy() {
  old_ifs="$IFS"
  IFS=","
  set -- $expected_nodes
  IFS="$old_ifs"
  for name in "$@"; do
    quoted="$(sql_quote "$name")"
    status="$(mysql_query "SELECT status FROM nodes WHERE name='${quoted}' LIMIT 1;")"
    [ "$status" = "healthy" ] || return 1
  done
}

reverse_connected() {
  [ -n "$reverse_node" ] || return 0
  quoted="$(sql_quote "$reverse_node")"
  count="$(mysql_query "SELECT COUNT(*) FROM node_transports t JOIN nodes n ON n.id=t.node_id WHERE n.name='${quoted}' AND t.transport_type='reverse_ws_parent' AND t.status='connected';")"
  [ "${count:-0}" -ge 1 ]
}

direct_available() {
  [ -n "$direct_node" ] || return 0
  quoted="$(sql_quote "$direct_node")"
  count="$(mysql_query "SELECT COUNT(*) FROM node_transports t JOIN nodes n ON n.id=t.node_id WHERE n.name='${quoted}' AND t.transport_type='direct_udp_candidate' AND t.status='available';")"
  [ "${count:-0}" -ge 1 ]
}

wait_for() {
  label="$1"
  shift
  tries=0
  until "$@"; do
    tries=$((tries + 1))
    if [ "$tries" -ge 45 ]; then
      echo "${label} failed" >&2
      return 1
    fi
    sleep 2
  done
}

assert_runtime() {
  wait_for "container_running" container_running
  wait_for "panel_healthy" panel_healthy
  wait_for "node_auth_ready" node_auth_ready
  wait_for "nodes_healthy" nodes_healthy
  wait_for "reverse_connected" reverse_connected
  wait_for "direct_available" direct_available
}

print_runtime() {
  runtime="$(docker inspect "$container" --format '{{.Config.Image}} {{.State.Status}} {{.State.StartedAt}}')"
  nodes="$(mysql_query "SELECT CONCAT(name, ':', status) FROM nodes ORDER BY id;" | tr '\n' ',' | sed 's/,$//')"
  transports="$(mysql_query "SELECT CONCAT(n.name, ':', t.transport_type, ':', t.status) FROM node_transports t JOIN nodes n ON n.id=t.node_id ORDER BY n.id,t.id;" | tr '\n' ',' | sed 's/,$//')"
  echo "runtime=${runtime}"
  echo "nodes=${nodes}"
  echo "transports=${transports}"
}

deploy() {
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
  docker inspect "$container" --format '{{range .Config.Env}}{{println .}}{{end}}' | grep -Ev '^(PATH|GOLANG_VERSION|GOTOOLCHAIN|GOPATH|ZONEINFO)=' > "$env_file"
  docker rename "$container" "$backup"
  renamed=1
  docker stop "$backup" >/dev/null
  docker run -d \
    --name "$container" \
    --restart unless-stopped \
    --network one-proxy-net \
    --env-file "$env_file" \
    -v one-proxy-node-runtime:/app/runtime \
    -p 2988:2988 \
    -p 2989:2989 \
    -p 2990:2990 \
    -p 2991:2991/udp \
    -p 2992:2992/udp \
    "$image" >/dev/null

  assert_runtime
  completed=1
  rm -f "$env_file"
  trap - ERR HUP INT TERM
  echo "backup=${backup}"
  print_runtime
}

case "$mode" in
  check)
    assert_runtime
    print_runtime
    ;;
  deploy)
    deploy
    ;;
esac
REMOTE
