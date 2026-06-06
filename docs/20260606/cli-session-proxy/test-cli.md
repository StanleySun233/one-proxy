# Test Progress: cli

**Engineer:** test-cli
**Scope:** Verify CLI command behavior, daemon behavior, compile checks, and product-facing command contracts.

## Tasks

- [x] add CLI unit tests for storage, route matching, env output, command parsing, and JSON schemas in apps/cli/test/cli.test.mjs
  - Commit: 4785afb
  - Result: FAIL, 3/6 pass. Storage normalization and route matching pass; env output, command parsing, and status JSON fail because `apps/cli/src/main.ts` imports `.js` files that are absent in the source tree.
- [x] add daemon tests for HTTP CONNECT, random consecutive port selection, lifecycle metadata, and doctor failure reporting in apps/cli/test/daemon.test.mjs
  - Commit: 2f50f3a
  - Result: FAIL, 3/6 pass. Port selection tests pass; HTTP CONNECT and doctor fail on missing `.js` source imports; lifecycle import fails under Node TypeScript stripping on `await` in a default parameter initializer.
- [x] run TypeScript compile checks and CLI tests using the Node.js environment recorded in ./.codex/ENVS.md
  - Commit: N/A, verification only
  - Result: FAIL. `node --test apps/cli/test/*.mjs` failed 6/12 tests. `node --check apps/cli/src/*.ts apps/cli/src/daemon/*.ts` returned 0 with no output; per-file `node --check` also passed.

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
| 2026-06-06 | Source-level CLI execution fails because `.ts` files import sibling modules using `.js` specifiers, but no built `dist` files or source resolver are present for `node apps/cli/src/main.ts`. | Open |
| 2026-06-06 | `apps/cli/src/daemon/lifecycle.ts` cannot be imported by Node v26 TypeScript stripping because `resolveBindings(config = await readConfig())` uses `await` in a parameter initializer. | Open |
