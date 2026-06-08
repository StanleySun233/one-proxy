# Tenant Permission API Audit

## Permission Model

- Read: a tenant can read resources that are directly granted or visible through granted chains, scopes, links, and access paths.
- Use: a tenant can reference visible resources in newly created tenant-owned resources and can run proxy traffic through visible nodes.
- Manage: a tenant admin can update, delete, and re-grant only resources explicitly granted with `manage`.
- Super admin: can operate globally, or within a selected tenant context when the endpoint requires one.

## Fixes Applied

- `GET /api/nodes` now includes nodes visible through direct node grants, granted chains, granted node links, granted access paths, access-path relay nodes, and granted scopes.
- Proxy token validation now allows a selected tenant to run through any visible node, not only directly granted nodes.
- Proxy session ingest now accepts sessions for visible nodes and still rejects node-token mismatches.
- Node health history now accepts visible nodes.
- Existing-node bootstrap token creation now requires direct node `manage`.

## API Inventory

### Public and Auth

| API | Permission result |
| --- | --- |
| `GET /healthz` | Public health check. |
| `GET /api/setup/status` | Public setup status. |
| `GET /api/enums` | Public enum metadata. |
| `POST /api/auth/login` | Public credential exchange. |
| `POST /api/auth/refresh` | Public refresh-token exchange. |
| `POST /api/auth/logout` | Authenticated account only. |

### Tenant and Account

| API | Permission result |
| --- | --- |
| `GET /api/tenants` | Authenticated account sees own tenants; super admin sees all. |
| `POST /api/tenants` | Super admin only. |
| `GET /api/tenants/{tenantId}` | Super admin or member of the requested tenant. |
| `PATCH /api/tenants/{tenantId}` | Super admin or requested tenant admin. |
| `DELETE /api/tenants/{tenantId}` | Super admin or requested tenant admin. |
| `GET /api/tenants/{tenantId}/memberships` | Super admin or member of the requested tenant. |
| `PUT /api/tenants/{tenantId}/memberships/{accountId}` | Super admin or requested tenant admin. |
| `DELETE /api/tenants/{tenantId}/memberships/{accountId}` | Super admin or requested tenant admin. |
| `GET /api/accounts` | Super admin or active tenant admin. |
| `POST /api/accounts` | Super admin only. |
| `PATCH /api/accounts/{accountId}` | Super admin, or self password change only. |
| `DELETE /api/accounts/{accountId}` | Super admin only. |
| `GET /api/groups` | Super admin only. |
| `POST /api/groups` | Super admin only. |
| `GET /api/groups/{groupId}` | Super admin only. |
| `PUT /api/groups/{groupId}` | Super admin only. |
| `DELETE /api/groups/{groupId}` | Super admin only. |
| `GET /api/groups/{groupId}/accounts` | Super admin only. |
| `PUT /api/groups/{groupId}/accounts` | Super admin only. |
| `PUT /api/groups/{groupId}/scopes` | Super admin only. |

### Grants

| API | Permission result |
| --- | --- |
| `GET /api/grants/tenants` | Super admin or active tenant admin. |
| `GET /api/grants?resourceType=&resourceId=` | Super admin, or active tenant admin with resource `manage`. |
| `PUT /api/grants/{resourceType}/{resourceId}/{tenantId}` | Super admin, or active tenant admin with resource `manage`; last manage grant is protected. |
| `DELETE /api/grants/{resourceType}/{resourceId}/{tenantId}` | Super admin, or active tenant admin with resource `manage`; last manage grant is protected. |

### Core Resource Console

| API | Permission result |
| --- | --- |
| `GET /api/overview` | Active tenant scoped. |
| `GET /api/nodes` | Active tenant visible nodes. |
| `PATCH /api/nodes/{nodeId}` | Active tenant admin with direct node `manage`. |
| `DELETE /api/nodes/{nodeId}` | Active tenant admin with direct node `manage`; shared resource delete is protected. |
| `GET /api/nodes/{nodeId}/access/manage` | Active tenant admin with direct node `manage`. |
| `POST /api/nodes/{nodeId}/approve` | Active tenant admin with direct node `manage`. |
| `POST /api/nodes/{nodeId}/reject` | Active tenant admin with direct node `manage`. |
| `GET /api/nodes/pending` | Active tenant admin; scoped to visible nodes. |
| `GET /api/nodes/transports` | Active tenant visible nodes only. |
| `GET /api/nodes/health` | Active tenant visible nodes only. |
| `GET /api/nodes/health/history` | Active tenant visible target node only. |
| `POST /api/nodes/bootstrap/token` | Active tenant admin; new node uses visible parent and scope, existing node requires direct node `manage`. |
| `GET /api/nodes/bootstrap/tokens/unconsumed` | Active tenant admin; scoped to directly managed bootstrap targets. |
| `DELETE /api/nodes/bootstrap/tokens/{tokenId}` | Active tenant admin; scoped to directly managed bootstrap targets. |
| `POST /api/policies/publish` | Active tenant admin. |
| `GET /api/policies/revisions` | Active tenant scoped. |

### Proxy Resources

| API | Permission result |
| --- | --- |
| `GET /api/proxy/scopes` | Active tenant visible scopes. |
| `POST /api/proxy/scopes` | Active tenant admin; creates tenant-owned manage grant. |
| `PATCH /api/proxy/scopes/{scopeId}` | Active tenant admin with scope `manage`. |
| `DELETE /api/proxy/scopes/{scopeId}` | Active tenant admin with scope `manage`; shared and in-use deletes are protected. |
| `GET /api/proxy` | Active tenant visible chains. |
| `GET /api/proxy/{chainId}` | Active tenant visible chain. |
| `POST /api/proxy` | Active tenant admin; referenced scopes and nodes must be visible. |
| `PATCH /api/proxy/{chainId}` | Active tenant admin with chain `manage`; referenced scopes and nodes must be visible. |
| `DELETE /api/proxy/{chainId}` | Active tenant admin with chain `manage`; shared deletes are protected. |
| `GET /api/proxy/{chainId}/probe` | Active tenant direct chain grant. |
| `POST /api/proxy/{chainId}/probe` | Active tenant visible chain. |
| `POST /api/proxy/validate` | Active tenant visible nodes and links. |
| `POST /api/proxy/preview` | Active tenant scoped validation. |
| `GET /api/proxy/links` | Active tenant visible node links. |
| `POST /api/proxy/links` | Active tenant admin; source and target nodes must be visible. |
| `PATCH /api/proxy/links/{linkId}` | Active tenant admin with link `manage`; source and target nodes must be visible. |
| `DELETE /api/proxy/links/{linkId}` | Active tenant admin with link `manage`; shared deletes are protected. |
| `GET /api/proxy/paths` | Active tenant visible access paths. |
| `POST /api/proxy/paths` | Active tenant admin; referenced chain and nodes must be visible. |
| `PATCH /api/proxy/paths/{pathId}` | Active tenant admin with path `manage`; referenced chain and nodes must be visible. |
| `DELETE /api/proxy/paths/{pathId}` | Active tenant admin with path `manage`; shared deletes are protected. |
| `GET /api/proxy/routes` | Active tenant visible route rules. |
| `GET /api/proxy/routes/{ruleId}` | Active tenant visible route rule. |
| `POST /api/proxy/routes` | Active tenant admin; referenced chain or scope must be visible. |
| `PATCH /api/proxy/routes/{ruleId}` | Active tenant admin with route rule `manage`; referenced chain or scope must be visible. |
| `DELETE /api/proxy/routes/{ruleId}` | Active tenant admin with route rule `manage`; shared deletes are protected. |
| `POST /api/proxy/routes/validate` | Active tenant scoped validation. |
| `GET /api/proxy/routes/suggestions` | Active tenant visible route rules. |

### Extension, Direct, Audit

| API | Permission result |
| --- | --- |
| `GET /api/proxy/extension/bootstrap` | Authenticated account with active tenant; resources scoped to active tenant and account groups. |
| `POST /api/proxy/extension/direct/session` | Active tenant visible access path. |
| `GET /api/proxy/extension/page/status` | Active tenant scoped. |
| `GET /api/audit/proxy/sessions` | Active tenant scoped in-memory proxy sessions. |
| `GET /api/audit/proxy/events` | Active tenant scoped in-memory proxy events. |
| `GET /api/audit/business/events` | Super admin or active tenant admin; query forced to tenant unless global super admin. |
| `GET /api/audit/network/sessions` | Super admin or active tenant admin; query forced to tenant unless global super admin. |
| `GET /api/audit/dashboard` | Super admin or active tenant admin; query forced to tenant unless global super admin. |

### Node Agent

| API | Permission result |
| --- | --- |
| `POST /api/nodes/enroll` | Bootstrap-token based enrollment. |
| `POST /api/nodes/exchange` | Enrollment-secret based node exchange. |
| `GET /api/node/agent/policy` | Node token only. |
| `GET /api/node/agent/auth/validate` | Node token only. |
| `POST /api/node/agent/heartbeat` | Node token only; node ID forced from token. |
| `POST /api/node/agent/cert/renew` | Node token only; node ID forced from token. |
| `POST /api/node/agent/transports` | Node token only; node ID forced from token. |
| `POST /api/node/agent/direct/candidates` | Node token only; node ID forced from token. |
| `GET /api/node/agent/direct/link/plan` | Node token only. |
| `POST /api/node/agent/direct/status` | Node token only; peer link must be authorized by store. |
| `POST /api/node/agent/direct/client/session/validate` | Node token plus issued direct-session punch token. |
| `POST /api/node/agent/proxy/token/validate` | Node token plus user proxy token; active tenant must be able to use the node. |
| `POST /api/node/agent/proxy/sessions` | Node token; each reported session tenant must be able to use the reporting node. |

## Residual Risk

- Route handlers and service checks are now aligned with the tenant resource model.
- Store-level list methods still return empty slices on SQL errors, matching current project behavior.
- Runtime verification on `camelbot` confirmed the deployed MySQL supports `JSON_CONTAINS` and the `astar` tenant resolves six visible nodes with `use` permission under the new visibility query.
