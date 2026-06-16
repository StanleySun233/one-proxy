# API Contract: Route Rule Groups

## Resource Model

`route_rule_group` is the grantable routing resource. A route rule belongs to exactly one route rule group through `groupId`.

## Route Rule Group

```json
{
  "id": "route-rule-group-1",
  "name": "Public HK",
  "description": "Public routing rules",
  "enabled": true,
  "createId": "account-1",
  "ownerId": "account-1",
  "createdAt": "2026-06-16T00:00:00Z",
  "updatedAt": "2026-06-16T00:00:00Z",
  "permission": "manage",
  "ruleCount": 3
}
```

## Route Rule

```json
{
  "id": "route-rule-1",
  "groupId": "route-rule-group-1",
  "createId": "account-1",
  "ownerId": "account-1",
  "priority": 100,
  "matchType": "domain",
  "matchValue": "example.com",
  "actionType": "chain",
  "chainId": "chain-1",
  "destinationScope": "scope-1",
  "enabled": true,
  "permission": "manage"
}
```

`permission` on a route rule is inherited from the parent group.

## Endpoints

### `GET /api/proxy/route-groups`

Returns route rule groups visible to the active tenant. Super admin without an active tenant receives all groups.

Response: `RouteRuleGroup[]`

### `POST /api/proxy/route-groups`

Requires active tenant admin or super admin with an active tenant.

Request:

```json
{
  "name": "Public HK",
  "description": "Public routing rules"
}
```

Response: `201 RouteRuleGroup`

The active tenant receives `manage` permission on the new group.

### `PATCH /api/proxy/route-groups/{groupId}`

Requires `manage` on the group.

Request:

```json
{
  "name": "Public HK",
  "description": "Public routing rules",
  "enabled": true
}
```

Response: `200 RouteRuleGroup`

### `GET /api/proxy/route-groups/{groupId}/delete-impact`

Requires `manage` on the group.

Response:

```json
{
  "groupId": "route-rule-group-1",
  "delete": {
    "group": [],
    "routeRules": [],
    "tenantBindings": []
  }
}
```

### `DELETE /api/proxy/route-groups/{groupId}`

Requires `manage` on the group.

Deletes tenant group bindings, contained route rules, and the group through DeletePlan.

Response:

```json
{"status": "deleted"}
```

### `GET /api/proxy/routes`

Returns rules whose parent group is visible to the active tenant. Rules in disabled groups are returned for management, but policy compilation and extension bootstrap ignore disabled groups.

Response: `RouteRule[]`

### `POST /api/proxy/routes`

Requires `manage` on `groupId`.

Request:

```json
{
  "groupId": "route-rule-group-1",
  "priority": 100,
  "matchType": "domain",
  "matchValue": "example.com",
  "actionType": "chain",
  "chainId": "chain-1",
  "destinationScope": "scope-1"
}
```

Response: `201 RouteRule`

### `PATCH /api/proxy/routes/{ruleId}`

Requires `manage` on the rule's parent group. `groupId` can move the rule to another group if the caller has `manage` on both the current and target groups.

Request:

```json
{
  "groupId": "route-rule-group-2",
  "priority": 100,
  "matchType": "domain",
  "matchValue": "example.com",
  "actionType": "chain",
  "chainId": "chain-1",
  "destinationScope": "scope-1",
  "enabled": true
}
```

Response: `200 RouteRule`

### `DELETE /api/proxy/routes/{ruleId}`

Requires `manage` on the rule's parent group.

Response:

```json
{"status": "deleted"}
```

### `POST /api/proxy/routes/validate`

Accepts optional `groupId` for priority conflict checks scoped to that group.

## Grants

`ResourceBindingType` replaces `route_rule` with `route_rule_group`.

Grant endpoints stay unchanged:

- `GET /api/grants?resourceType=route_rule_group&resourceId={groupId}`
- `PUT /api/grants/route_rule_group/{groupId}/{tenantId}`
- `DELETE /api/grants/route_rule_group/{groupId}/{tenantId}`

## Production SQL Migration

Production data is converted by creating route groups from the existing tenant binding signatures, assigning each rule to one generated group, creating `tenant_route_rule_groups`, making `route_rules.group_id` required, and dropping `tenant_route_rules`.
