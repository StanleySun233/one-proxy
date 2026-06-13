# Dev Roadmap: CLI TUI Runtime

**Date:** 20260613
**Status:** completed
**Product document:** ./product-requirements.md

## Summary

Add a default lightweight TUI runtime for interactive OneProxy CLI sessions. The footer reserves one to three terminal rows below the child PTY and displays unlabeled account, tenant, ping, right-aligned traffic totals, and compact route path.

The current implementation attempts the TUI footer by default for `onep ssh`, `onep shell`, and `onep run`. The existing stdio path remains the fallback for unsupported terminal environments.

## Team

| Role | Agent Name | Progress File |
|------|------------|---------------|
| Backend | cli-tui-core | [->](./backend-cli-tui-runtime.md) |
| Backend | cli-tui-wiring | [->](./backend-cli-tui-wiring.md) |
| Backend | cli-tui-tests | [->](./backend-cli-tui-tests.md) |
| Test | test-cli-tui | [->](./test-cli-tui.md) |
| Product | product-manager | [->](./product-manager.md) |

## Tasks

### API Contract

- [x] cli-tui-core: finalize default TUI command behavior, footer shape, color semantics, and fallback semantics in `docs/20260613/cli-tui-runtime/api-contract.md`

### Backend Tasks

- [x] cli-tui-core: add TUI capability detection in `apps/cli/src/tui/capability.ts` (depends: api-contract.md)
- [x] cli-tui-core: add footer row planning, ANSI macaron styling, visible-width truncation, right-aligned totals, and path formatting in `apps/cli/src/tui/footer.ts` (depends: api-contract.md)
- [x] cli-tui-core: add PTY adapter boundary in `apps/cli/src/tui/pty.ts` (depends: api-contract.md)
- [x] cli-tui-core: add TUI runtime orchestration in `apps/cli/src/tui/runtime.ts` (depends: capability.ts, footer.ts, pty.ts)
- [x] cli-tui-core: add TUI status snapshot collection in `apps/cli/src/tui/status.ts` (depends: api-contract.md)
- [x] cli-tui-wiring: route `onep ssh` through the TUI runtime by default in `apps/cli/src/ssh.ts` while preserving fallback behavior (depends: runtime.ts, status.ts)
- [x] cli-tui-wiring: route `onep shell` through the TUI runtime by default in `apps/cli/src/shell.ts` while preserving fallback behavior (depends: runtime.ts, status.ts)
- [x] cli-tui-wiring: route `onep run` through the TUI runtime by default in `apps/cli/src/session-env.ts` while preserving fallback behavior (depends: runtime.ts, status.ts)
- [x] cli-tui-wiring: update command usage text in `apps/cli/src/main.ts` for default TUI commands (depends: ssh.ts, shell.ts, session-env.ts)
- [x] cli-tui-wiring: add required TUI dependency metadata in `apps/cli/package.json` without making CLI startup depend on native PTY loading (depends: pty.ts)
- [x] cli-tui-tests: add focused TUI unit tests in `apps/cli/test/tui.test.mjs` for capability detection, footer layout, color thresholds, visible-width alignment, path formatting, fake PTY resize, and child exit propagation (depends: core tasks)
- [x] cli-tui-tests: add command parsing and fallback coverage in `apps/cli/test/cli.test.mjs` for default TUI commands and compatible `--tui` stripping (depends: wiring tasks)

### Testing

- [x] test-cli-tui: run CLI tests for `apps/cli`
- [x] test-cli-tui: run proxy package tests for updated proxy error page path

### Product Verification

- [x] product-manager: verify implementation against `docs/20260613/cli-tui-runtime/product-requirements.md`

## Implementation Notes

- TUI is default-enabled for interactive `ssh`, `shell`, and `run` commands.
- Do not use Ink or blessed for the first implementation.
- Do not install dependencies or run package manager install commands without user approval.
- Do not touch unrelated dirty files in `apps/node/api/internal/proxy` or `apps/extension/chrome`.

## Blocker Log

| Date | Task | Engineer | Blocker Description | Resolution |
|------|------|----------|---------------------|------------|
