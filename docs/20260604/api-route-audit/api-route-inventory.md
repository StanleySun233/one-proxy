# API Route Inventory

## Backend Routes

### Panel API

Core:
- `GET /healthz`
- `GET /api/setup/status`
- `POST /api/setup/test`
- `POST /api/setup/key`
- `POST /api/setup/init`
- `GET /api/enums`
- `POST /api/auth/login`
- `POST /api/auth/refresh`
- `POST /api/auth/logout`
- `GET /api/overview`

Accounts, tenants, groups, grants:
- `GET /api/accounts`
- `POST /api/accounts`
- `PATCH /api/accounts/{accountId}`
- `DELETE /api/accounts/{accountId}`
- `GET /api/tenants`
- `POST /api/tenants`
- `GET /api/tenants/{tenantId}`
- `PATCH /api/tenants/{tenantId}`
- `DELETE /api/tenants/{tenantId}`
- `GET /api/tenants/{tenantId}/memberships`
- `PUT /api/tenants/{tenantId}/memberships/{accountId}`
- `DELETE /api/tenants/{tenantId}/memberships/{accountId}`
- `GET /api/groups`
- `POST /api/groups`
- `GET /api/groups/{groupId}`
- `PATCH /api/groups/{groupId}`
- `DELETE /api/groups/{groupId}`
- `PUT /api/groups/{groupId}/accounts`
- `PUT /api/groups/{groupId}/scopes`
- `GET /api/grants`
- `PUT /api/grants/{resourceType}/{resourceId}/{tenantId}`
- `DELETE /api/grants/{resourceType}/{resourceId}/{tenantId}`
- `GET /api/grants/tenants`

Nodes and policies:
- `GET /api/nodes`
- `PATCH /api/nodes/{nodeId}`
- `DELETE /api/nodes/{nodeId}`
- `GET /api/nodes/{nodeId}/access/manage`
- `POST /api/nodes/{nodeId}/reject`
- `POST /api/nodes/{nodeId}/approve`
- `POST /api/nodes/bootstrap/token`
- `GET /api/nodes/bootstrap/tokens/unconsumed`
- `DELETE /api/nodes/bootstrap/tokens/{tokenId}`
- `POST /api/nodes/enroll`
- `POST /api/nodes/exchange`
- `GET /api/nodes/pending`
- `GET /api/nodes/health`
- `GET /api/nodes/health/history`
- `GET /api/nodes/transports`
- `GET /api/policies/revisions`
- `POST /api/policies/publish`

Proxy:
- `GET /api/proxy`
- `POST /api/proxy`
- `PATCH /api/proxy/{chainId}`
- `DELETE /api/proxy/{chainId}`
- `POST /api/proxy/{chainId}/probe`
- `POST /api/proxy/validate`
- `POST /api/proxy/preview`
- `GET /api/proxy/paths`
- `POST /api/proxy/paths`
- `PATCH /api/proxy/paths/{pathId}`
- `DELETE /api/proxy/paths/{pathId}`
- `GET /api/proxy/scopes`
- `POST /api/proxy/scopes`
- `PATCH /api/proxy/scopes/{scopeId}`
- `DELETE /api/proxy/scopes/{scopeId}`
- `GET /api/proxy/links`
- `POST /api/proxy/links`
- `PATCH /api/proxy/links/{linkId}`
- `DELETE /api/proxy/links/{linkId}`
- `GET /api/proxy/routes`
- `POST /api/proxy/routes`
- `PATCH /api/proxy/routes/{ruleId}`
- `DELETE /api/proxy/routes/{ruleId}`
- `POST /api/proxy/routes/validate`
- `GET /api/proxy/routes/suggestions`
- `GET /api/proxy/extension/page/status`

Audit:
- `GET /api/audit/proxy/sessions`
- `GET /api/audit/proxy/events`

Node-agent:
- `GET /api/node/agent/policy`
- `GET /api/node/agent/auth/validate`
- `POST /api/node/agent/heartbeat`
- `POST /api/node/agent/cert/renew`
- `POST /api/node/agent/transports`
- `POST /api/node/agent/direct/candidates`
- `GET /api/node/agent/direct/link/plan`
- `POST /api/node/agent/direct/status`
- `POST /api/node/agent/proxy/token/validate`
- `POST /api/node/agent/proxy/sessions`

Control relay:
- `POST /api/control/relay/probe`

### Node API

Local console:
- `POST /api/local/login`
- `POST /api/local/logout`
- `GET /api/local/session`
- `GET /api/local/status`
- `GET /api/local/health`
- `GET /api/local/audit`
- `GET /api/local/policy`
- `GET /api/local/diagnostics`

Node runtime/control:
- `GET /healthz`
- `POST /api/control/relay/probe`
- `POST /api/node/bootstrap/attach`
- `GET /api/node/tunnel/connect`
- Proxy-forwarded upstream paths:
  - `/api/nodes/enroll`
  - `/api/nodes/exchange`
  - `/api/node/agent/policy`
  - `/api/node/agent/auth/validate`
  - `/api/node/agent/heartbeat`
  - `/api/node/agent/cert/renew`
  - `/api/node/agent/transports`
  - `/api/node/agent/proxy/sessions`
  - `/api/node/agent/direct/candidates`
  - `/api/node/agent/direct/link/plan`
  - `/api/node/agent/direct/status`

Node web routes, not API:
- `/`
- `/login`
- `/overview`
- `/health`
- `/audit`
- `/policy`
- `/diagnostics`
- `/assets/*`

### Panel Web BFF Routes

- `[...path]`
- `/api/audit/[...path]`
- `/healthz`

## Frontend-Requested API Routes

### Panel Web

Current base:
- `CONTROL_PLANE_PROXY_BASE = /api`

Logical request paths:
- `/setup/status`
- `/setup/test`
- `/setup/key`
- `/setup/init`
- `/enums`
- `/auth/login`
- `/auth/refresh`
- `/auth/logout`
- `/overview`
- `/accounts`
- `/accounts/{accountId}`
- `/tenants`
- `/tenants/{tenantId}`
- `/tenants/{tenantId}/memberships`
- `/tenants/{tenantId}/memberships/{accountId}`
- `/groups`
- `/groups/{groupId}`
- `/groups/{groupId}/accounts`
- `/groups/{groupId}/scopes`
- `/grants`
- `/grants/tenants`
- `/grants/{resourceType}/{resourceId}/{tenantId}`
- `/nodes`
- `/nodes/{nodeId}`
- `/nodes/{nodeId}/reject`
- `/nodes/{nodeId}/approve`
- `/nodes/bootstrap/token`
- `/nodes/bootstrap/tokens/unconsumed`
- `/nodes/bootstrap/tokens/{tokenId}`
- `/nodes/pending`
- `/nodes/health`
- `/nodes/health/history`
- `/transports`
- `/policies/revisions`
- `/policies/publish`
- `/proxy`
- `/proxy/{chainId}`
- `/proxy/{chainId}/probe`
- `/proxy/validate`
- `/proxy/preview`
- `/proxy/links`
- `/proxy/links/{linkId}`
- `/proxy/paths`
- `/proxy/paths/{pathId}`
- `/proxy/routes`
- `/proxy/routes/{ruleId}`
- `/proxy/routes/validate`
- `/proxy/scopes`
- `/proxy/scopes/{scopeId}`

Panel web BFF also proxies:
- `/api/audit/{...path}` to panel `/api/audit/{...path}`

### Node Web

- `POST /api/local/login`
- `POST /api/local/logout`
- `GET /api/local/session`
- `GET /api/local/status`
- `GET /api/local/health`
- `GET /api/local/audit?limit=50`
- `GET /api/local/policy`
- `GET /api/local/diagnostics`

### Chrome Extension

Panel-facing:
- `GET /healthz`
- `POST /api/auth/login`
- `POST /api/auth/refresh`
- `POST /api/auth/logout`
- `GET /api/proxy/extension/bootstrap`
- `GET /api/proxy/extension/page/status`

Node-facing:
- `POST /api/control/relay/probe`

## Naming Rule Audit

### Current Rule Violations

None in source-owned API route paths after the approved migration.

### Canonical Renames Applied

- `/api/nodes/{nodeId}/manage-access` -> `/api/nodes/{nodeId}/access/manage`
- `/api/nodes/approve/{nodeId}` -> `/api/nodes/{nodeId}/approve`
- `/api/node-agent/*` -> `/api/node/agent/*`
- `/api/node-agent/proxy-token/validate` -> `/api/node/agent/proxy/token/validate`
- `/api/node-agent/proxy-sessions` -> `/api/node/agent/proxy/sessions`
- `/api/node-agent/direct/link-plan` -> `/api/node/agent/direct/link/plan`
- `/api/control-relay/probe` -> `/api/control/relay/probe`
- `/api/node-tunnel/connect` -> `/api/node/tunnel/connect`
- `/api/proxy/extension/page-status` -> `/api/proxy/extension/page/status`
- `/api/extension/bootstrap` -> `/api/proxy/extension/bootstrap`
- `/api/transports` -> `/api/nodes/transports`
- panel web `/api/v1/[...path]` -> `/api/[...path]`

### Allowed Exceptions

None recommended at this stage. Every current dashed segment has a straightforward slash-separated equivalent.
