# Product Requirements: Connection Resilience

## Background

OneProxy already has access paths, route rules, node health reporting, extension status bubbles, CLI access, and panel node management. Recent field testing showed that users still struggle to understand where a connection fails, why a route is not usable, and whether the system can recover without manual intervention.

This project combines six related improvements into one development plan: connection-state visibility, automatic reconnect, path self-healing, user-facing diagnostics, node runtime stability snapshots, and reduced manual configuration. The target is latest-version-only behavior with no old group model, old interface fallback, or distributed migration sequence.

## Goals

- Show a clear connection state machine across extension, CLI, node, and panel surfaces.
- Reconnect automatically after transient panel, node, proxy, or Chrome proxy-setting failures.
- Select and repair usable access paths with minimal user action.
- Expose actionable diagnostics that identify whether failure is local, first-hop, relay, target, auth, policy, or panel sync.
- Persist node runtime snapshots so panel status distinguishes missing report, stale report, degraded runtime, and reachable runtime.
- Automatically create or select chains and access paths for common relay and public-node flows.

## Non-Goals

- No compatibility with legacy group-based client contracts.
- No fallback to old node, panel, extension, or CLI interfaces.
- No multi-step SQL migration plan for final delivery.
- No new service process management by the client.

## User Outcomes

- A user can enable OneProxy and see the exact connection phase instead of only healthy, unreported, or unknown.
- A user can refresh the browser or lose temporary connectivity without having to manually log in, reselect paths, or reapply proxy settings.
- A user can create a relay or public node with defaults selected from available healthy resources.
- An operator can open panel diagnostics and determine which segment is failing without reading raw logs first.
- A node can report enough runtime state for the panel to avoid misleading healthy or unreported labels.

## Functional Requirements

### Connection State Machine

- Define canonical states: idle, resolving, syncing, selecting_path, probing_entry, probing_relay, applying_proxy, connected, degraded, reconnecting, failed.
- Record current phase, last successful phase, last error code, last error message, retry count, next retry time, selected route, selected access path, and selected topology.
- Use the same state vocabulary in extension status bubble, extension options, CLI status output, node local console, and panel operational views.

### Automatic Reconnect

- Extension reconnects with bounded exponential backoff after sync failure, entry probe failure, relay probe failure, proxy auth failure, or proxy setting drift.
- CLI session proxy reconnects after upstream tunnel or access path failure when the selected route remains valid.
- Reconnect attempts must be visible in diagnostics and must not hide terminal auth or policy errors.

### Path Self-Healing

- Route evaluation chooses an available access path automatically when multiple paths satisfy the route.
- Entry-node failure triggers path reselection if an alternative access path exists.
- Relay-chain creation defaults to existing healthy relay nodes when possible.
- Public-node creation derives public endpoint from node report and listen port instead of requiring manual host entry.

### Diagnostics

- Extension status bubble exposes route, access path, topology, every hop RTT, last error, retry state, and proxy target.
- Panel node and proxy views expose runtime snapshots and path diagnostics.
- CLI exposes a compact status command that reports panel sync, selected access path, proxy target, current state, last error, and retry state.

### Node Runtime Snapshot

- Node reports listener state, policy revision, panel reachability, upstream URL reachability, public endpoint, selected parent, relay session state, and recent runtime errors.
- Panel stores latest snapshot and recent history.
- Maintenance marks snapshots stale based on heartbeat age and avoids showing stale nodes as healthy.

### Automatic Configuration

- Access path creation is automatic for healthy node combinations required by route creation.
- Relay mode selects an existing healthy parent or relay candidate by default.
- Public mode uses reported public host and configured port by default.
- Route creation can complete without manually creating access paths first.

## Acceptance Criteria

- Extension shows a non-empty state and phase for every proxy route evaluation.
- Extension auto-recovers from a simulated first-hop probe failure once the entry endpoint becomes healthy.
- CLI status reports the same selected access path and phase as extension bootstrap data.
- Panel distinguishes unreported, stale, degraded, reconnecting, and healthy node states.
- Relay and public-node creation flows require fewer manual fields and validate connectivity before final save.
- Tests cover contract schemas, extension state transitions, CLI status output, node snapshot reporting, panel snapshot storage, and automatic access-path creation.
