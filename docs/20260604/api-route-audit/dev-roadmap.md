# Dev Roadmap: API Route Audit

**Date:** 20260604
**Status:** implemented
**Product document:** docs/20260604/api-route-audit/product-requirements.md

## Summary

Audit all panel, node, panel web, node web, and Chrome extension API routes and normalize route naming. The main rule is that each path segment should be a single word; dashed segments must be removed unless explicitly approved.

## Team

| Role | Agent Name | Progress File |
|------|-----------|---------------|
| Backend | panel-api-routes | [→](./backend-panel-api-routes.md) |
| Backend | node-api-routes | [→](./backend-node-api-routes.md) |
| Frontend | panel-web-routes | [→](./frontend-panel-web-routes.md) |
| Frontend | extension-routes | [→](./frontend-extension-routes.md) |
| Test | route-contract-test | [→](./test-route-contract.md) |

## Tasks

Each task represents one atomic, file-scoped unit of work. No two engineers may touch the same file.

### API Contract
- [x] panel-api-routes: write approved canonical route table and old-to-new migration table

### Backend Tasks
- [x] panel-api-routes: update apps/panel/api/internal/httpapi route registration and resourceID prefixes to approved canonical paths
- [x] panel-api-routes: update apps/panel/api/internal/features/proxy/httpapi route registration and resourceID prefixes to approved canonical paths
- [x] panel-api-routes: update apps/panel/api/internal/controlrelay and apps/panel/api/openapi.yaml to approved canonical paths
- [x] node-api-routes: update apps/node/api controlplane clients, relay probes, tunnel path defaults, and route registrations to approved canonical paths
- [x] node-api-routes: update docker/one-proxy-node.env.example route defaults to approved canonical paths

### Frontend Tasks
- [x] panel-web-routes: update apps/panel/web/lib/api client base and request paths to approved canonical paths
- [x] panel-web-routes: replace apps/panel/web/app/api/v1 proxy route with approved unversioned proxy route
- [x] extension-routes: update apps/extension/chrome/tools/background-source API and probe paths to approved canonical paths
- [x] extension-routes: update apps/extension/chrome/background/one-proxy-worker.js generated runtime paths to approved canonical paths
- [x] node-web-routes: no node-local API path rename required

### Testing
- [x] route-contract-test: run route inventory checks ensuring no dashed API segments and no `/api/v1` references remain in source-owned API code
- [x] route-contract-test: run Go tests for apps/panel/api and apps/node/api
- [x] route-contract-test: run syntax checks allowed by project rules for node web and extension code
- [ ] route-contract-test: panel web type check is blocked by stale `.next/types` cache

## Approval Checklist

Approved:

- [x] Keep `/api/proxy` as the proxy chain root.
- [x] Move extension bootstrap to `/api/proxy/extension/bootstrap`.
- [x] Move node transport listing to `/api/nodes/transports`.
- [x] Use resource-first action paths, e.g. `/api/nodes/{nodeId}/approve`.
- [x] Do not allow dashed route segments when a slash-separated path can replace them.

## Blocker Log

| Date | Task | Engineer | Blocker Description | Resolution |
|------|------|----------|---------------------|------------|
