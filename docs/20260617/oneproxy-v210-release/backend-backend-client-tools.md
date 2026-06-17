# Backend Progress: backend-client-tools

**Engineer:** backend-client-tools
**Scope:** TypeScript CLI, Go CLI, and VS Code extension connection behavior.

## Tasks

- [x] Remove legacy bootstrap token wrappers and legacy route fallbacks in `apps/cli/src/control-plane.ts`
  - Commit: 77277af
- [x] Align TypeScript CLI route matching to the latest contract in `apps/cli/src/daemon/router.ts`
  - Commit: 2c57a0a
- [x] Require authorized loopback daemon IPC in `apps/cli/src/daemon/lifecycle.ts`
  - Commit: 4b38175
- [x] Validate CONNECT proxy responses and stream HTTP proxy bodies in `apps/cli/src/daemon/http-proxy.ts`
  - Commit: e24b601
- [x] Remove password CLI flag and align default ports in `apps/extension/cli/cmd/oneproxy/main.go`
  - Commit: 11a14c1
- [x] Verify direct QUIC node identity in `apps/extension/cli/internal/proxycommand/direct.go`
  - Commit: 18fe33f
- [x] Align VS Code SSH generation to access-path identity in `apps/extension/vscode/src/extension.ts`
  - Commit: 8a1c166
- [x] Send daemon IPC secret from route and probe commands in `apps/cli/src/commands.ts`
  - Commit: 6d8e9a0
- [x] Persist daemon IPC secret in CLI metadata storage type in `apps/cli/src/storage.ts`
  - Commit: d20d411
- [x] Type CONNECT tunnel sockets in `apps/cli/src/daemon/http-proxy.ts`
  - Commit: 721b9bc
- [x] Regenerate tracked CLI dist artifacts in `apps/cli/dist/**` for v2.1.0 source changes
  - Commit: 9845752
- [x] Update `onep init` to consume only latest access-path bootstrap state in `apps/cli/src/init.ts`
  - Commit: b3ac4cf
- [x] Replace visible `group list|use` CLI surface with `access-path list|use` in `apps/cli/src/main.ts` and `apps/cli/src/control-plane.ts`
  - Commit: b3ac4cf
- [x] Regenerate tracked CLI dist artifacts after init and command-surface changes in `apps/cli/dist/**`
  - Commit: b3ac4cf
- [x] Enforce latest bootstrap state in `onep init` and reject missing latest access paths
  - Commit: 17b220c

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
| 2026-06-17 | `apps/cli/src/commands.ts` owns route/test IPC calls and is not assigned to this engineer; those calls still need to send `X-One-Proxy-Daemon-Secret` after lifecycle now rejects missing daemon secrets. | Resolved in 6d8e9a0 |
