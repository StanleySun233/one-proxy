# one-proxy

## Structure

- `apps/chrome-extension`: user-side Chrome extension
- `apps/one-panel-api`: Go control-plane backend
- `apps/one-proxy-panel`: Next.js admin console
- `apps/one-proxy-node`: Go proxy node
- `docker/`: container build and startup assets

## Current Direction

- Chrome extension stays plain JavaScript for now
- backend moves to Go
- admin web moves to Next.js
- one-panel-api uses MySQL 8.0, while one-proxy-node keeps local SQLite state

## Docker Run

### Control Plane

1. Start MySQL 8.0:

```bash
docker run -d --name one-proxy-mysql8 \
  -e MYSQL_ROOT_PASSWORD=password \
  -e MYSQL_DATABASE=one_proxy \
  -p 3306:3306 \
  mysql:8.0
```

2. Prepare env file:

```bash
cp docker/one-proxy-panel.env.example .env.control-plane
```

Default timezone is `Asia/Shanghai`. Override `TZ` in `.env.control-plane` if needed.

3. Build and run the single control-plane container:

```bash
docker build -f docker/one-proxy-panel.Dockerfile -t one-proxy-panel .
docker run --rm --name one-proxy-panel \
  --add-host host.docker.internal:host-gateway \
  --env-file .env.control-plane \
  -p 2886:2886 \
  one-proxy-panel
```

Open `http://127.0.0.1:2886`. The frontend is the only exposed port. `/api/v1/*` is proxied inside the same container to the backend on `127.0.0.1:2887`.

### Proxy Node

1. Prepare env file:

```bash
cp docker/one-proxy-node.env.example .env.proxy-node
```

Default timezone is `Asia/Shanghai`. Override `TZ` in `.env.proxy-node` if needed.

2. Build and run:

```bash
docker build -f docker/one-proxy-node.Dockerfile -t one-proxy-node .
docker run --rm --name one-proxy-node \
  --env-file .env.proxy-node \
  -p 2888:2888 \
  -p 2889:2889 \
  one-proxy-node
```

The node keeps its local runtime state in SQLite/JSON files inside the container. Mount `/app/runtime` if you want persistence.
Set `NODE_REVERSE_TARGET_URL=http://172.20.116.91:2333` when a node should act as a reverse HTTP/WebSocket entry for a JupyterLab service.

## GHCR

Pushes to `main` trigger the split image workflows and publish:

- `ghcr.io/stanleysun233/one-proxy-panel:latest`
- `ghcr.io/stanleysun233/one-proxy-node:latest`
