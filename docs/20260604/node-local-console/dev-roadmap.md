# Dev Roadmap: Node Local Console

**Date:** 20260604
**Status:** in-progress
**Product document:** docs/20260604/node-local-console/product-requirements.md

## Summary

Build a first-phase node-local console focused on read-only audit, status, health, policy snapshot, and diagnostics for the current node. Node, extension, and panel-api changes are in scope; panel-web changes remain deferred until manual approval.

## Team

| Role | Agent Name | Progress File |
|------|-----------|---------------|
| Backend | node-api | [→](./backend-node-api.md) |
| Backend | panel-api | [→](./backend-panel-api.md) |
| Frontend | node-web | [→](./frontend-node-web.md) |
| Frontend | extension-api | [→](./frontend-extension-api.md) |
| Test | test-engineer | [→](./test-engineer.md) |

## Tasks

Each task represents one atomic, file-scoped unit of work. No two engineers may touch the same file.

### API Contract
- [x] node-api: write api-contract.md covering node-local endpoints and extension path migration

### Backend Tasks
- [x] node-api: reorganize node Go module paths under apps/one-proxy-node/api while preserving existing package imports (depends: api-contract.md)
- [x] node-api: implement node-local route split and static web serving in apps/one-proxy-node/api/cmd/one-proxy-node/main.go (depends: api-contract.md)
- [x] node-api: implement read-only local console handlers under apps/one-proxy-node/api/internal/localconsole (depends: api-contract.md)
- [x] node-api: update docker/one-proxy-node.Dockerfile, docker/one-proxy-node-base.Dockerfile, and docker/one-proxy-node.env.example for api plus web layout in one container (depends: frontend-node-web tasks)
- [x] panel-api: modify apps/one-panel-api routes for `/api/*` and node manage-access validation (depends: user manual approval)

### Frontend Tasks
- [x] node-web: create node console web app under apps/one-proxy-node/web with login, overview, health, audit, policy, and diagnostics views (depends: api-contract.md)
- [x] node-web: add node console build metadata and static asset output expected by the node Dockerfile (depends: api-contract.md)
- [x] extension-api: update apps/chrome-extension-tools/background-source/api.js to use new panel API paths without old-path compatibility (depends: api-contract.md)
- [x] extension-api: update apps/chrome-extension/background/one-proxy-worker.js generated runtime API paths consistently with source changes (depends: api-contract.md)
- [x] extension-api: update node probe API paths in apps/chrome-extension-tools/background-source/status-bubble.js, apps/chrome-extension-tools/background-source/monitor.js, and generated worker output (depends: api-contract.md)

### Deferred Tasks
- [ ] pending-approval: modify apps/one-proxy-panel API client and proxy routes for `/api/*` (blocked: user approval)

### Testing
- [x] test-engineer: compile check and interface tests for node-local API plus extension path checks (depends: all node and extension tasks)
- [x] test-engineer: compile check for panel-api route migration and manage-access endpoint (depends: panel-api task)

## Blocker Log

| Date | Task | Engineer | Blocker Description | Resolution |
|------|------|----------|---------------------|------------|
