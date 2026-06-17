# API Contract: OneProxy v2.1.0 Release

## Scope

This contract defines the v2.1.0 production API and runtime contract for the panel, node, Chrome extension, TypeScript CLI, Go CLI, VS Code extension, Docker release images, deployment gates, and release evidence.

v2.1.0 is latest-only. Implementations must not preserve legacy client bootstrap wrappers, legacy route-group client state, raw proxy-token validation, unauthenticated daemon IPC, default node join passwords, or duplicate audit proxy fallbacks.

## Global API Rules

All HTTP JSON APIs return this envelope:

```ts
type ApiEnvelope<T> = {
  code: 0 | number;
  message: 'ok' | string;
  data: T | null;
};
```

Success responses use `code: 0`, `message: "ok"`, and an endpoint-specific `data` value. Error responses use the HTTP status as `code`, a stable snake_case `message`, and no internal error text. SQL errors, network addresses from internal probes, stack traces, DSN values, and raw exception messages must not be returned to clients.

Authentication headers:

```text
X-One-Proxy-Access-Token: <raw account access token>
X-One-Proxy-Refresh-Token: <raw account refresh token>
X-One-Proxy-Tenant-ID: <tenant id>
X-One-Proxy-Node-Token: <raw node access token>
```

Raw tokens are only shown at issuance time and only to the caller that requested them. At rest, account access tokens, account refresh tokens, panel proxy tokens, node API tokens, and bootstrap tokens are stored as lowercase SHA-256 hex hashes of the raw token. Token lookup always hashes the presented raw token before querying storage.

Timestamps are RFC3339 UTC strings. IDs are opaque strings. Ports are integers from `1` through `65535` unless a response explicitly includes an assigned port field.

## Account Session Contract

### `POST /api/auth/login`

Request:

```ts
type LoginRequest = {
  account: string;
  password: string;
};
```

Response:

```ts
type LoginResult = {
  account: Account;
  accessToken: string;
  refreshToken: string;
  expiresAt: string;
  mustRotatePassword: boolean;
  tenantMemberships: TenantMembership[];
  activeTenantId: string | null;
};
```

`LoginResult` must not include a proxy token. Clients obtain proxy tokens only through the latest bootstrap endpoint after tenant selection.

### `POST /api/auth/refresh`

The refresh token is supplied through `X-One-Proxy-Refresh-Token`. Browser production paths may use an HttpOnly cookie that the server-side panel route converts to the same header. A request body `refreshToken` is not part of the v2.1.0 production contract.

Response body is `LoginResult`. Refresh token storage uses hashes only.

### `POST /api/auth/logout`

Requires `X-One-Proxy-Access-Token`. The response data is:

```ts
type LogoutResult = {
  status: 'logged_out';
};
```

Panel web must not persist raw account access tokens or refresh tokens in browser `localStorage`. The production web path uses memory-only token handling or HttpOnly cookies. Chrome extension production state must not persist raw proxy tokens or long-lived account tokens in extension local storage.

## Latest Client Bootstrap

### `GET /api/proxy/extension/bootstrap`

Requires:

```text
X-One-Proxy-Access-Token: <raw account access token>
X-One-Proxy-Tenant-ID: <tenant id>
```

Response:

```ts
type ClientBootstrap = {
  schemaVersion: 'v2.1.0';
  account: Account;
  tenant: TenantMembership;
  policyRevision: string;
  fetchedAt: string;
  proxyToken: string;
  proxyTokenExpiresAt: string;
  nodes: BootstrapNode[];
  accessPaths: AccessPathSnapshot[];
  routes: RouteSnapshot[];
  routeEvaluation: RouteEvaluationContract;
};
```

`groups`, route-group wrappers, and token wrapper variants are not part of the v2.1.0 client bootstrap contract.

```ts
type BootstrapNode = {
  id: string;
  name: string;
  mode: 'edge' | 'relay' | 'internal' | string;
  scopeKey: string;
  parentNodeId: string;
  enabled: boolean;
  status: string;
  publicHost?: string;
  publicPort?: number;
};

type AccessPathSnapshot = {
  id: string;
  name: string;
  chainId: string;
  mode: 'forward' | 'reverse' | 'direct' | 'tcp' | 'udp';
  protocol: 'http' | 'https' | 'connect' | 'tcp' | 'udp' | 'quic';
  serviceType: 'http_forward_proxy' | 'reverse_proxy' | 'tcp_access' | 'udp_access' | 'direct_quic';
  targetNodeId: string;
  entryNodeId: string;
  relayNodeIds: string[];
  listenHost: string;
  listenPort: number;
  targetProtocol: 'http' | 'https' | 'tcp' | 'udp' | 'ssh' | string;
  targetHost: string;
  targetPort: number;
  targetSni: string;
  tlsMode: 'passthrough' | 'terminate' | 'direct_verify' | '';
  authMode: 'proxy_token';
  enabled: boolean;
  options: Record<string, string>;
  topology: TopologyHop[];
  health: AccessPathHealth;
};

type TopologyHop = {
  nodeId: string;
  nodeName: string;
  mode: string;
  scopeKey: string;
  publicHost?: string;
  publicPort?: number;
  transport: 'public_http' | 'public_https' | 'reverse_ws' | 'direct_quic' | 'internal_stream' | string;
};

type AccessPathHealth = {
  status: 'available' | 'degraded' | 'unavailable' | 'unknown';
  reason: string;
  checkedAt: string;
};

type RouteSnapshot = {
  id: string;
  priority: number;
  matchType: 'domain' | 'domain_suffix' | 'ip' | 'ip_cidr' | 'protocol' | 'default';
  matchValue: string;
  actionType: 'chain' | 'direct' | 'deny';
  chainId: string;
  accessPathId: string;
  destinationScope: string;
  enabled: boolean;
  topology: TopologyHop[];
};

type RouteEvaluationContract = {
  defaultClientMode: 'direct';
  defaultNodeMode: 'deny';
  ruleOrder: 'priority_asc_then_id_asc';
  noMatchNodeDenyReason: 'route_not_found';
  supportedMatchTypes: RouteSnapshot['matchType'][];
  supportedActions: RouteSnapshot['actionType'][];
};
```

Bootstrap must include only tenant-authorized nodes, access paths, and routes. A route with `actionType: "chain"` must include an enabled `accessPathId` that is usable by the client transport. A route with `actionType: "deny"` must not include a proxy topology.

## Access Path Management

### `GET /api/proxy/paths`

Returns `AccessPathSnapshot[]` for the active tenant.

### `POST /api/proxy/paths`

Request:

```ts
type CreateAccessPathRequest = {
  chainId: string;
  name: string;
  mode: AccessPathSnapshot['mode'];
  protocol: AccessPathSnapshot['protocol'];
  serviceType: AccessPathSnapshot['serviceType'];
  targetNodeId: string;
  entryNodeId: string;
  relayNodeIds: string[];
  listenHost: string;
  listenPort: number;
  targetProtocol: AccessPathSnapshot['targetProtocol'];
  targetHost: string;
  targetPort: number;
  targetSni: string;
  tlsMode: AccessPathSnapshot['tlsMode'];
  authMode: 'proxy_token';
  options: Record<string, string>;
};
```

Response is the created `AccessPathSnapshot`.

### `PATCH /api/proxy/paths/{pathId}`

Request is `CreateAccessPathRequest` plus:

```ts
type UpdateAccessPathRequest = CreateAccessPathRequest & {
  enabled: boolean;
};
```

Response is the updated `AccessPathSnapshot`.

### `DELETE /api/proxy/paths/{pathId}`

Response:

```ts
type DeleteResult = {
  status: 'deleted';
};
```

### `GET /api/proxy/paths/{pathId}/delete-impact`

Returns the access-path delete impact for the active tenant.

### Access Path Validation

Create and update operations fail with `400 invalid_access_path` when any required latest-contract field is unusable.

Validation rules:

- `listenPort=0` is rejected. v2.1.0 does not define automatic listener port assignment.
- `listenPort`, `targetPort`, `publicPort`, relay ports, TCP access ports, UDP access ports, and direct QUIC ports must be `1..65535` whenever present.
- `entryNodeId`, `targetNodeId`, `relayNodeIds`, and `chainId` must reference enabled tenant-visible resources.
- `authMode` for public access paths is always `proxy_token`.
- `mode`, `protocol`, `serviceType`, and `targetProtocol` must be internally consistent.
- Reverse proxy access paths require a non-empty reverse target and still require proxy-token authorization.
- Direct QUIC access paths require node identity material in the direct session contract.

## Route Evaluation

Clients evaluate `ClientBootstrap.routes` locally. Panel UI route preview, Chrome PAC, TypeScript CLI daemon routing, Go CLI direct selection, VS Code SSH config generation, and node proxy authorization use the same rules.

Evaluation input:

```ts
type RouteEvaluationInput = {
  url?: string;
  host: string;
  port: number;
  protocol: 'http' | 'https' | 'connect' | 'ssh' | 'tcp' | 'udp' | string;
  accessPathId?: string;
};
```

Evaluation output:

```ts
type RouteEvaluationResult = {
  mode: 'proxy' | 'direct' | 'deny';
  source: 'policy' | 'default_direct' | 'default_deny' | 'local_safety_direct';
  routeId: string;
  chainId: string;
  accessPathId: string;
  targetHost: string;
  targetPort: number;
  protocol: string;
  topology: TopologyHop[];
  denyReason: '' | 'route_not_found' | 'route_denied' | 'access_path_unavailable' | 'node_unavailable';
};
```

Algorithm:

1. Normalize host to lowercase. Normalize absent ports from protocol defaults: HTTP `80`, HTTPS `443`, SSH `22`.
2. Safety-direct the control-plane host, loopback hosts, and the selected local helper listener to avoid proxy loops.
3. Filter to `enabled` routes.
4. Sort by ascending `priority`, then ascending `id`.
5. Match rules:
   - `domain`: exact host match.
   - `domain_suffix`: `example.com`, `.example.com`, and `*.example.com` match `example.com` and subdomains.
   - `ip`: exact IP literal match.
   - `ip_cidr`: IPv4 CIDR match.
   - `protocol`: normalized protocol match.
   - `default`: always matches.
6. Apply the first match:
   - `chain`: return `mode: "proxy"` with the route `accessPathId` and topology.
   - `direct`: return `mode: "direct"` with no topology.
   - `deny`: return `mode: "deny"` with `denyReason: "route_denied"`.
7. If no rule matches:
   - Browser and CLI client-side routing returns `mode: "direct"`, `source: "default_direct"`.
   - Node-controlled public proxy, TCP access, UDP access, and reverse proxy paths return `mode: "deny"`, `source: "default_deny"`, `denyReason: "route_not_found"` unless proxy-token validation explicitly returns `allowLocalProxy: true`.

## Proxy Token Validation

### `POST /api/node/agent/proxy/token/validate`

Requires `X-One-Proxy-Node-Token`. Node agents send only the token hash:

```ts
type ProxyTokenValidateRequest = {
  tokenHash: string;
  accessPathId: string;
  targetHost: string;
  targetPort: number;
  protocol: string;
  routeId?: string;
};
```

The `token` request field is forbidden in v2.1.0. Raw-token validation fallbacks must be removed.

Success response:

```ts
type ProxyTokenValidation = {
  valid: true;
  tenantId: string;
  accountId: string;
  expiresAt: string;
  cacheTtlSeconds: number;
  allowLocalProxy: boolean;
  scopes: string[];
  accessPathIds: string[];
  routeIds: string[];
};
```

Invalid, expired, tenant-mismatched, access-path-mismatched, or route-mismatched tokens return `401 invalid_proxy_token`.

Fail-closed requirements:

- Missing proxy token denies the request.
- Missing node token denies validation.
- Invalid node token denies validation.
- Control-plane validation timeout or network failure denies the data-path request.
- Missing token validator in node runtime denies public forward proxy, reverse proxy, TCP access, UDP access, and direct session validation.
- Positive validation cache entries expire at `min(expiresAt, now + cacheTtlSeconds)`.
- Negative and error results are not cached as authorization grants.

## Node Bootstrap and Node Auth

### Panel Bootstrap Token APIs

`POST /api/nodes/bootstrap/token` creates one raw bootstrap token for one target node:

```ts
type CreateBootstrapTokenRequest = {
  targetType: 'node';
  targetId: string;
  nodeName: string;
  nodeMode: string;
  scopeKey: string;
  parentNodeId: string;
  publicHost: string;
  publicPort: number;
};

type BootstrapToken = {
  id: string;
  token?: string;
  targetType: 'node';
  targetId: string;
  nodeName: string;
  nodeMode: string;
  scopeKey: string;
  parentNodeId: string;
  publicHost: string;
  publicPort: number;
  expiresAt: string;
  createdAt: string;
};
```

The raw `token` field is returned only by the create response. List responses from `GET /api/nodes/bootstrap/tokens/unconsumed` must omit `token`.

`DELETE /api/nodes/bootstrap/tokens/{tokenId}` revokes an unconsumed bootstrap token.

### Node Enrollment

`POST /api/nodes/enroll` is unauthenticated but requires a valid one-use bootstrap token:

```ts
type EnrollNodeRequest = {
  token: string;
  name: string;
  mode: string;
  scopeKey: string;
  parentNodeId: string;
  publicHost: string;
  publicPort: number;
};

type EnrollNodeResult = {
  node: BootstrapNode;
  enrollmentSecret: string;
  approvalState: 'pending' | 'approved';
};
```

`POST /api/nodes/{nodeId}/approve` requires a tenant admin or super admin. It returns the node access token once:

```ts
type ApproveNodeEnrollmentResult = {
  node: BootstrapNode;
  accessToken: string;
  trustMaterial: string;
  expiresAt: string;
};
```

`POST /api/nodes/exchange` requires `nodeId` and `enrollmentSecret`. It returns `ApproveNodeEnrollmentResult` only after approval.

### Local Node Attach

`POST /api/node/bootstrap/attach` is a node-local endpoint. It is disabled unless an explicit `NODE_JOIN_PASSWORD` is configured. There is no default join password.

Attach request:

```ts
type NodeBootstrapAttachRequest = {
  password: string;
  newPassword?: string;
  controlPlaneUrl: string;
  nodeId: string;
  nodeAccessToken: string;
  nodeName: string;
  nodeMode: string;
  nodeScopeKey: string;
  nodeParentId: string;
  nodePublicHost: string;
  nodePublicPort: number;
  localIps: string[];
};
```

Attach response:

```ts
type NodeBootstrapAttachResult = {
  connectionStatus: 'connected';
  localIps: string[];
  nodeListenAddr: string;
  nodeHttpsListenAddr: string;
  controlPlaneBound: true;
  mustRotatePassword: boolean;
};
```

If the node is already bound, attach cannot overwrite its binding without a valid current join password and a valid node access token. Public unauthenticated takeover of an uninitialized node is forbidden.

### Node Agent APIs

All node agent APIs require `X-One-Proxy-Node-Token`:

```text
GET  /api/node/agent/auth/validate
GET  /api/node/agent/policy
POST /api/node/agent/heartbeat
POST /api/node/agent/cert/renew
POST /api/node/agent/transports
POST /api/node/agent/direct/candidates
GET  /api/node/agent/direct/link/plan
POST /api/node/agent/direct/status
POST /api/node/agent/direct/client/session/validate
POST /api/node/agent/proxy/token/validate
POST /api/node/agent/proxy/sessions
```

`POST /api/node/agent/direct/candidates` request:

```ts
type DirectCandidatesRequest = {
  udpListenPort: number;
  natType: string;
  candidates: DirectCandidate[];
  observedAt: string;
  directIdentity: DirectNodeIdentity;
};
```

`GET /api/node/agent/direct/link/plan` response:

```ts
type DirectLinkPlanResult = {
  nodeId: string;
  links: DirectLinkPlan[];
};

type DirectLinkPlan = {
  linkId: string;
  peerNodeId: string;
  role: 'dialer' | 'listener' | string;
  preferredTransport: 'direct_quic';
  fallbackTransport: 'relay_ws_parent';
  punchToken: string;
  expiresAt: string;
  peerCandidates: DirectCandidate[];
  peerIdentity: DirectNodeIdentity;
};
```

Node-to-node direct QUIC stream clients must use `peerIdentity`. A link plan without complete peer identity is unusable for direct QUIC and must not become a connected direct peer.

`GET /api/node/agent/auth/validate` returns:

```ts
type NodeAuthValidation = {
  nodeId: string;
};
```

If node auth validation fails or the control plane is unavailable at startup, the node must not expose public proxy, TCP, UDP, reverse, or direct listeners as authorized traffic paths. A node without a non-empty policy snapshot must keep policy-controlled paths closed.

## Direct QUIC Session Contract

### `POST /api/proxy/extension/direct/session`

Requires account access token and tenant header.

Request:

```ts
type ClientDirectSessionRequest = {
  accessPathId: string;
  clientId: string;
  targetHost: string;
  targetPort: number;
};
```

Response:

```ts
type ClientDirectSession = {
  sessionId: string;
  accessPathId: string;
  targetNodeId: string;
  targetHost: string;
  targetPort: number;
  relayEntryHost: string;
  relayEntryPort: number;
  punchToken: string;
  expiresAt: string;
  nodeCandidates: DirectCandidate[];
  nodeIdentity: DirectNodeIdentity;
};

type DirectCandidate = {
  type: 'host' | 'srflx' | string;
  address: string;
  port: number;
  protocol: 'udp';
  stunServer?: string;
  priority?: number;
};

type DirectNodeIdentity = {
  nodeId: string;
  serverName: string;
  certificateFingerprintSha256: string;
  trustMaterial: string;
};
```

`trustMaterial` is the PEM-encoded certificate material used to build a scoped trust pool for that peer. Direct QUIC clients must verify the peer identity using `nodeIdentity` or `peerIdentity`. `InsecureSkipVerify` is not valid in the v2.1.0 production path.

### `POST /api/node/agent/direct/client/session/validate`

Requires node token.

Request:

```ts
type ClientDirectSessionValidateRequest = {
  sessionId: string;
  punchToken: string;
  targetHost: string;
  targetPort: number;
};
```

Success response:

```ts
type ClientDirectSessionValidateResult = {
  valid: true;
  targetHost: string;
  targetPort: number;
};
```

Invalid direct sessions return `401 invalid_direct_session`.

## Data Path Contract

### HTTP Forward Proxy

Clients authenticate to public forward proxy listeners with either:

```text
Proxy-Authorization: Bearer <proxy token>
```

or browser-compatible proxy credentials:

```text
Proxy-Authorization: Basic base64("token:<proxy token>")
```

HTTP forwarding streams request and response bodies where possible. Forwarding applies bounded connect, header, and idle timeouts. Unsafe non-idempotent requests such as `POST`, `PATCH`, and `DELETE` are not retried automatically.

### HTTP CONNECT

CONNECT requests use the same proxy-token validation contract. Failed validation returns `407 proxy_auth_required`. Missing route or denied route returns `403 route_not_found` or `403 route_denied`.

Public multi-hop CONNECT forwarding must either use the validated internal stream path between nodes or authenticate to the next public hop with a valid hop credential. Unauthenticated next-hop public CONNECT is forbidden.

### Reverse Proxy

Reverse proxy access paths require proxy-token authorization. Missing token, invalid token, missing validator, or control-plane validation failure returns `401 reverse_auth_required`. Reverse proxy forwarding streams request and response bodies and must not leak `X-One-Proxy-*` internal headers to upstream targets.

### TCP Access

The first line sent by the client is:

```ts
type TcpAccessAuthFrame = {
  token: string;
  accessPathId: string;
  targetHost: string;
  targetPort: number;
  nextNodeId?: string;
  remainingHopNodeIds?: string[];
  chainNodeIds?: string[];
};
```

Success response frame:

```ts
type TcpAccessResponse = {
  status: 'connected';
  message?: '';
};
```

Failure response frame uses `status: "failed"` and stable messages such as `auth_required`, `invalid_target`, `route_denied`, or `stream_registry_unavailable`.

TCP access fails closed when the authorizer is nil, the control plane is unavailable, the token is invalid, or the access path is not enabled.

### UDP Access

Packet request:

```ts
type UdpAccessPacket = {
  token: string;
  accessPathId: string;
  targetHost: string;
  targetPort: number;
  data: string;
};
```

`data` is base64-encoded UDP payload bytes. Node implementations must enforce a configured maximum decoded packet size and an idle timeout.

Response:

```ts
type UdpAccessResponse = {
  status: 'ok' | 'failed';
  message?: string;
  data?: string;
};
```

UDP access fails closed under the same conditions as TCP access.

## Client Sync and Local Runtime

All production clients consume `ClientBootstrap` from `GET /api/proxy/extension/bootstrap`.

Chrome extension requirements:

- Store route state as `accessPaths`, `routes`, `nodes`, `policyRevision`, and `fetchedAt`.
- Compile PAC rules from `RouteEvaluationContract`.
- Do not expose raw session or proxy tokens in runtime message responses.
- Restrict extension permissions to the production surface required for proxy, storage, alarms, and active-tab status.

TypeScript CLI requirements:

- Store local state as active tenant plus active access path, not active route group.
- Remove legacy bootstrap token wrappers and route group fallbacks.
- The daemon protects loopback IPC with a per-session secret.
- IPC requests include `X-One-Proxy-Daemon-Secret`.
- Missing or invalid daemon secret returns `401 daemon_auth_required`.
- Local HTTP proxy and CONNECT handling validate upstream proxy responses and stream request and response bodies.

Go CLI requirements:

- Login writes proxy token files with mode `0600`.
- The password CLI flag is removed from production command help.
- Default ports match the release port contract.
- Direct QUIC verifies `DirectNodeIdentity`.

VS Code extension requirements:

- SSH generation selects a concrete `accessPathId`.
- Generated `ProxyCommand` uses the latest access-path endpoint, proxy token, and target metadata.
- Session material is stored only in VS Code SecretStorage.

## Release Image Contract

GitHub Actions must build immutable test images before the final tag and final release images on tag.

Image repositories:

```text
ghcr.io/stanleysun233/oneproxy-panel-base
ghcr.io/stanleysun233/oneproxy-panel
ghcr.io/stanleysun233/oneproxy-node-base
ghcr.io/stanleysun233/oneproxy-node
```

Pre-tag test image tags:

```text
sha-<40-char-git-sha>
```

Final tag image tags for v2.1.0:

```text
v2.1.0
latest
```

Every published image must include OCI labels:

```text
org.opencontainers.image.revision=<git sha>
org.opencontainers.image.version=<image tag>
org.opencontainers.image.source=<repository url>
org.opencontainers.image.created=<RFC3339 timestamp>
```

Canonical exposed ports:

| Component | Port | Purpose |
|-----------|------|---------|
| Panel web container | `2886` | Browser and client-facing panel endpoint |
| Panel API internal | `2887` | Internal API behind panel web container |
| Node HTTP | `2988` | Public node HTTP, CONNECT, reverse, tunnel, and local console |
| Node HTTPS | `2989` | HTTPS node endpoint |
| Node TCP access | `2990` | Optional TCP access listener |
| Node UDP access | `2991` | Optional UDP access listener |
| Node direct QUIC | `2992` | Optional direct QUIC listener |
| Local helper SOCKS5 | `1080` | Optional local client helper |

Docker examples, README defaults, node defaults, CLI defaults, and deployment scripts must use these ports.

## Deployment Contract

Deployment proceeds in this order:

1. Build and publish immutable pre-tag test images for the exact git SHA under test.
2. Run the isolated local Docker scenario.
3. Run the isolated camelbot Docker scenario using separate container names, network names, volumes, and database names from the standing service.
4. Replace the local node with the GitHub Actions built node image.
5. Replace the camelbot node with the GitHub Actions built node image.
6. Replace the camelbot panel with the GitHub Actions built panel image.
7. Run database-backed and real-user-style validation against replaced deployments.
8. Create and push `v2.1.0` only after all evidence gates pass.

The standing camelbot panel or node must not be stopped, replaced, or modified by the isolated scenario. Replacement scripts require explicit image references by immutable tag or digest.

## Test Evidence Contract

The release test plan records one evidence object per gate:

```ts
type ReleaseEvidence = {
  release: 'v2.1.0';
  gitSha: string;
  generatedAt: string;
  images: {
    panel: ImageEvidence;
    node: ImageEvidence;
    panelBase: ImageEvidence;
    nodeBase: ImageEvidence;
  };
  localScenario: ScenarioEvidence;
  camelbotScenario: ScenarioEvidence;
  replacementDeployment: DeploymentEvidence;
  databaseChecks: DatabaseEvidence[];
  functionalTests: FunctionalEvidence[];
  finalTagReady: boolean;
};

type ImageEvidence = {
  repository: string;
  tag: string;
  digest: string;
  workflowRunUrl: string;
};

type ScenarioEvidence = {
  target: 'local' | 'camelbot';
  startedAt: string;
  completedAt: string;
  status: 'passed' | 'failed';
  imageDigests: string[];
  checks: string[];
  logPath: string;
};

type DeploymentEvidence = {
  target: 'local-node' | 'camelbot-node' | 'camelbot-panel';
  previousImage: string;
  replacementImage: string;
  replacementDigest: string;
  healthStatus: 'passed' | 'failed';
  checkedAt: string;
};

type DatabaseEvidence = {
  target: 'local' | 'camelbot';
  queryName: string;
  sql: string;
  expected: string;
  observed: string;
  status: 'passed' | 'failed';
};

type FunctionalEvidence = {
  target: 'local' | 'camelbot';
  flow: string;
  status: 'passed' | 'failed';
  observed: string;
};
```

Required evidence gates:

- Compile and unit tests pass for panel API, node API, panel web, Chrome extension, VS Code extension, Go CLI, and TypeScript CLI.
- Local Docker scenario proves login, tenant selection, access path selection, HTTP proxy, CONNECT, SSH/TCP, direct path, route preview, and denied-path behavior.
- Camelbot isolated Docker scenario passes the same flow without touching the standing service.
- GitHub Actions built immutable images replace local node, camelbot node, and camelbot panel.
- Database queries confirm node health, transport status, access path state, route policy state, hashed session state, hashed proxy-token state, and node token state.
- Real-user-style functional tests pass against the replaced local and camelbot deployments.

Minimum database checks:

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

Token evidence must prove hash-shaped values are stored. It must not copy raw token values into release notes.

## Tag Gate

`v2.1.0` may be created only when:

- All implementation tasks depending on this contract are complete.
- All test evidence gates pass.
- Product verification confirms the implementation matches `docs/20260617/oneproxy-v210-release/product-requirements.md`.
- The pre-tag `sha-<40-char-git-sha>` images for the verified commit are published and tested; the tag workflow then publishes `v2.1.0` images from the same commit.
