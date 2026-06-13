# Backend Progress: cli-tui-runtime

**Engineer:** cli-tui-core
**Scope:** Implement optional TUI runtime core modules for interactive CLI commands.

## Current Analysis

The existing CLI starts interactive children with inherited stdio:

- `apps/cli/src/ssh.ts` uses `spawn(executable, args, { stdio: 'inherit' })`.
- `apps/cli/src/shell.ts` starts the child shell with inherited stdio.
- `apps/cli/src/session-env.ts` starts `onep run` children with inherited stdio on non-Windows.

The current CLI Session Proxy contract also says `onep run` streams child stdout and stderr without changing content and forwards stdin to the child. A footer TUI changes terminal ownership, so it must be optional first and must have a clean fallback to the existing path.

## Proposed Architecture

```text
onep ssh --tui
  -> build existing SSH command plan
  -> build TUI status snapshot from config, tokens, daemon metadata, and route
  -> start child command inside PTY sized to terminal rows minus footer rows
  -> render footer with ANSI cursor control
  -> forward stdin to PTY except supported shortcuts
  -> resize PTY and redraw footer on SIGWINCH
  -> exit with child exit code
```

## Module Plan

```text
apps/cli/src/tui/capability.ts
apps/cli/src/tui/footer.ts
apps/cli/src/tui/pty.ts
apps/cli/src/tui/runtime.ts
apps/cli/src/tui/status.ts
```

`capability.ts` decides whether TUI can run.

`footer.ts` maps status snapshots and terminal size into one to three display lines.

`pty.ts` isolates dynamic `node-pty` loading and exposes a small adapter so tests can use a fake PTY.

`runtime.ts` owns input routing, child lifecycle, resize handling, and footer redraw.

`status.ts` reads current account, tenant, daemon/session totals, route path, and command lifecycle state.

## Footer Configuration Recommendation

Use these fields first:

- Account
- Tenant
- Ping
- Total upload
- Total download
- Path

Do not display low-signal fields such as PID, local ports, policy revision, or token expiry in the default footer. Those belong in `onep status` or diagnostics.

Footer line layout:

```text
<account>  <tenant>  <latency>
                              Total ↑ <upload> | ↓ <download>
<node-a>-<node-b>-<node-c>
```

Line 1 must not render labels. Account, tenant, and ping are separated by three high-contrast macaron color treatments instead. Line 2 is right-aligned and is the only footer line with a textual hint. Line 3 should reuse the Chrome extension status bubble path payload shape, especially `path.nodes[]`, and render it as compact text without a `Path:` label.

Footer colors:

- Account uses macaron mint.
- Tenant uses macaron lavender.
- Ping uses gray, mint green, butter yellow, or coral red by latency threshold.
- Upload and download markers use separate blue/cyan accents.
- Path nodes use macaron segment colors matching node role and route order.
- Width calculations must strip ANSI codes before truncation and right alignment.

## Phasing

1. Implement shared TUI modules with fake PTY tests.
2. Add `onep ssh --tui`.
3. Manually verify SSH with real terminal resize.
4. Add `onep shell --tui`.
5. Add `onep run --tui`.
6. Decide whether any TUI path should become default after production feedback.

## Open Risks

- Native `node-pty` packaging can fail on some Node/platform combinations.
- Some full-screen children already manage alternate screen buffers; footer rendering must be tested with vim, ssh, code, and shell prompts before default enablement.

## Tasks

- [x] Finalize optional TUI command flags, footer shape, color semantics, and fallback semantics in `docs/20260613/cli-tui-runtime/api-contract.md`
  - Commit: `745c2bc`
- [x] Add capability detector
  - Commit: `b421aa1`
- [x] Add footer planner and formatter
  - Commit: `85cc1cb`
- [x] Add PTY adapter
  - Commit: `d101e48`
- [x] Add runtime orchestrator
  - Commit: `4dfcbd9`
- [x] Add status collector
  - Commit: `653523c`

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
