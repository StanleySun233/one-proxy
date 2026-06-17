# Product Requirements: OneProxy v2.1.0 Release

**Date:** 20260617
**Requirement:** oneproxy-v210-release
**Release target:** v2.1.0

## Source Request

Start design and development with the develop-team workflow. Do not preserve compatibility for old versions; design for the newest release scenario only. After this version update, create a complete Docker test scenario on remote camelbot, finish scenario testing, replace the local node plus camelbot node and panel with the GitHub Actions built version, then query the database and complete real-user-style functional tests. After all gates pass, tag `v2.1.0`.

## Audit Baseline

The current codebase has four release-blocking risk groups:

- Node proxy authorization can fail open when the node is not bound or the validator is unavailable.
- User connection behavior is inconsistent across panel access paths, Chrome PAC routing, TypeScript CLI routing, Go CLI defaults, and node policy.
- Panel and client token handling exposes long-lived secrets through raw database storage, browser localStorage, extension local storage, and unauthenticated loopback IPC.
- The UI is card-heavy and route-state visibility is weaker than the operational task requires.

## Release Requirements

1. Node proxy, TCP access, UDP access, and reverse proxy paths must fail closed whenever control-plane validation is unavailable.
2. Node bootstrap must have no default shared join password and must not allow public unauthenticated takeover of an uninitialized node.
3. Panel access and refresh sessions must be stored as hashes, not raw tokens.
4. Panel web authentication must avoid persistent raw account tokens in browser localStorage for the production release path.
5. Direct QUIC sessions must verify node identity instead of using insecure TLS verification.
6. Public multi-hop chain forwarding must authenticate to the next hop, or route through a validated internal stream path.
7. HTTP forwarding must stream request and response bodies where possible, apply timeouts, and not retry unsafe non-idempotent requests.
8. Access paths must reject unusable listener definitions such as `listenPort=0` unless the API explicitly returns an assigned port.
9. Chrome extension, TypeScript CLI, Go CLI, and VS Code extension must consume the same latest access-path and route contract.
10. Legacy route-group, legacy token-wrapper, raw-token-validation, and duplicate audit proxy fallbacks must be removed unless the latest contract still explicitly requires them.
11. Panel and node consoles must present route, node, transport, token, and failure state in an operationally dense UI suitable for repeated debugging.
12. Chrome extension permissions and runtime messages must be narrowed to the required production surface.
13. TypeScript CLI daemon IPC must require a local secret or equivalent per-session authorization.
14. Setup APIs must be hardened against internal network probing, unsafe database names, and raw internal error disclosure.
15. Docker, README, node defaults, and CLI defaults must agree on exposed ports and release image names.
16. GitHub Actions must produce immutable testable panel and node images before the final tag, then publish final `v2.1.0` images on tag.
17. Remote camelbot testing must create an isolated Docker scenario for panel, node, database, and test targets before replacing the standing camelbot panel and node.
18. Final validation must include database queries for nodes, transports, access paths, route policies, and user session/proxy-token state.

## Non-Requirements

- No old client compatibility layer is required for this release.
- No downgrade path to previous API shapes is required.
- No broad defensive abstractions should be added unless they remove a concrete failure path listed above.

## Acceptance Gates

1. Local compile and unit tests pass for panel API, node API, panel web, Chrome extension, VS Code extension, Go CLI, and TypeScript CLI.
2. Local Docker scenario proves login, tenant selection, access path selection, HTTP proxy, CONNECT, SSH/TCP, direct path, route preview, and denied-path behavior.
3. Remote camelbot isolated Docker scenario passes the same real-user flow without touching the existing standing service until the scenario is green.
4. GitHub Actions built immutable images replace local node, camelbot node, and camelbot panel.
5. Database queries confirm expected node health, transport status, route/access-path state, and user token/session state.
6. Real-user-style function tests pass against the replaced local and camelbot deployments.
7. `v2.1.0` tag is created only after all gates have evidence.
