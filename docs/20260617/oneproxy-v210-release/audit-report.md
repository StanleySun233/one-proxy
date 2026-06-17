# OneProxy v2.1.0 Full Audit Report

## Scope

This audit covers the OneProxy node runtime, panel API, panel web console, Chrome extension, TypeScript CLI, Go CLI, and VS Code extension. The final corrected panel runtime commit is `ca6d859` (`style(panel): hide all modal scrollbars`). Tags `v2.1.0` and `v2.1.1` were published before final panel delivery and were superseded by the immutable correction tag `v2.1.2`.

The release direction is latest-only. The final delivery must not preserve old bootstrap contracts, route-group client state, raw token validation, old access-path enum values, downgrade behavior, or incremental SQL files such as `00001` and `00002`.

## Executive Result

The v2.1.0 runtime has been brought to a latest-only contract:

- Panel schema creation now uses `apps/panel/api/schema/final.sql` as the only schema source for fresh databases.
- The panel API no longer runs goose migrations or numbered SQL upgrade chains.
- Access-path mode, protocol, service type, TLS mode, panel UI form state, and panel API validation have been aligned to the v2.1.0 contract only.
- Client bootstrap now uses `schemaVersion: "v2.1.0"`, `accessPaths`, and `routes`, without route-group wrappers.
- CLI and extension clients consume the latest bootstrap state and do not keep old group-oriented command or runtime paths.
- Token storage and validation paths were changed to hash raw tokens at rest and compare presented tokens by hash.
- Node proxy, TCP access, UDP access, and direct QUIC behavior were hardened to fail closed when authorization or identity validation is not available.

The remaining release risk is operational, not a compatibility task: an existing non-empty old database will not be upgraded by the final schema baseline. That is intentional for this release. Such environments must be reset or provisioned directly into the final schema before using the final image.

## User Connection Chain Experience

The user connection model now centers on access paths and route snapshots:

- Panel-created access paths require concrete listener ports and internally consistent `mode`, `protocol`, `serviceType`, and `targetProtocol` values.
- Routes point to access paths and include topology and health information for client-side preview.
- Relay bootstrap auto-selects an enabled parent node from the current tenant inventory and exposes a parent URL probe before the command is copied.
- Edge bootstrap no longer requires users to type a public host; the node reports its detected public IP at enrollment while the panel keeps the selected public port.
- Generated bootstrap commands now embed the same immutable node image tag as the panel image build.
- Chrome PAC routing, Chrome route preview, TypeScript CLI daemon routing, Go CLI access-path selection, and VS Code SSH generation use the same route and access-path model.
- `onep init` rejects bootstrap payloads that are not `v2.1.0` and requires non-empty latest `accessPaths`.
- CLI visible commands now use access-path terminology instead of route-group terminology.
- The node proxy authorization path validates proxy tokens through the panel using hash-shaped inputs and fails closed on unavailable validation.

This removes the most confusing old flow: users no longer receive a bootstrap payload with groups, then have different clients reinterpret that state into separate proxy, PAC, CLI, and SSH behaviors.

## Network Security Findings

Resolved high-risk findings:

- Node proxy authorization no longer falls open when the node is unbound or panel validation is unavailable.
- TCP and UDP access paths use closed authorization semantics.
- Public next-hop CONNECT forwarding authenticates the next hop instead of forwarding anonymously.
- HTTP forwarding streams bodies where possible, applies timeouts, and avoids retrying unsafe non-idempotent requests.
- Direct QUIC sessions require peer identity material and fail closed without it.
- Default shared node join password behavior was removed.
- Panel access tokens, refresh tokens, node API tokens, bootstrap tokens, and proxy tokens are stored and validated as SHA-256 hashes.
- Proxy-token validation no longer falls back to raw token lookup.
- Panel setup and API responses avoid exposing internal raw error text.
- Chrome extension permissions and runtime message surfaces were narrowed.
- TypeScript CLI daemon IPC requires a local secret.

Residual security and operations risks:

- Final schema initialization skips non-empty databases. This avoids upgrade compatibility code, but it means old production databases must not be treated as automatically supported final-version inputs.
- The standing environment was reset after this risk was observed. The release is still gated by manual panel setup, post-setup node bootstrap, standing database evidence, and real-user functional validation.
- The camelbot panel deployment script now rejects final-schema deployment unless an explicit empty final database, final panel secrets, and `ONEPROXY_FINAL_SCHEMA_CONFIRM=deploy-final-schema` are provided.
- `scripts/deploy-v210-post-setup-nodes.sh` now provides the continuation path after manual panel setup; it starts fresh node runtimes, provisions latest access paths and routes, publishes policy, validates latest bootstrap, validates proxy-token hashes, and prints database evidence.
- Raw panel web `tsc --noEmit` still reads a stale `.next/types/validator.ts` generated file that references a removed duplicate audit route. Source TypeScript checks pass when stale `.next` output is excluded.
- GitHub Actions image workflows pass, but both runs emit the upstream `actions/checkout@v4` Node.js 20 deprecation warning.

## Frontend And UX Findings

Panel web improvements:

- The access-path editor now exposes latest modes and protocols directly and derives service type from the selected mode.
- The UI rejects unusable listener definitions instead of letting invalid paths reach runtime use.
- Route health and access-path topology are surfaced in the console.
- Auth handling moved away from persistent raw account tokens in browser `localStorage` for the production path.
- Browser refresh now restores the session from the refresh-token path, so a normal F5 reload does not force a login when the refresh session is still valid.
- Node status rendering now shows one lifecycle status instead of combining stale health and lifecycle badges in the same cell.
- The route-group view action now opens the corresponding route-rule list with the selected group filter.
- Console modals and dialog panels now hide scrollbars across modal-level and child scroll regions while preserving scroll behavior.
- Console visual tokens and layout density were revised to reduce card chrome and make repeated operational scanning easier.

Chrome extension improvements:

- Popup, options, PAC routing, monitor, status bubble, and smoke fixtures use access paths and routes.
- The extension no longer presents old route-group wording in visible surfaces.
- Multiple access-path proxy challenges are handled instead of caching a single proxy target.

CLI improvements:

- TypeScript CLI route, probe, daemon, and init flows now use latest access-path bootstrap state.
- `group list|use` was replaced by `access-path list|use`.
- Daemon IPC calls include the daemon secret.
- CLI HTTP proxy behavior validates CONNECT responses and streams HTTP bodies.

## Old Compatibility Removal

The following old-version paths were removed or rejected:

- Goose migration runtime and numbered SQL files under `apps/panel/api/migrations`.
- `schema/001_init.sql`.
- Old panel access-path enum values such as route-chain and upstream-pull modes.
- Old raw TCP/UDP service names and SOCKS-style protocol aliases.
- Old TLS mode values that are not part of the final access-path model.
- Old client bootstrap `groups` state and token wrapper variants.
- Raw proxy-token validation fallback.
- CLI group commands and old bootstrap fallback behavior.
- Duplicate panel audit proxy fallback route.

A source scan of panel API, panel web, and Docker files at `65411e7` found no matches for the removed migration/runtime and old enum markers. The post-UX release image at `3cf4562` also removes `oneproxy-node:latest` from the panel-generated bootstrap command path.

## Verification Evidence

Static and unit checks:

- `cd apps/node/api && go test ./...`: pass.
- `cd apps/panel/api && go test ./...`: pass.
- `cd apps/extension/cli && go test ./internal/proxycommand`: pass.
- Panel web source TypeScript check excluding stale `.next`: pass.
- Raw panel web TypeScript check including `.next`: fails only on stale generated validator output for the removed duplicate audit route.
- `node --test apps/cli/test/*.mjs`: pass, 50 tests.
- `node apps/extension/chrome/tools/domain_suffix_test.mjs`: pass.
- `node apps/extension/chrome/tools/validate_extension.mjs`: pass with `chrome_extension_static_ok`.
- `node apps/extension/chrome/tools/build_background_bundle.mjs`: pass.

Image verification:

- Panel image workflow: https://github.com/StanleySun233/one-proxy/actions/runs/27682252948
- Node image workflow: https://github.com/StanleySun233/one-proxy/actions/runs/27682252778
- Immutable test tag: `v2.1.0-rc.3cf4562`
- `ghcr.io/stanleysun233/oneproxy-panel:v2.1.0-rc.3cf4562`: `sha256:ae936f12bd415f7cf264a4162706f739595f43bd0c53185f9f4bd030baaa6f83`
- `ghcr.io/stanleysun233/oneproxy-panel-base:v2.1.0-rc.3cf4562`: `sha256:e52c16867a20c16c0840270a33843466dfbf30975610477ff8bc8baa8ff7e964`
- `ghcr.io/stanleysun233/oneproxy-node:v2.1.0-rc.3cf4562`: `sha256:e5c1797e65f021dad46c1550bf2b9ca4d5a14c70748008109266c907c80bd18b`
- `ghcr.io/stanleysun233/oneproxy-node-base:v2.1.0-rc.3cf4562`: `sha256:ec661e4ae6b2087cbe8cb1a9d8926792c580a4bbb381af6012a9b2d697d8a295`

Final corrected image verification:

- Panel image workflow: https://github.com/StanleySun233/one-proxy/actions/runs/27683952017
- Node image workflow: https://github.com/StanleySun233/one-proxy/actions/runs/27683952034
- Final correction tag: `v2.1.2`
- `ghcr.io/stanleysun233/oneproxy-panel:v2.1.2`: `sha256:011c0bcdff63e9d13cff0370b4d800e41cbc38e4617a9893c8866d79e19e1f46`
- `ghcr.io/stanleysun233/oneproxy-panel-base:v2.1.2`: `sha256:9bc72ce8f1ff14db1a14536cd636405ab2e64a02693a305dd5a2b64e3f3cb7c5`
- `ghcr.io/stanleysun233/oneproxy-node:v2.1.2`: `sha256:99f0b648ebbdaa8bd2444e48c7f0f9f532ba9cdffa2a481f7f41489c53865c89`
- `ghcr.io/stanleysun233/oneproxy-node-base:v2.1.2`: `sha256:c2a8f731eab7ba2ff12b9fb9c561a9724cd9fdbda2f460c02047d9801e8e6994`

Local isolated scenario:

- Command: `ONEPROXY_IMAGE_TAG=v2.1.0-rc.65411e7 scripts/test-v210-docker-scenario.sh run`
- Result: pass.
- Evidence: panel health ok; tenant, scope, bootstrap token, node enrollment, node approval, chain, access path, route group, and route created; bootstrap returned `schema=v2.1.0 access_paths=1 routes=1`; proxy-token validation ok.
- Database hash-shape evidence: sessions, node API tokens, and bootstrap tokens were stored as hash-shaped values.

Camelbot isolated scenario:

- Command: `ONEPROXY_IMAGE_TAG=v2.1.0-rc.65411e7 ONEPROXY_MYSQL_IMAGE=mysql:8.0 ONEPROXY_REDIS_IMAGE=redis:7-alpine scripts/test-camelbot-v210-scenario.sh run`
- Result: pass.
- Evidence: panel health ok; tenant, scope, bootstrap token, node enrollment, node approval, chain, access path, route group, and route created; bootstrap returned `schema=v2.1.0 access_paths=1 routes=1`; proxy-token validation ok.
- Database hash-shape evidence: sessions, node API tokens, and bootstrap tokens were stored as hash-shaped values.

Standing post-setup evidence:

- The user completed manual setup against database `one_proxy`.
- Standing panel image: `ghcr.io/stanleysun233/oneproxy-panel:v2.1.0-rc.3cf4562`; `/healthz` returns `status=ok`.
- Standing remote edge image: `ghcr.io/stanleysun233/oneproxy-node:v2.1.0-rc.3cf4562`; `/healthz` returns `controlPlaneBound=true`.
- Standing local relay image: `ghcr.io/stanleysun233/oneproxy-node:v2.1.0-rc.3cf4562`; `/healthz` returns `controlPlaneBound=true`.
- Node DB state: `camelbot` is `edge`, `healthy`, public endpoint `103.214.172.211:2988`; `astar-58` is `relay`, `healthy`, parent node `1`.
- Health snapshots exist for nodes `1` and `3` with `{"runtime":"up"}`.
- Relay transport state includes `reverse_ws_parent` from node `3` to `ws://103.214.172.211:2988/api/node/tunnel/connect?parentNodeId=1` with `connected`.
- Parent URL probe for `http://103.214.172.211:2988` returns `reachable=true`, `statusCode=200`, `mode=proxy-node`, and `controlPlaneBound=true`.
- The deployed panel static chunk contains `ghcr.io/stanleysun233/oneproxy-node:v2.1.0-rc.3cf4562` and no `oneproxy-node:latest` reference.
- Final standing panel image: `ghcr.io/stanleysun233/oneproxy-panel:v2.1.2`; `/api/setup/status` returns `configured=true`, and `/healthz` returns `dbBackend=mysql`, `httpAddr=127.0.0.1:2887`, and `status=ok`.
- The deployed `v2.1.2` panel static chunks contain the modal scrollbar selectors for `.console-modal *` and `.dialog-panel *`.
- Local node cleanup for manual retry is complete; no local `one-proxy-node` container or `one-proxy-node-runtime*` volume remains.

## Release Decision

The code is suitable for a fresh v2.1.0 final-schema deployment after manual panel setup and direct provisioning of final access paths and route state. The final standing panel has been replaced with the audited immutable `v2.1.2` image and verifies setup health, route-group view navigation, modal scrollbar suppression, relay-parent UX, and edge-public-IP UX. Node redeployment is intentionally left for manual retry after local node cleanup.

The code is not suitable as an automatic upgrade over an old non-empty panel database. That would require migration compatibility, which is explicitly out of scope for the final delivery. The correct final-version path is to create a fresh final schema or rebuild the database state directly in the final model.
