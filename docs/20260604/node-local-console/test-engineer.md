# Test Progress: test-engineer

**Engineer:** test-engineer
**Scope:** Compile and interface verification for node-local console and extension path migration.

## Tasks

- [x] compile check and interface tests for node-local API plus extension path checks
  - Result: passed for node-local API plus extension path scope.
- [x] compile check for panel-api route migration and manage-access endpoint
  - Result: passed.

## Commands Run

| Command | Result |
|---------|--------|
| `go test ./...` from `apps/node/api` | pass |
| `go test ./...` from `apps/panel/api` | pass |
| `npm run check` from `apps/node/api/web` | pass |
| `rg -n '/api/v1' apps/panel/api/internal apps/panel/api/openapi.yaml` | pass: no matches |
| `rg -n '/api/v1' apps/node/api apps/extension/chrome/tools apps/extension/chrome docker/one-proxy-node.env.example` | pass: no matches |
| `rg -n 'controlPlaneUrl\|control plane\|login\|NODE_CONTROL_PLANE\|CONTROL' docs/20260604/node-local-console/api-contract.md apps/node/api apps/node/api/web/src apps/node/api/web/scripts docker/one-proxy-node.env.example` | pass: login contract and implementation use node-configured control plane URL; browser login payload only submits credentials |
| `sed -n '1,240p' docker/one-proxy-node.Dockerfile` | pass: web assets are built in `node:26-bookworm AS web-builder` and copied from that stage |
| `git status --short apps/panel/web` | residual risk: panel-web remains out of scope for this phase |

## Results

- Node API compile/tests passed.
- Node web syntax check passed.
- No `/api/v1` remains in the requested node API, extension, or node env example paths.
- Dockerfile builds web assets in a build stage instead of copying workspace `web/dist`.
- Node local login no longer accepts a browser-supplied `controlPlaneUrl`; the handler reads the configured upstream control plane URL and forwards credentials to `/api/auth/login`.
- Panel API route migration and node manage-access compile checks passed.
- Deferred panel-web task remains unchecked and blocked pending user approval.

## Residual Risks

| Date | Risk | Status |
|------|------|--------|
| 2026-06-04 | The workspace has dirty panel-web files. They were treated as out of scope for this panel-api plus node-local API plus extension path test phase and were not inspected. | open |
