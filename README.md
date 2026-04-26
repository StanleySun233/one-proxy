# one-proxy

## Structure

- `apps/chrome-extension`: user-side Chrome extension
- `apps/control-plane-api`: planned Go backend
- `apps/control-plane-web`: planned Next.js admin console
- `prototypes/proxy-node-demo`: archived Python demo
- `docs/`: numbered design and development documents
- `todolist.md`: progress tracker

## Current Direction

- Chrome extension stays plain JavaScript for now
- backend moves to Go
- admin web moves to Next.js
- control-plane backend uses MySQL 8.0, while proxy-node keeps local SQLite state

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
cp docker/control-plane.env.example .env.control-plane
```

3. Build and run the single control-plane container:

```bash
docker build -f docker/control-plane.Dockerfile -t one-proxy-control-plane .
docker run --rm --name one-proxy-control-plane \
  --add-host host.docker.internal:host-gateway \
  --env-file .env.control-plane \
  -p 2886:2886 \
  one-proxy-control-plane
```

Open `http://127.0.0.1:2886`. The frontend is the only exposed port. `/api/control-plane/*` is proxied inside the same container to the backend on `127.0.0.1:2887`.

### Proxy Node

1. Prepare env file:

```bash
cp docker/proxy-node.env.example .env.proxy-node
```

2. Build and run:

```bash
docker build -f docker/proxy-node.Dockerfile -t one-proxy-proxy-node .
docker run --rm --name one-proxy-proxy-node \
  --env-file .env.proxy-node \
  -p 2888:2888 \
  -p 2889:2889 \
  one-proxy-proxy-node
```

The node keeps its local runtime state in SQLite/JSON files inside the container. Mount `/app/runtime` if you want persistence.

## GHCR

Pushes to `main` trigger `.github/workflows/docker-images.yml`. The workflow builds both images, runs local smoke tests with `docker run`, then publishes:

- `ghcr.io/stanleysun233/one-proxy-control-plane:latest`
- `ghcr.io/stanleysun233/one-proxy-proxy-node:latest`
