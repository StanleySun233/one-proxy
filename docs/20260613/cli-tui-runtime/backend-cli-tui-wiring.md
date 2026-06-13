# Backend Progress: cli-tui-wiring

**Engineer:** cli-tui-wiring
**Scope:** Wire default TUI runtime into CLI commands while preserving stdio fallback behavior.

## Tasks

- [x] Route `onep ssh` through the TUI runtime by default in `apps/cli/src/ssh.ts`
  - Commit: 8e7986b
- [x] Route `onep shell` through the TUI runtime by default in `apps/cli/src/shell.ts`
  - Commit: f7a7a10
- [x] Route `onep run` through the TUI runtime by default in `apps/cli/src/session-env.ts`
  - Commit: 51a77a4
- [x] Update command usage text in `apps/cli/src/main.ts` for default TUI commands
  - Commit: 796bdea
- [x] Add required TUI dependency metadata in `apps/cli/package.json`
  - Commit: af4384c

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
