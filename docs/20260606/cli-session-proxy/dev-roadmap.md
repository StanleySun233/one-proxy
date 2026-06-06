# Dev Roadmap: CLI Session Proxy

**Date:** 20260606
**Status:** completed
**Product document:** ./product-requirements.md

## Summary

The project will add a new Node.js `onep` CLI that applies OneProxy rules to selected CLI workflows instead of changing global system proxy settings. It supports login, tenant and group selection, shell-session environment activation, command wrapping, local overrides, route testing, SSH, diagnostics, and a rootless loopback daemon.

## Team

| Role | Agent Name | Progress File |
|------|-----------|---------------|
| Backend | backend-cli-contract | [->](./backend-cli-contract.md) |
| Backend | backend-cli-commands | [->](./backend-cli-commands.md) |
| Backend | backend-daemon-runtime | [->](./backend-daemon-runtime.md) |
| Test | test-cli | [->](./test-cli.md) |
| Product | product-manager | [->](./product-manager.md) |

## Tasks

Each task represents one atomic, file-scoped unit of work. No two engineers may touch the same file.

### API Contract

- [x] backend-cli-contract: write api-contract.md covering CLI command contracts, local storage schemas, daemon IPC contracts, route/test/doctor output schemas, and error formats

### Backend Tasks

- [x] backend-cli-commands: create the Node package manifest in apps/cli/package.json (depends: api-contract.md)
- [x] backend-cli-commands: create TypeScript configuration in apps/cli/tsconfig.json (depends: package.json)
- [x] backend-cli-commands: implement command entrypoint and argument routing in apps/cli/src/main.ts (depends: api-contract.md)
- [x] backend-cli-commands: implement control-plane login, refresh, logout, tenant, group, and sync client in apps/cli/src/control-plane.ts (depends: api-contract.md)
- [x] backend-cli-commands: implement ~/.oneproxy storage and permission handling in apps/cli/src/storage.ts (depends: api-contract.md)
- [x] backend-cli-commands: implement env on/off shell output and onep run command execution in apps/cli/src/session-env.ts (depends: storage.ts)
- [x] backend-cli-commands: implement override, route, status, and JSON output helpers in apps/cli/src/commands.ts (depends: storage.ts, control-plane.ts)
- [x] backend-daemon-runtime: implement daemon lifecycle and loopback port metadata in apps/cli/src/daemon/lifecycle.ts (depends: api-contract.md)
- [x] backend-daemon-runtime: implement HTTP CONNECT local proxy in apps/cli/src/daemon/http-proxy.ts (depends: lifecycle.ts)
- [x] backend-daemon-runtime: implement random consecutive proxy port selection in apps/cli/src/daemon/port-selection.ts (depends: lifecycle.ts)
- [x] backend-daemon-runtime: implement route matching and local override precedence in apps/cli/src/daemon/router.ts (depends: api-contract.md)
- [x] backend-daemon-runtime: implement probe and doctor checks in apps/cli/src/doctor.ts (depends: lifecycle.ts, router.ts)
- [x] backend-daemon-runtime: implement SSH command routing in apps/cli/src/ssh.ts (depends: lifecycle.ts, router.ts)

### Testing

- [x] test-cli: add CLI unit tests for storage, route matching, env output, command parsing, and JSON schemas in apps/cli/test/cli.test.mjs (depends: all backend tasks)
- [x] test-cli: add daemon tests for HTTP CONNECT, random consecutive port selection, lifecycle metadata, and doctor failure reporting in apps/cli/test/daemon.test.mjs (depends: all daemon tasks)
- [x] test-cli: run TypeScript compile checks and CLI tests using the Node.js environment recorded in ./.codex/ENVS.md (depends: all tests)

### Product Verification

- [x] product-manager: verify delivered behavior against product-requirements.md after tests pass

## Blocker Log

| Date | Task | Engineer | Blocker Description | Resolution |
|------|------|----------|---------------------|------------|
