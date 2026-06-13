# Backend Progress: cli-tui-wiring

**Engineer:** cli-tui-wiring
**Scope:** Wire optional TUI runtime into CLI commands while preserving default stdio behavior.

## Tasks

- [x] Route `onep ssh --tui` through the TUI runtime in `apps/cli/src/ssh.ts`
  - Commit: 8e7986b
- [ ] Route `onep shell --tui` through the TUI runtime in `apps/cli/src/shell.ts`
  - Commit:
- [ ] Route `onep run --tui` through the TUI runtime in `apps/cli/src/session-env.ts`
  - Commit:
- [ ] Update command usage text in `apps/cli/src/main.ts` for `--tui` options
  - Commit:
- [ ] Add optional TUI dependency metadata in `apps/cli/package.json`
  - Commit:

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
