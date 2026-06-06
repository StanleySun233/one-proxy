# API Contract: Traffic Audit Governance

## Scope

This contract extends the existing audit API without removing existing fields or endpoints.

## Extended Network Session Shape

`NetworkAuditSession` keeps all existing fields and adds:

```json
{
  "governanceMode": "enforce",
  "policyRevision": "5",
  "matchedRuleId": "3",
  "matchedRuleType": "domain_suffix",
  "matchedRulePattern": "openai.com",
  "matchedAction": "chain",
  "decisionSource": "policy"
}
```

Field meanings:

- `governanceMode`: `monitor`, `enforce`, or `shadow`. The first release writes `enforce` for node-enforced sessions.
- `policyRevision`: policy revision used by the node or panel at ingest time.
- `matchedRuleId`: the route rule responsible for the decision.
- `matchedRuleType`: rule match type such as `domain`, `domain_suffix`, `ip_cidr`, or `default`.
- `matchedRulePattern`: normalized rule pattern or target pattern.
- `matchedAction`: route action such as `direct`, `chain`, or `deny`.
- `decisionSource`: `policy`, `default`, `auth`, or `unknown`.

## List Network Sessions

`GET /api/audit/network/sessions`

Existing query parameters remain valid. New optional query parameters:

- `denyReason`
- `policyRevision`
- `matchedRuleId`
- `decisionSource`

Response:

```json
{
  "items": [],
  "summary": {
    "total": 0,
    "bytesIn": 0,
    "bytesOut": 0,
    "durationAvgMs": 0,
    "decisionCount": {},
    "denyReasonCount": {},
    "topTargets": [],
    "userTraffic": [],
    "nodeTraffic": [],
    "tenantTraffic": [],
    "recentBusinessEvents": []
  }
}
```

## Dashboard

`GET /api/audit/dashboard`

The dashboard response includes the extended network summary fields, especially `denyReasonCount`.

## Backward Compatibility

Existing clients that ignore unknown fields continue to work. Existing query parameters and response fields are unchanged.
