# Dev Roadmap: Connection Resilience

**Date:** 20260620
**Status:** planned
**Product document:** ./product-requirements.md

## Summary

This project turns connection handling into an explicit latest-contract-only runtime across extension, CLI, node, and panel. It combines automatic path setup, connection-state visibility, reconnect behavior, self-healing access-path selection, runtime snapshots, and operator diagnostics into one delivery.

The implementation must not preserve legacy group contracts, old route fallbacks, or compatibility branches. Every task is scoped so each source file is owned by one engineer for this plan.

## Team

| Role | Agent Name | Progress File |
|------|------------|---------------|
| Backend | backend-contract | [->](./backend-backend-contract.md) |
| Backend | backend-panel-runtime | [->](./backend-backend-panel-runtime.md) |
| Backend | backend-node-runtime | [->](./backend-backend-node-runtime.md) |
| Backend | backend-client-runtime | [->](./backend-backend-client-runtime.md) |
| Frontend | frontend-extension-runtime | [->](./frontend-frontend-extension-runtime.md) |
| Frontend | frontend-panel-ops | [->](./frontend-frontend-panel-ops.md) |
| Testing | test-resilience | [->](./test-resilience.md) |
| Product | product-manager | [->](./product-manager.md) |

## Tasks

Each task represents one atomic, file-scoped unit of work. No two engineers may touch the same source file.

### API Contract

- [ ] backend-contract: write `docs/20260620/connection-resilience/api-contract.md` covering connection state, reconnect state, runtime snapshot, diagnostics, automatic access-path selection, and CLI status contracts

### Backend Tasks

- [ ] backend-panel-runtime: define connection and runtime snapshot API models in `apps/panel/api/internal/domain/connection_state.go` (depends: api-contract.md)
- [ ] backend-panel-runtime: persist latest node runtime snapshots and recent snapshot history in `apps/panel/api/internal/store/mysql_runtime_snapshot.go` (depends: connection_state.go)
- [ ] backend-panel-runtime: expose runtime snapshot service methods in `apps/panel/api/internal/service/runtime_snapshot.go` (depends: mysql_runtime_snapshot.go)
- [ ] backend-panel-runtime: serve connection diagnostics endpoints in `apps/panel/api/internal/httpapi/handler_connection_state.go` (depends: runtime_snapshot.go)
- [ ] backend-panel-runtime: wire diagnostics routes in `apps/panel/api/internal/httpapi/router.go` (depends: handler_connection_state.go)
- [ ] backend-panel-runtime: apply stale snapshot maintenance in `apps/panel/api/internal/store/mysql_maintenance.go` (depends: mysql_runtime_snapshot.go)
- [ ] backend-panel-runtime: implement automatic access-path candidate selection in `apps/panel/api/internal/features/proxy/service/access_path_autoselect.go` (depends: api-contract.md)
- [ ] backend-panel-runtime: use automatic access-path selection from route creation in `apps/panel/api/internal/features/proxy/service/route.go` (depends: access_path_autoselect.go)
- [ ] backend-panel-runtime: use automatic relay and public endpoint defaults in `apps/panel/api/internal/features/proxy/service/access_path.go` (depends: access_path_autoselect.go)
- [ ] backend-panel-runtime: add panel runtime snapshot and autoselect tests in `apps/panel/api/internal/service/connection_resilience_test.go` (depends: backend-panel-runtime tasks)
- [ ] backend-node-runtime: define node runtime snapshot collector in `apps/node/api/internal/runtime/snapshot.go` (depends: api-contract.md)
- [ ] backend-node-runtime: include runtime snapshot payloads in heartbeat loop in `apps/node/api/internal/heartbeat/loop.go` (depends: snapshot.go)
- [ ] backend-node-runtime: report upstream reachability and parent relay state in `apps/node/api/internal/controlplane/client.go` (depends: snapshot.go)
- [ ] backend-node-runtime: expose local runtime snapshot in `apps/node/api/internal/localconsole/handler.go` (depends: snapshot.go)
- [ ] backend-node-runtime: include relay probe phase and error codes in `apps/node/api/internal/controlrelay/probe.go` (depends: api-contract.md)
- [ ] backend-node-runtime: add node runtime snapshot tests in `apps/node/api/internal/runtime/snapshot_test.go` (depends: backend-node-runtime tasks)
- [ ] backend-client-runtime: add CLI connection status command in `apps/extension/cli/cmd/oneproxy/main.go` (depends: api-contract.md)
- [ ] backend-client-runtime: implement CLI reconnect loop helpers in `apps/extension/cli/internal/proxycommand/reconnect.go` (depends: api-contract.md)
- [ ] backend-client-runtime: add CLI reconnect tests in `apps/extension/cli/internal/proxycommand/reconnect_test.go` (depends: reconnect.go)
- [ ] backend-client-runtime: show selected access path and connection state in VS Code extension commands in `apps/extension/vscode/src/extension.ts` (depends: api-contract.md)

### Frontend Tasks

- [ ] frontend-extension-runtime: add extension connection-state reducer in `apps/extension/chrome/tools/background-source/connection-state.js` (depends: api-contract.md)
- [ ] frontend-extension-runtime: persist connection state and retry metadata in `apps/extension/chrome/tools/background-source/state.js` (depends: connection-state.js)
- [ ] frontend-extension-runtime: sync connection diagnostics from panel bootstrap and status APIs in `apps/extension/chrome/tools/background-source/api.js` (depends: state.js)
- [ ] frontend-extension-runtime: implement bounded reconnect scheduling in `apps/extension/chrome/tools/background-source/monitor.js` (depends: connection-state.js, api.js)
- [ ] frontend-extension-runtime: detect proxy setting drift and proxy auth failures in `apps/extension/chrome/tools/background-source/proxy-auth.js` (depends: monitor.js)
- [ ] frontend-extension-runtime: expose route, phase, retry, selected access path, and hop diagnostics in `apps/extension/chrome/tools/background-source/status-bubble.js` (depends: connection-state.js)
- [ ] frontend-extension-runtime: render diagnostic details in `apps/extension/chrome/tools/content-source/status-bubble.js` (depends: status-bubble.js)
- [ ] frontend-extension-runtime: style diagnostic details in `apps/extension/chrome/content/status-bubble.css` (depends: content-source/status-bubble.js)
- [ ] frontend-extension-runtime: update extension localization keys in `apps/extension/chrome/_locales/zh_CN/messages.json` and `apps/extension/chrome/_locales/en/messages.json` (depends: status-bubble.js)
- [ ] frontend-extension-runtime: regenerate Chrome extension runtime bundle in `apps/extension/chrome/background/one-proxy-worker.js` (depends: frontend-extension-runtime tasks)
- [ ] frontend-extension-runtime: add extension connection-state and reconnect tests in `apps/extension/chrome/test/connection_state_test.mjs` (depends: frontend-extension-runtime tasks)
- [ ] frontend-panel-ops: show runtime snapshot status in `apps/panel/web/app/[locale]/(console)/nodes/_components/registry-node-table.tsx` (depends: api-contract.md)
- [ ] frontend-panel-ops: add node diagnostics drawer in `apps/panel/web/app/[locale]/(console)/nodes/_components/node-diagnostics-drawer.tsx` (depends: registry-node-table.tsx)
- [ ] frontend-panel-ops: use automatic relay and public endpoint defaults in `apps/panel/web/app/[locale]/(console)/proxy/studio/_components/access-path-panel.tsx` (depends: api-contract.md)
- [ ] frontend-panel-ops: show route access-path selection and validation results in `apps/panel/web/app/[locale]/(console)/proxy/routes/_components/route-rule-form.tsx` (depends: access-path-panel.tsx)
- [ ] frontend-panel-ops: add connection diagnostics API client helpers in `apps/panel/web/lib/api/connection-diagnostics.ts` (depends: api-contract.md)
- [ ] frontend-panel-ops: add diagnostic and reconnect visual tokens in `apps/panel/web/app/styles/status.css` (depends: frontend-panel-ops UI tasks)
- [ ] frontend-panel-ops: update panel node and proxy messages in `apps/panel/web/messages/zh/nodesConsole.json` and `apps/panel/web/messages/en/nodesConsole.json` (depends: frontend-panel-ops UI tasks)

### Testing

- [ ] test-resilience: run Go unit tests for panel and node runtime snapshot changes (depends: backend-panel-runtime tasks, backend-node-runtime tasks)
- [ ] test-resilience: run Go unit tests for CLI reconnect helpers (depends: backend-client-runtime tasks)
- [ ] test-resilience: run Chrome extension static validation and node tests (depends: frontend-extension-runtime tasks)
- [ ] test-resilience: run panel TypeScript and lint checks for diagnostics UI (depends: frontend-panel-ops tasks)
- [ ] test-resilience: run browser smoke for extension status bubble reconnect and diagnostics behavior (depends: frontend-extension-runtime tasks)
- [ ] product-manager: verify implementation against `docs/20260620/connection-resilience/product-requirements.md` (depends: test-resilience)

## Blocker Log

| Date | Task | Engineer | Blocker Description | Resolution |
|------|------|----------|---------------------|------------|
