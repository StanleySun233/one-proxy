# Backend Progress: backend-client-tools

**Engineer:** backend-client-tools
**Scope:** TypeScript CLI, Go CLI, and VS Code extension connection behavior.

## Tasks

- [ ] Remove legacy bootstrap token wrappers and legacy route fallbacks in `apps/cli/src/control-plane.ts`
  - Commit:
- [ ] Align TypeScript CLI route matching to the latest contract in `apps/cli/src/daemon/router.ts`
  - Commit:
- [ ] Require authorized loopback daemon IPC in `apps/cli/src/daemon/lifecycle.ts`
  - Commit:
- [ ] Validate CONNECT proxy responses and stream HTTP proxy bodies in `apps/cli/src/daemon/http-proxy.ts`
  - Commit:
- [ ] Remove password CLI flag and align default ports in `apps/extension/cli/cmd/oneproxy/main.go`
  - Commit:
- [ ] Verify direct QUIC node identity in `apps/extension/cli/internal/proxycommand/direct.go`
  - Commit:
- [ ] Align VS Code SSH generation to access-path identity in `apps/extension/vscode/src/extension.ts`
  - Commit:

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
