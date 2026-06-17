# OneProxy v2.1.0 Release Test Plan

## Scope

This plan defines the release gates for OneProxy `v2.1.0`: immutable image publication, isolated local and camelbot Docker scenarios, replacement deployment for the local node, camelbot node, and camelbot panel, database-backed evidence, real-user functional validation, and the final tag gate.

No isolated scenario may replace or stop the standing camelbot production panel or node. Replacement deployment requires an explicit `deploy` mode and an immutable image tag.

## Image And Artifact Gate

The node and panel image workflows must be run before the final release tag.

| Workflow | Dispatch Inputs | Required Evidence |
|----------|-----------------|-------------------|
| `one-proxy-node-image` | `image_version=v2.1.0-rc.<run_number>` or an explicit immutable test tag; `push_image=true` | Workflow run URL, git SHA, pushed `oneproxy-node-base` digest, pushed `oneproxy-node` digest |
| `one-proxy-panel-image` | `image_version=v2.1.0-rc.<run_number>` or an explicit immutable test tag; `push_image=true` | Workflow run URL, git SHA, pushed `oneproxy-panel-base` digest, pushed `oneproxy-panel` digest |

Acceptance criteria:

- Pre-tag image tags are immutable and are not `latest`.
- Published image labels include the tested git SHA, image version, source repository, and creation time.
- The same git SHA is used for the local scenario, camelbot isolated scenario, replacement deployment, database evidence, and product verification.
- The final `v2.1.0` tag is created only after all gates pass. The tag workflow must publish `v2.1.0` and `latest` for the node and panel repositories from the verified commit.

## Scenario Gates

### Local Docker Scenario

Use `scripts/test-v210-docker-scenario.sh`.

| Mode | Behavior |
|------|----------|
| `check` | Prints the `oneproxy-v210-local` project plan, required services, image tags, host ports, and local tool availability. This is the default mode. |
| `build` | Builds local panel base, panel, node base, and node images under the configured immutable test tag. |
| `run` | Recreates the isolated Compose project, starts MySQL, Redis, and panel, creates a tenant, scope, and node bootstrap token, starts an edge node, approves the pending node, creates a chain, access path, route group, and route, validates latest extension bootstrap, validates hashed proxy-token authorization through the node API, and queries database evidence. |
| `clean` | Removes only the isolated `oneproxy-v210-local` Compose project resources and volumes. |

Required evidence:

- Command, mode, image tag, git SHA, and timestamp.
- Panel `/healthz` result through the isolated host port.
- Node `/healthz` result through the isolated host port with `controlPlaneBound=true`.
- Latest bootstrap result with `schemaVersion=v2.1.0`, non-empty `nodes`, `accessPaths`, and `routes`, and no legacy `groups`.
- Proxy-token validation result through `POST /api/node/agent/proxy/token/validate` using only `tokenHash`.
- Database query output for the minimum database evidence set and hash-shape checks.
- Real-user functional results for the local target.

### Camelbot Isolated Scenario

Use `scripts/test-camelbot-v210-scenario.sh`.

Required environment:

| Variable | Purpose |
|----------|---------|
| `CAMELBOT_SSH_HOST` | SSH host used for the isolated scenario. Defaults to `camelbot`. |
| `CAMELBOT_V210_REMOTE_DIR` | Remote scenario directory. Defaults to `oneproxy-v210-isolated`. |
| `ONEPROXY_CAMELBOT_V210_PROJECT` | Remote Compose project. Defaults to `oneproxy-v210-camelbot`. |
| `ONEPROXY_IMAGE_TAG` | Immutable pre-tag image version, or pass the tag as the second script argument. |
| `ONEPROXY_PANEL_IMAGE_REPO` | Panel image repository. |
| `ONEPROXY_NODE_IMAGE_REPO` | Node image repository. |
| `ONEPROXY_CAMELBOT_V210_ADMIN_PASSWORD` | Isolated panel admin password. |
| `ONEPROXY_CAMELBOT_V210_TENANT_NAME` | Isolated scenario tenant name. |
| `ONEPROXY_CAMELBOT_V210_NODE_NAME` | Isolated scenario node name. |

Acceptance criteria:

- `check` runs through SSH and does not create, start, stop, or remove containers.
- `build` pulls the configured immutable images into the isolated remote directory and Compose project.
- `run` recreates only the isolated MySQL, Redis, panel, and node services under the `oneproxy-v210-camelbot` Compose project and validates the same latest bootstrap, node enrollment, proxy-token hash authorization, and database evidence gates as the local scenario.
- `clean` removes only resources owned by the isolated Compose project.
- Standing camelbot production container names, networks, and volumes are not reused by the isolated scenario.

## Replacement Deployment Gate

Use `scripts/deploy-v210-release-images.sh`.

Targets:

| Target | Image | Default Operation |
|--------|-------|-------------------|
| `local-node` | `ONEPROXY_NODE_IMAGE_REPO:<immutable_tag>` | Check current local node image and status. |
| `camelbot-node` | `ONEPROXY_NODE_IMAGE_REPO:<immutable_tag>` | Reuse `scripts/deploy-camelbot-node.sh` for check and deploy. |
| `camelbot-panel` | `ONEPROXY_PANEL_IMAGE_REPO:<immutable_tag>` | Check current remote panel image and health. |

Modes:

| Mode | Behavior |
|------|----------|
| `dry-run` | Prints the exact target and immutable replacement image. |
| `check` | Runs non-destructive current-image and health checks. This is the default mode. |
| `deploy` | Performs replacement deployment. This mode is required for local node, camelbot node, and camelbot panel replacement. |

Acceptance criteria:

- The script rejects missing tags and mutable tags such as `latest`.
- The local node replacement preserves the current environment, renames the previous container, starts the new immutable image, and checks `/healthz`.
- The camelbot node path reuses the existing `deploy-camelbot-node.sh` deploy contract.
- The camelbot panel replacement preserves the current environment, renames the previous container, starts the new immutable panel image, and checks `/healthz`.
- Rollback evidence records the previous image and backup container name whenever deployment starts.

## Database Evidence

Run the database evidence against both local and camelbot after replacement deployment. Record output as structured evidence and do not copy raw account tokens, proxy tokens, node access tokens, bootstrap tokens, or refresh tokens into release notes.

Minimum queries:

```sql
SELECT id, status, enabled, public_host, public_port FROM nodes ORDER BY id;
SELECT node_id, transport_type, direction, address, status FROM node_transports ORDER BY node_id, transport_type;
SELECT id, chain_id, entry_node_id, target_node_id, listen_port, target_host, target_port, enabled FROM node_access_paths ORDER BY id;
SELECT id, priority, match_type, match_value, action_type, chain_id, destination_scope, enabled FROM route_rules ORDER BY priority, id;
SELECT id, destination_scope, enabled FROM chains ORDER BY id;
SELECT id, account_id, access_token_hash, refresh_token_hash, expires_at FROM sessions ORDER BY id;
SELECT id, node_id, token_hash, expires_at FROM node_api_tokens ORDER BY node_id, id;
SELECT id, token_hash, target_type, target_id, consumed_at, expires_at FROM bootstrap_tokens ORDER BY id;
```

Hash-shape checks:

```sql
SELECT id, access_token_hash REGEXP '^[0-9a-f]{64}$' AS access_hash_shape, refresh_token_hash REGEXP '^[0-9a-f]{64}$' AS refresh_hash_shape FROM sessions ORDER BY id;
SELECT id, token_hash REGEXP '^[0-9a-f]{64}$' AS token_hash_shape FROM node_api_tokens ORDER BY node_id, id;
SELECT id, token_hash REGEXP '^[0-9a-f]{64}$' AS token_hash_shape, consumed_at, expires_at FROM bootstrap_tokens ORDER BY id;
```

Expected evidence:

- Nodes are enabled only where expected and report the intended health status.
- Reverse, direct, TCP, UDP, and public transports show the expected status for the tested topology.
- Access paths use non-zero listener ports and point at the expected entry and target nodes.
- Route rules follow priority order and include the expected chain, direct, and deny decisions.
- Sessions, node API tokens, and bootstrap tokens store hash-shaped values only.
- Consumed bootstrap tokens have `consumed_at` set; active bootstrap tokens remain within their expiry window.

## Real-User Functional Tests

Run these flows against both the local replacement target and camelbot replacement target after database evidence is captured.

| Flow | Steps | Expected Result |
|------|-------|-----------------|
| Panel login and tenant selection | Log in through the panel, select the release tenant, refresh the page, and inspect browser storage. | The session works without persistent raw account tokens in browser `localStorage`. |
| Latest bootstrap | Fetch `/api/proxy/extension/bootstrap` through the panel path with an active tenant. | Response uses `schemaVersion=v2.1.0`, includes access paths and routes, and omits legacy groups and token wrappers. |
| Access path selection | Select an HTTP access path and route preview target in the panel. | The UI shows the selected access path, topology, route action, and health state. |
| HTTP proxy | Send an authenticated HTTP request through the node HTTP proxy. | Request succeeds through the selected route and records proxy session evidence. |
| CONNECT proxy | Open an authenticated CONNECT tunnel through the node. | Tunnel succeeds for allowed targets and records proxy session evidence. |
| SSH/TCP access | Use the generated TCP or SSH access path with the issued proxy material. | Connection reaches the intended target through the selected access path. |
| Direct path | Exercise the direct QUIC access path where the topology supports it. | Direct path verifies node identity and reports the expected transport state. |
| Denied path | Request a destination covered by a deny route or by no matching node route. | Request fails closed with the expected denial reason. |
| Chrome extension route sync | Sync the extension and preview the same target set. | PAC behavior and route preview match the bootstrap route contract. |
| TypeScript CLI route sync | Run daemon route and probe commands with the daemon IPC secret. | CLI route selection matches panel bootstrap and rejects missing daemon secret. |
| Go CLI and VS Code SSH | Generate and execute a release access path through Go CLI and VS Code SSH configuration. | Generated commands use concrete access-path identity and do not expose raw tokens on the command line. |

## Final Tag Gate

Before creating `v2.1.0`, record:

- Verified git SHA.
- Node and panel workflow run URLs.
- Immutable pre-tag image digests for panel base, panel, node base, and node.
- Local scenario evidence.
- Camelbot isolated scenario evidence.
- Replacement deployment evidence for local node, camelbot node, and camelbot panel.
- Database evidence for local and camelbot.
- Real-user functional evidence for local and camelbot.
- Product verification result against `docs/20260617/oneproxy-v210-release/product-requirements.md`.

`v2.1.0` may be created and pushed only after every item above is present and passing. After the tag workflow completes, record the final `v2.1.0` and `latest` image digests and confirm they were built from the verified commit.
