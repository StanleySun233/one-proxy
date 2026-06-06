# Test Progress: cli

**Engineer:** test-cli
**Scope:** Verify CLI command behavior, daemon behavior, compile checks, and product-facing command contracts.

## Tasks

- [x] add CLI unit tests for storage, route matching, env output, command parsing, and JSON schemas in apps/cli/test/cli.test.mjs
  - Commit: 4785afb
  - Result: PASS after fix c6ab076. CLI tests pass for storage normalization, route matching, env output, command parsing, and status JSON.
- [x] add daemon tests for HTTP CONNECT, random consecutive port selection, lifecycle metadata, and doctor failure reporting in apps/cli/test/daemon.test.mjs
  - Commit: 2f50f3a
  - Result: PASS after fix c6ab076. Daemon tests pass for HTTP CONNECT, random consecutive port selection, lifecycle metadata, and doctor failure reporting.
- [x] run TypeScript compile checks and CLI tests using the Node.js environment recorded in ./.codex/ENVS.md
  - Commit: c6ab076
  - Result: PASS. `node --test apps/cli/test/*.mjs` passed 12/12. Per-file `node --check` passed for CLI source files.

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
| 2026-06-06 | Source-level CLI execution failed because `.ts` files imported sibling modules using `.js` specifiers, but no built `dist` files or source resolver were present for `node apps/cli/src/main.ts`. | Resolved by c6ab076 |
| 2026-06-06 | `apps/cli/src/daemon/lifecycle.ts` could not be imported by Node v26 TypeScript stripping because default parameter initializers used `await`. | Resolved by c6ab076 |
