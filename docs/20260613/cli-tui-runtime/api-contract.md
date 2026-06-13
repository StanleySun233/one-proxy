# API Contract: CLI TUI Runtime

## Scope

This contract defines the default TUI runtime for interactive `onep` commands. It extends the existing CLI Session Proxy behavior without removing the stdio fallback for unsupported environments.

## Command Flags

The TUI footer is attempted by default for interactive command paths. `--tui` is retained only as a compatibility flag and is stripped before child command execution.

### `onep ssh <target> [options]`

Runs the existing SSH command plan inside the OneProxy TUI runtime when supported.

Existing SSH options remain valid:

```text
onep ssh <host> [-p <port>]
onep ssh <user>@<host> [-p <port>]
```

### `onep shell`

Runs the activated child shell inside the OneProxy TUI runtime when supported.

### `onep run <command...>`

Runs the wrapped child command inside the OneProxy TUI runtime when supported.

For `onep run`, `--tui` is parsed as a OneProxy option only before the child command begins. Values after the child command are passed to the child unchanged.

## TUI Activation Rules

The TUI runtime may start only when all conditions are true:

- The command is interactive.
- `process.stdin.isTTY`, `process.stdout.isTTY`, and `process.stderr.isTTY` are true.
- `process.env.TERM` is not `dumb`.
- Platform is `darwin` or `linux`.
- Terminal height is at least the configured minimum.
- PTY support is available.
- The command does not use `--json`.

If any condition fails:

- Print `! TUI failed to start; falling back to standard terminal mode.` to stderr exactly once and use the existing stdio path.
- Fallback must not change the child command, environment, stdio inheritance, or exit-code behavior used by the existing command path.

## PTY Geometry

Let:

```text
terminalColumns = process.stdout.columns
terminalRows = process.stdout.rows
footerRows = 1..3
minimumMainRows = 8
```

The child PTY must be sized as:

```text
childColumns = terminalColumns
childRows = max(terminalRows - footerRows, minimumMainRows)
```

Footer row count:

- `3` when `terminalRows >= 18`
- `2` when `terminalRows >= 14`
- `1` when `terminalRows >= 10`
- fallback to stdio when `terminalRows < 10`

On terminal resize:

- Recompute `footerRows`.
- Resize the child PTY.
- Redraw the footer.

## Rendering Contract

The renderer owns only the footer rows. Child PTY output must be written to the terminal without semantic parsing.

Footer rendering must:

- Save the current cursor position.
- Move to the footer start row.
- Clear each footer row.
- Write the footer lines truncated to terminal width.
- Right-align footer line 2 when traffic totals are displayed.
- Apply ANSI color only inside footer-owned rows when color is enabled.
- Compute visible width after stripping ANSI sequences.
- Restore the cursor position.

The renderer must hide implementation details from child output. It must not add timestamps, prefixes, or status text into the child PTY stream.

## Input Contract

Default behavior:

- All stdin bytes are forwarded to the child PTY.
- `Ctrl+C` is forwarded to the child PTY.
- `Ctrl+D` is forwarded to the child PTY.

## Status Snapshot

The footer renderer consumes this internal shape:

```ts
type TuiPathNode = {
  id: string;
  name: string;
  kind: 'user' | 'node' | 'web' | string;
  transport: string;
};

type TuiStatusSnapshot = {
  account: string;
  tenant: string;
  pingMs: number | null;
  uploadBytes: number | null;
  downloadBytes: number | null;
  path: {
    mode: string;
    transport: string;
    fallbackReason: string;
    nodes: TuiPathNode[];
  };
};
```

Initial data sources:

- Account: `tokens.json`.
- Tenant: `config.activeTenantId`.
- Ping: daemon IPC health timing when available; otherwise `null`.
- Upload and download totals: local daemon/session metrics when available; otherwise `null`.
- Path: existing route topology mapped to the Chrome extension status bubble path shape, equivalent to `path.nodes[]` from the extension payload.

## Color Contract

Color is optional and controlled by terminal capability detection. It must be disabled when:

- stdout is not a TTY.
- `NO_COLOR` is set.
- `TERM=dumb`.
- The TUI fallback path is used.

ANSI style mapping:

| Element | Condition | Style |
|---------|-----------|-------|
| Account segment | always | macaron mint background, dark foreground |
| Tenant segment | always | macaron lavender background, dark foreground |
| Ping segment | `null` | neutral gray background, dark foreground |
| Ping segment | `< 100ms` | macaron mint background, dark foreground |
| Ping segment | `100ms..299ms` | butter yellow background, dark foreground |
| Ping segment | `>= 300ms` | coral red background, dark foreground |
| Upload marker | always | blue |
| Download marker | always | cyan |
| Path user node | always | macaron mint background, dark foreground |
| Path proxy node | normal | alternating macaron lavender and butter yellow backgrounds, dark foreground |
| Path web node | always | coral background, dark foreground |
| Path separator | always | dim gray |
| Path fallback segment | fallback known | yellow |
| Path failed segment | failure known | red |

Footer formatter functions must return both styled text and visible width, or must apply width calculations with ANSI sequences stripped. Right alignment must use visible width.

## Footer Formatting

For three footer rows:

```text
<account>  <tenant>  <latency>
                              Total ↑ <upload> | ↓ <download>
<node-a>-<node-b>-<node-c>
```

For two footer rows:

```text
<account>  <tenant>  <latency>
                              Total ↑ <upload> | ↓ <download>
```

For one footer row:

```text
<account>  <tenant>  <latency>
```

Footer text rules:

- Do not render labels for account, tenant, ping, or path.
- Only line 2 may render a textual hint label: `Total`.
- Line 1 segments must remain distinguishable by high-contrast macaron colors.
- Line 3 path nodes must use the same high-contrast macaron visual treatment.

Path formatting rules:

- Use `path.nodes`.
- Prefer each node's `name`.
- Fall back to `id`.
- Remove empty labels.
- Join with `-`.
- Truncate from the middle when the rendered line exceeds terminal width.

## Dependency Contract

`node-pty` is a runtime dependency for the TUI path. TUI module contracts must be statically imported by command wiring so missing exports fail during build instead of disappearing at runtime.

## Exit Contract

- The CLI exits with the child process exit code.
- If the child exits by signal, return `1`, matching current behavior.
- Footer cleanup restores the terminal cursor and clears only footer-owned rows.
- The daemon session cleanup behavior remains the same as the existing command path.

## Test Contract

Unit tests must cover:

- TUI capability detection.
- Footer row selection by terminal height.
- Footer line truncation.
- Traffic total right alignment.
- ANSI color mapping and width calculation with color enabled.
- Path text derived from `path.nodes`.
- Status snapshot construction.
- Default TUI activation for `ssh`, `shell`, and `run`.
- Compatible `--tui` argument stripping for `ssh`, `shell`, and `run`.
- Fallback behavior when PTY is unavailable.

Integration or smoke tests should use a fake PTY adapter rather than requiring native `node-pty` in CI.
