# API Contract

## Naming Rules

- API paths use `/api/...`.
- Panel web proxies panel API through `/api/[...path]`.
- A route segment must be a single word.
- No dashed route segment is allowed.
- `healthz` remains outside `/api` for service probes.

## Panel Backend

Setup:
- `GET /healthz`
- `GET /api/setup/status`
- `POST /api/setup/test`
- `GET /api/setup/key`
- `POST /api/setup/init`

Account and tenant:
- `GET /api/enums`
- `POST /api/auth/login`
- `POST /api/auth/refresh`
- `POST /api/auth/logout`
- `GET /api/overview`
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
- `PUT /api/groups/{groupId}`
- `DELETE /api/groups/{groupId}`
- `GET /api/groups/{groupId}/accounts`
- `PUT /api/groups/{groupId}/accounts`
- `PUT /api/groups/{groupId}/scopes`
- `GET /api/grants`
- `GET /api/grants/tenants`
- `PUT /api/grants/{resourceType}/{resourceId}/{tenantId}`
- `DELETE /api/grants/{resourceType}/{resourceId}/{tenantId}`

Node management:
- `GET /api/nodes`
- `PATCH /api/nodes/{nodeId}`
- `DELETE /api/nodes/{nodeId}`
- `GET /api/nodes/{nodeId}/access/manage`
- `POST /api/nodes/{nodeId}/approve`
- `POST /api/nodes/{nodeId}/reject`
- `GET /api/nodes/transports`
- `POST /api/nodes/bootstrap/token`
- `GET /api/nodes/bootstrap/tokens/unconsumed`
- `DELETE /api/nodes/bootstrap/tokens/{tokenId}`
- `POST /api/nodes/enroll`
- `POST /api/nodes/exchange`
- `GET /api/nodes/pending`
- `GET /api/nodes/health`
- `GET /api/nodes/health/history`

Node agent:
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

Proxy and audit:
- `GET /api/proxy`
- `POST /api/proxy`
- `GET /api/proxy/{chainId}`
- `PATCH /api/proxy/{chainId}`
- `DELETE /api/proxy/{chainId}`
- `GET /api/proxy/{chainId}/probe`
- `POST /api/proxy/{chainId}/probe`
- `POST /api/proxy/validate`
- `POST /api/proxy/preview`
- `GET /api/proxy/scopes`
- `POST /api/proxy/scopes`
- `GET /api/proxy/scopes/{scopeId}`
- `PATCH /api/proxy/scopes/{scopeId}`
- `DELETE /api/proxy/scopes/{scopeId}`
- `GET /api/proxy/links`
- `POST /api/proxy/links`
- `PATCH /api/proxy/links/{linkId}`
- `DELETE /api/proxy/links/{linkId}`
- `GET /api/proxy/paths`
- `POST /api/proxy/paths`
- `PATCH /api/proxy/paths/{pathId}`
- `DELETE /api/proxy/paths/{pathId}`
- `GET /api/proxy/routes`
- `POST /api/proxy/routes`
- `GET /api/proxy/routes/{ruleId}`
- `PATCH /api/proxy/routes/{ruleId}`
- `DELETE /api/proxy/routes/{ruleId}`
- `POST /api/proxy/routes/validate`
- `GET /api/proxy/routes/suggestions`
- `GET /api/proxy/extension/bootstrap`
- `GET /api/proxy/extension/page/status`
- `GET /api/audit/proxy/sessions`
- `GET /api/audit/proxy/events`
- `GET /api/policies/revisions`
- `POST /api/policies/publish`

## Node Backend

Node local console:
- `POST /api/local/login`
- `POST /api/local/logout`
- `GET /api/local/session`
- `GET /api/local/status`
- `GET /api/local/health`
- `GET /api/local/audit`
- `GET /api/local/policy`
- `GET /api/local/diagnostics`

Node runtime:
- `GET /healthz`
- `POST /api/control/relay/probe`
- `POST /api/node/bootstrap/attach`
- `GET /api/node/tunnel/connect`

Node upstream proxy:
- `POST /api/nodes/enroll`
- `POST /api/nodes/exchange`
- `GET /api/node/agent/policy`
- `GET /api/node/agent/auth/validate`
- `POST /api/node/agent/heartbeat`
- `POST /api/node/agent/cert/renew`
- `POST /api/node/agent/transports`
- `POST /api/node/agent/proxy/sessions`
- `POST /api/node/agent/direct/candidates`
- `GET /api/node/agent/direct/link/plan`
- `POST /api/node/agent/direct/status`

## Frontend And Extension Requests

Panel web BFF:
- `/api/[...path]` -> panel `/api/{path}`
- `/api/audit/[...path]` -> panel `/api/audit/{path}`

Node web:
- `POST /api/local/login`
- `POST /api/local/logout`
- `GET /api/local/session`
- `GET /api/local/status`
- `GET /api/local/health`
- `GET /api/local/audit`
- `GET /api/local/policy`
- `GET /api/local/diagnostics`

Chrome extension:
- `POST /api/auth/login`
- `POST /api/auth/refresh`
- `POST /api/auth/logout`
- `GET /api/proxy/extension/bootstrap`
- `GET /api/proxy/extension/page/status`
- `POST /api/control/relay/probe`

CLI and VSCode extension:
- `POST /api/auth/login`
- `GET /api/proxy/extension/bootstrap`

## Applied Renames

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
- `/api/v1/[...path]` -> `/api/[...path]`
