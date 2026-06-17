# Product Verification: product-manager

**Scope:** Verify the v2.1.0 implementation against `docs/20260617/oneproxy-v210-release/product-requirements.md`.

## Tasks

- [x] Add residual v2.1.0 latest-contract gaps found during post-implementation audit to `dev-roadmap.md`
  - Commit: b40843b
- [x] Verify implementation against product requirements after final isolated scenario evidence
  - Evidence:
    - Runtime commit under audit: `65411e7`
    - Full audit report: `docs/20260617/oneproxy-v210-release/audit-report.md`
    - Local isolated scenario: pass with `v2.1.0-rc.65411e7`
    - Camelbot isolated scenario: pass with `v2.1.0-rc.65411e7`
    - Panel and node image workflows: pass
    - Final schema policy: `schema/final.sql` only; no goose runtime; no numbered SQL migration chain
    - Camelbot panel replacement script now rejects final-schema deploy without an explicit empty final database and confirmation
    - Final standing cutover script added for fresh panel DB, fresh panel/node volumes, fresh bootstrap, route creation, policy publish, bootstrap validation, proxy-token validation, and DB evidence
    - Standing replacement over an old non-empty panel database is intentionally withheld because this release does not include old-version database migration compatibility

## Verification Report

### Requirements Met

- Node proxy, TCP access, UDP access, and reverse proxy authorization now fail closed when validation is unavailable.
- Default shared node join password behavior has been removed.
- Panel account sessions, node API tokens, bootstrap tokens, and proxy tokens are stored as hashes.
- Panel web production auth no longer depends on persistent raw account tokens in browser `localStorage`.
- Direct QUIC paths require node identity material.
- Public next-hop CONNECT forwarding authenticates the next hop.
- HTTP forwarding streams bodies where possible and avoids unsafe retries for non-idempotent requests.
- Access paths reject unusable listener definitions.
- Chrome extension, TypeScript CLI, Go CLI, and VS Code extension consume latest access-path and route state.
- Old route-group, old token-wrapper, raw-token-validation, and duplicate audit proxy fallbacks have been removed from the release path.
- Panel and extension UX now exposes access-path, route, topology, and health state instead of old group state.
- Chrome extension permissions and runtime messages were narrowed.
- TypeScript CLI daemon IPC requires a local secret.
- Setup and API error paths were hardened against raw internal error disclosure.
- Docker image workflows produce immutable test images with recorded digests.
- Local and camelbot isolated Docker scenarios pass on the final-schema runtime.

### Gaps Found

- Raw panel web TypeScript checking still reads stale generated `.next` output unless `.next` is excluded or refreshed.
- The observed standing old panel database is not a valid automatic upgrade target for final-schema-only v2.1.0. It must be reset or directly provisioned into the final schema and route/access-path model.
- Final replacement deployment and `v2.1.0` tag creation should wait until the standing environment is prepared as a final-schema deployment target.
- Standing replacement now needs explicit authorization for final DB creation/use, fresh panel data volume use, service replacement, fresh node runtime binding, and final tag creation. Final panel secrets and local node parent URL can be resolved by the cutover script without printing secret values.
