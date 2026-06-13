# Test Progress: cli-tui

**Engineer:** test-cli-tui
**Scope:** Verify default CLI TUI runtime behavior.

## Test Plan

- Capability detection:
  - TTY true on macOS/Linux allows TUI.
  - non-TTY disables TUI.
  - `TERM=dumb` disables TUI.
  - Windows disables TUI for V1.
  - tiny terminal disables TUI.

- Footer layout:
  - 18 or more rows uses three footer rows.
  - 14 to 17 rows uses two footer rows.
  - 10 to 13 rows uses one footer row.
  - fewer than 10 rows falls back.
  - long values truncate without wrapping.
  - traffic totals render only on line 2.
  - traffic totals are right-aligned to terminal width.
  - footer line 3 renders route path from `path.nodes` as `a-b-c`.
  - account, tenant, and ping render without labels.
  - account, tenant, and ping use three high-contrast macaron color segments.
  - ping color is mint under 100ms, butter yellow from 100ms to 299ms, coral red at 300ms or above, and gray when unknown.
  - path renders without a `Path:` label and uses macaron node colors.
  - ANSI colors do not break visible-width truncation or right alignment.

- Runtime:
  - child PTY receives terminal rows minus footer rows.
  - resize recomputes geometry and redraws footer.
  - child exit code propagates.
  - PTY load failure falls back when requested.

- Command behavior:
  - `onep ssh host`
  - `onep ssh host -p 2222`
  - `onep shell`
  - `onep run code .`
  - compatible `--tui` stripping

## Tasks

- [x] Add unit tests for capability detection
- [x] Add unit tests for footer row planning
- [x] Add unit tests for footer line formatting
- [x] Add fake PTY runtime tests
- [x] Add command parsing tests
- [x] Run CLI tests
- [x] Run proxy package tests

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
