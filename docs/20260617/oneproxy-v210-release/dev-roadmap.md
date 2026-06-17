# Dev Roadmap: OneProxy v2.1.0 Release

**Date:** 20260617
**Status:** standing post-setup replacement complete; final tag pending
**Product document:** ./product-requirements.md

## Summary

OneProxy v2.1.0 has been brought to a latest-contract-only runtime with safer node authorization, unified access-path routing, hardened token handling, clearer operational UX, and a final-schema-only panel baseline. The audited runtime commit is `3cf4562`; full audit evidence is recorded in `audit-report.md`.

The manual panel setup flow has been completed by the user, and the standing panel, remote edge node, and local relay node now run the immutable `v2.1.0-rc.3cf4562` images. The `v2.1.0` tag remains gated by final product verification.

## Team

| Role | Agent Name | Progress File |
|------|------------|---------------|
| Backend | backend-contract | [->](./backend-backend-contract.md) |
| Backend | backend-node-security | [->](./backend-backend-node-security.md) |
| Backend | backend-panel-security | [->](./backend-backend-panel-security.md) |
| Backend | backend-client-tools | [->](./backend-backend-client-tools.md) |
| Frontend | frontend-panel-ops | [->](./frontend-frontend-panel-ops.md) |
| Frontend | frontend-extension-routing | [->](./frontend-frontend-extension-routing.md) |
| Testing | test-release | [->](./test-release.md) |
| Product | product-manager | [->](./product-manager.md) |

## Tasks

Each task is scoped to the named file or external gate. No old-version compatibility is required for this release.

### API Contract

- [ ] backend-contract: write `docs/20260617/oneproxy-v210-release/api-contract.md` covering latest access-path bootstrap, route evaluation, proxy-token validation, node auth, client sync, release image, and test evidence contracts

### Backend Tasks

- [ ] backend-node-security: make proxy token authorization fail closed in `apps/node/api/internal/proxy/auth.go` (depends: api-contract.md)
- [ ] backend-node-security: remove default join password and require explicit bootstrap secret in `apps/node/api/internal/agentconfig/config.go` (depends: api-contract.md)
- [ ] backend-node-security: harden node attach and password rotation behavior in `apps/node/api/internal/bootstrap/handler.go` (depends: agentconfig/config.go)
- [ ] backend-node-security: prevent unbound node proxy exposure in `apps/node/api/cmd/one-proxy-node/main.go` (depends: auth.go, bootstrap/handler.go)
- [ ] backend-node-security: require closed authorization semantics in `apps/node/api/internal/tcpaccess/server.go` (depends: auth.go)
- [ ] backend-node-security: require closed authorization semantics and packet limits in `apps/node/api/internal/udpaccess/server.go` (depends: auth.go)
- [ ] backend-node-security: authenticate public next-hop CONNECT forwarding in `apps/node/api/internal/proxy/connect_tunnel.go` (depends: api-contract.md)
- [ ] backend-node-security: stream HTTP forwarding, add timeouts, and limit unsafe retries in `apps/node/api/internal/proxy/forward_http.go` (depends: api-contract.md)
- [ ] backend-panel-security: hash panel access and refresh session tokens in `apps/panel/api/internal/store/mysql_account.go` (depends: api-contract.md)
- [ ] backend-panel-security: sanitize service error responses in `apps/panel/api/internal/httpapi/response.go` (depends: api-contract.md)
- [ ] backend-panel-security: remove raw proxy-token validation fallback and enforce latest scope fields in `apps/panel/api/internal/httpapi/proxy_token.go` (depends: api-contract.md)
- [ ] backend-panel-security: harden setup DB test/init behavior in `apps/panel/api/internal/setup/handler.go` (depends: api-contract.md)
- [ ] backend-panel-security: enforce listener port and latest access-path validation in `apps/panel/api/internal/features/proxy/service/access_path.go` (depends: api-contract.md)
- [ ] backend-panel-security: make access-path update semantics latest-contract-only in `apps/panel/api/internal/store/proxy_repository.go` (depends: access_path.go)
- [ ] backend-client-tools: remove legacy bootstrap token wrappers and legacy route fallbacks in `apps/cli/src/control-plane.ts` (depends: api-contract.md)
- [ ] backend-client-tools: align TypeScript CLI route matching to the latest contract in `apps/cli/src/daemon/router.ts` (depends: control-plane.ts)
- [ ] backend-client-tools: require authorized loopback daemon IPC in `apps/cli/src/daemon/lifecycle.ts` (depends: api-contract.md)
- [ ] backend-client-tools: validate CONNECT proxy responses and stream HTTP proxy bodies in `apps/cli/src/daemon/http-proxy.ts` (depends: router.ts)
- [ ] backend-client-tools: remove password CLI flag and align default ports in `apps/extension/cli/cmd/oneproxy/main.go` (depends: api-contract.md)
- [ ] backend-client-tools: verify direct QUIC node identity in `apps/extension/cli/internal/proxycommand/direct.go` (depends: api-contract.md)
- [ ] backend-client-tools: align VS Code SSH generation to access-path identity in `apps/extension/vscode/src/extension.ts` (depends: api-contract.md)
- [ ] backend-client-tools: send daemon IPC secret from route and probe commands in `apps/cli/src/commands.ts` (depends: lifecycle.ts)
- [ ] backend-client-tools: align daemon metadata storage type with IPC secret in `apps/cli/src/storage.ts` (depends: lifecycle.ts)
- [ ] backend-client-tools: fix CONNECT tunnel socket typing in `apps/cli/src/daemon/http-proxy.ts` (depends: http-proxy.ts)
- [ ] backend-client-tools: regenerate tracked CLI dist artifacts in `apps/cli/dist/**` for v2.1.0 source changes (depends: all CLI source tasks)
- [ ] backend-node-security: update node proxy tests for v2.1.0 fail-closed behavior in `apps/node/api/internal/proxy/server_test.go` and `apps/node/api/internal/proxy/reverse_test.go` (depends: auth.go)
- [ ] backend-client-tools: update `onep init` to consume only latest access-path bootstrap state in `apps/cli/src/init.ts` (depends: control-plane.ts)
- [ ] backend-client-tools: replace visible `group list|use` CLI surface with `access-path list|use` in `apps/cli/src/main.ts` and `apps/cli/src/control-plane.ts` (depends: init.ts)
- [ ] backend-client-tools: regenerate tracked CLI dist artifacts after init and command-surface changes in `apps/cli/dist/**` (depends: init.ts, main.ts)
- [ ] backend-node-security: verify direct QUIC identity in node-to-node stream clients in `apps/node/api/internal/direct/quic_stream.go` (depends: api-contract.md)
- [ ] backend-node-security: pass direct QUIC peer identity through node candidates and link plans in `apps/node/api/internal/domain/direct.go`, `apps/node/api/internal/direct/**`, and `apps/panel/api/internal/{domain,store}/direct_transport.go` (depends: quic_stream.go)

### Frontend Tasks

- [ ] frontend-panel-ops: move production panel web auth away from localStorage token persistence in `apps/panel/web/components/auth-provider.tsx` (depends: api-contract.md)
- [ ] frontend-panel-ops: align panel API client auth behavior with the latest session contract in `apps/panel/web/lib/api/client.ts` (depends: auth-provider.tsx)
- [ ] frontend-panel-ops: remove duplicate audit proxy fallback in `apps/panel/web/app/api/audit/[...path]/route.ts` (depends: api-contract.md)
- [ ] frontend-panel-ops: add production security headers in `apps/panel/web/next.config.mjs` (depends: api-contract.md)
- [ ] frontend-panel-ops: make access-path editor reject unusable listeners and show route health in `apps/panel/web/app/[locale]/(console)/proxy/studio/_components/access-path-panel.tsx` (depends: access_path.go)
- [ ] frontend-panel-ops: revise operational panel visual tokens in `apps/panel/web/app/styles/tokens.css` (depends: access-path-panel.tsx)
- [ ] frontend-panel-ops: reduce nested card chrome and improve console density in `apps/panel/web/app/styles/layout.css` (depends: tokens.css)
- [ ] frontend-extension-routing: replace legacy group state with latest route/access-path state in `apps/extension/chrome/tools/background-source/state.js` (depends: api-contract.md)
- [ ] frontend-extension-routing: sync latest bootstrap contract in `apps/extension/chrome/tools/background-source/api.js` (depends: state.js)
- [ ] frontend-extension-routing: compile latest route rules into PAC behavior in `apps/extension/chrome/tools/background-source/pac.js` (depends: routing.js)
- [ ] frontend-extension-routing: make route preview share the same evaluator assumptions as PAC in `apps/extension/chrome/tools/background-source/routing.js` (depends: api.js)
- [ ] frontend-extension-routing: restrict runtime message responses and session exposure in `apps/extension/chrome/tools/background-source/messages.js` (depends: state.js)
- [ ] frontend-extension-routing: minimize Chrome extension permissions in `apps/extension/chrome/manifest.json` (depends: messages.js)
- [ ] frontend-extension-routing: update popup route/group display for latest access-path state in `apps/extension/chrome/popup/runtime.js` (depends: state.js)
- [ ] frontend-extension-routing: update options route/group display for latest access-path state in `apps/extension/chrome/options/runtime.js` (depends: state.js)
- [ ] frontend-extension-routing: authorize multiple access-path proxy challenges in `apps/extension/chrome/tools/background-source/proxy-auth.js` (depends: pac.js)
- [ ] frontend-extension-routing: sync Chrome page-source popup/options with latest access-path runtime in `apps/extension/chrome/tools/page-source/popup/index.js` and `apps/extension/chrome/tools/page-source/options/index.js` (depends: popup/runtime.js, options/runtime.js)
- [ ] frontend-extension-routing: regenerate tracked Chrome extension bundles in `apps/extension/chrome/background/one-proxy-worker.js`, `apps/extension/chrome/popup/runtime.js`, and `apps/extension/chrome/options/runtime.js` (depends: page-source and background-source tasks)
- [ ] frontend-panel-ops: remove default join-password guidance from node console messages in `apps/panel/web/messages/en/nodesConsole.json` and `apps/panel/web/messages/zh/nodesConsole.json` (depends: agentconfig/config.go)
- [ ] frontend-extension-routing: replace status bubble and monitor legacy group probes with latest access-path route topology in `apps/extension/chrome/tools/background-source/status-bubble.js` and `apps/extension/chrome/tools/background-source/monitor.js` (depends: state.js, routing.js)
- [ ] frontend-extension-routing: update shared page-source contracts and extension smoke fixtures to latest access-path bootstrap state in `apps/extension/chrome/tools/page-source/shared/contracts.js`, `apps/extension/chrome/tools/domain_suffix_test.mjs`, and `apps/extension/chrome/tools/service_worker_smoke.mjs` (depends: api.js)
- [ ] frontend-extension-routing: regenerate tracked Chrome extension bundles after monitor/status/test fixture updates (depends: status-bubble.js, monitor.js)

### Release, Test, and Deployment Tasks

- [ ] test-release: add local v2.1.0 Docker scenario runner in `scripts/test-v210-docker-scenario.sh` (depends: all backend and frontend tasks)
- [ ] test-release: add camelbot isolated scenario runner in `scripts/test-camelbot-v210-scenario.sh` (depends: scripts/test-v210-docker-scenario.sh)
- [ ] test-release: extend GitHub Actions image workflows for immutable pre-tag test images in `.github/workflows/one-proxy-node-image.yml` (depends: all backend tasks)
- [ ] test-release: extend GitHub Actions image workflows for immutable pre-tag test images in `.github/workflows/one-proxy-panel-image.yml` (depends: all frontend and panel backend tasks)
- [ ] test-release: add release deployment script for local node, camelbot node, and camelbot panel in `scripts/deploy-v210-release-images.sh` (depends: GitHub Actions workflow tasks)
- [ ] test-release: document database-backed real-user verification in `docs/20260617/oneproxy-v210-release/release-test-plan.md` (depends: deployment script)
- [ ] test-release: run compile, unit, extension smoke, local Docker scenario, camelbot isolated scenario, replacement deployment, DB queries, and real-user functional tests (depends: all implementation tasks)
- [ ] product-manager: verify delivered implementation against `docs/20260617/oneproxy-v210-release/product-requirements.md` (depends: test-release)
- [ ] test-release: create and push tag `v2.1.0` after all gates pass (depends: product-manager)

## Blocker Log

| Date | Task | Engineer | Blocker Description | Resolution |
|------|------|----------|---------------------|------------|
| 2026-06-17 | Standing replacement | test-release | The old standing database was not a valid final-schema-only target. | Reset completed; panel is running `v2.1.0-rc.65411e7` in setup mode with `configured=false`. |
| 2026-06-17 | Standing node bootstrap | test-release | Remote and local nodes could not be bootstrapped until the panel setup flow wrote final DB configuration. | User completed setup; panel, remote edge, and local relay were replaced with `v2.1.0-rc.3cf4562` and verified healthy. |
