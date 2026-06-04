# Product Requirements: Node Local Console

## Scope

Build the first phase of a node-local web console for One Proxy nodes. The console is a read-only operational surface for the current node: status, health, audit, policy snapshot, and diagnostics. It must not provide complex management or configuration editing.

## Requirements

- The node HTTP listener should serve web pages at direct paths such as `/`, `/login`, `/overview`, `/health`, `/audit`, `/policy`, and `/diagnostics`.
- Node-local APIs should live under `/api/local/*`.
- Existing node runtime proxy behavior must remain available for non-console traffic.
- The node console should use the same simple visual language as the panel where practical.
- The node console must authenticate through the control plane, but this first phase must not modify panel frontend or backend code.
- Node authorization that depends on new panel APIs is deferred until panel/API changes are manually approved.
- The node console is mainly for audit and status querying. It must not expose configuration editing, restart, reload, certificate renewal, or other state-changing management actions.
- Extension API paths should be updated toward the new panel API shape. The extension only connects to panel by default, not node.
- Do not implement compatibility branches for old API paths.

## Deferred Until Approval

- Panel API route migration from `/api/v1/*` to `/api/*`.
- Panel web API client route migration.
- Panel endpoint for validating whether the logged-in account can manage a specific node.
- Full node login authorization based on panel-side node manage permission.
