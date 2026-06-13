# Product Requirements: CLI TUI Runtime

## Problem

`onep run`, `onep shell`, and `onep ssh` currently hand the terminal directly to the child process. This preserves native behavior, but users lose OneProxy context while the command is running. They cannot see account, tenant, latency, traffic totals, or route path without leaving the running workflow.

The requested experience is a lightweight terminal shell where the child process keeps the main screen area and OneProxy owns a small fixed footer.

## Goal

Add an optional TUI runtime for interactive CLI sessions. The TUI must show the child process in the main terminal area and reserve a one to three line footer for OneProxy status.

Example footer:

```text
stanley  demo  32ms
                              Total ↑ 12.4 MB | ↓ 91.8 MB
user-entry-a-b-target
```

## Scope

- Support `onep ssh` first.
- Support `onep shell` and `onep run` after the SSH path is proven.
- Use a real PTY for TUI-enabled interactive sessions on macOS and Linux.
- Keep the existing direct stdio path available and unchanged when TUI is disabled or unsupported.
- Resize the child PTY to `terminal rows - footer rows`.
- Re-render the footer on status updates and terminal resize.
- Degrade automatically when the terminal is too small, non-interactive, dumb, or unsupported.
- Do not implement Windows PTY support in the first implementation.

## Non-Goals

- Do not replace the daemon, route matcher, or proxy transport.
- Do not build a full-screen application framework.
- Do not buffer or reinterpret child process output beyond PTY transport.
- Do not store SSH credentials, private keys, or agent state.
- Do not require root privileges.

## User Experience

TUI-capable commands may be invoked with:

```text
onep ssh <host> --tui
onep shell --tui
onep run --tui <command...>
```

The default remains the current stdio behavior until the PTY path has enough production evidence.

When TUI is active:

- The top area belongs to the child PTY.
- The bottom footer belongs to OneProxy.
- The footer uses one line at minimum and up to three lines when space allows.
- `Ctrl+C` is forwarded to the child by default, matching normal terminal expectations.

When TUI cannot run:

- The command falls back to the current stdio path.
- If the user explicitly requested `--tui`, the CLI prints one warning to stderr before fallback.

## Footer Content

The footer should display only operational state that helps while the command is running.

Minimum one-line footer:

```text
<account>  <tenant>  <latency>
```

Two-line footer:

```text
<account>  <tenant>  <latency>
                              Total ↑ <upload> | ↓ <download>
```

Three-line footer:

```text
<account>  <tenant>  <latency>
                              Total ↑ <upload> | ↓ <download>
<node-a>-<node-b>-<node-c>
```

The footer must not render labels such as `Account:`, `Tenant:`, `Ping:`, or `Path:`. Line 1 uses three high-contrast macaron color segments to distinguish account, tenant, and ping. Line 3 uses the same visual language for the route path. Line 2 is the only line with a textual hint, and it is right-aligned to the terminal width with only total traffic counters.

The third footer line renders the active route path in compact text form, such as `a-b-c`. It should reuse the same path payload shape used by the Chrome extension status bubble: `path.nodes[]` with `id`, `name`, `kind`, and `transport`.

The footer supports ANSI color when stdout is a color-capable TTY. Color is limited to the footer and must not be injected into the child PTY stream.

## Status Semantics

- Account: authenticated account email, account name, or `not logged in`.
- Tenant: active tenant id or name when available; otherwise `none`.
- Ping: most recent control-plane or daemon health latency; `--` when unknown.
- Total upload: cumulative bytes written by the child command through the observed proxy/session metrics when available; otherwise `--`.
- Total download: cumulative bytes read by the child command through the observed proxy/session metrics when available; otherwise `--`.
- Path: compact route path derived from `path.nodes`. Prefer `node.name`, fall back to `node.id`, and omit empty nodes.

## Color Semantics

- Line 1:
  - account segment: macaron mint background, dark text
  - tenant segment: macaron lavender background, dark text
  - ping segment: latency-coded macaron background, dark text
- Ping segment:
  - `--`: gray
  - `< 100ms`: mint green
  - `100ms..299ms`: butter yellow
  - `>= 300ms`: coral red
- Upload marker `↑`: blue.
- Download marker `↓`: cyan.
- Path:
  - user node: macaron mint
  - proxy nodes: alternating macaron lavender and butter yellow
  - web/target node: macaron coral
  - separators: muted gray
  - current failed or fallback segment, when known: yellow or red

When color is disabled, all footer content must remain readable as plain text.

## Compatibility Requirements

- Non-interactive commands and `--json` output must never use the TUI.
- Current `onep run` stdout/stderr byte stream behavior must remain available.
- CI and scripts should continue using the existing stdio path.
- macOS and Linux are first-class for V1.
- Windows should fall back to stdio until a ConPTY implementation is explicitly added.

## Acceptance Criteria

- `onep ssh --tui <host>` opens SSH through the same route plan as current `onep ssh`.
- The child PTY never writes into the footer during normal resize and output flows.
- Resizing the terminal updates child PTY rows and footer layout.
- A terminal smaller than the minimum height falls back to stdio or a single-line footer without corrupting the child display.
- If `node-pty` is unavailable or unsupported, the command still runs through the existing stdio implementation.
- Existing CLI tests keep passing.
