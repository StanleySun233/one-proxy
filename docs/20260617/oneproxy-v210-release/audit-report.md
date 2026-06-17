# OneProxy v2.1.0 Full Audit Report

## Scope

This audit covers the OneProxy node runtime, panel API, panel web console, Chrome extension, TypeScript CLI, Go CLI, and VS Code extension. The audited runtime commit is `65411e7` (`fix(panel): ship final schema baseline`).

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
- A standing old database observed during release checks still had numbered migration history and no final access-path rows. Deploying the final image over that database would keep the old data shape and leave users without usable latest access paths.
- Raw panel web `tsc --noEmit` still reads a stale `.next/types/validator.ts` generated file that references a removed duplicate audit route. Source TypeScript checks pass when stale `.next` output is excluded.
- GitHub Actions image workflows pass, but both runs emit the upstream `actions/checkout@v4` Node.js 20 deprecation warning.

## Frontend And UX Findings

Panel web improvements:

- The access-path editor now exposes latest modes and protocols directly and derives service type from the selected mode.
- The UI rejects unusable listener definitions instead of letting invalid paths reach runtime use.
- Route health and access-path topology are surfaced in the console.
- Auth handling moved away from persistent raw account tokens in browser `localStorage` for the production path.
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

A source scan of panel API, panel web, and Docker files at `65411e7` found no matches for the removed migration/runtime and old enum markers.

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

- Panel image workflow: https://github.com/StanleySun233/one-proxy/actions/runs/27671854441
- Node image workflow: https://github.com/StanleySun233/one-proxy/actions/runs/27671854468
- Immutable test tag: `v2.1.0-rc.65411e7`
- `ghcr.io/stanleysun233/oneproxy-panel:v2.1.0-rc.65411e7`: `sha256:4803a7bf3a1b94be9a0e951c4ff3c13a596d222ce5ab73d0f49fd30c7c9e7d4a`
- `ghcr.io/stanleysun233/oneproxy-panel-base:v2.1.0-rc.65411e7`: `sha256:dfa63195389d6c82e9152121d43a69d76eacec55525463ff0de592ea2aa00d32`
- `ghcr.io/stanleysun233/oneproxy-node:v2.1.0-rc.65411e7`: `sha256:0a19998dd5011f0a065889fad1a7dba96bbb1542778514e136af6ffdb369f567`
- `ghcr.io/stanleysun233/oneproxy-node-base:v2.1.0-rc.65411e7`: `sha256:62fd4e5b070b1fbc2391dd05eb61b0844ded749d2d60b2b7dd727e5a707fe8ce`

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

## Release Decision

The code is suitable for a fresh v2.1.0 final-schema deployment after direct provisioning of final access paths and route state.

The code is not suitable as an automatic upgrade over an old non-empty panel database. That would require migration compatibility, which is explicitly out of scope for the final delivery. The correct final-version path is to create a fresh final schema or rebuild the database state directly in the final model.
