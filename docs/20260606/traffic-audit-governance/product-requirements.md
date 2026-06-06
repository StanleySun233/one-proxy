# Product Requirements: Traffic Audit Governance

## Problem

OneProxy traffic governance is currently too loose from an operator perspective. Admins can configure routes and chains, but the audit surface does not clearly explain who sent traffic, what target was accessed, why a session was allowed or denied, and which policy version or route decision was responsible. This makes troubleshooting, accountability, and gradual policy tightening difficult.

## Goal

Build a traffic-first audit and governance experience in the panel. The first release must turn network audit sessions into an investigation surface: denied traffic, high-risk targets, actor and node attribution, route and policy evidence, and per-session detail.

## Non-Goals

- Do not add packet-level logging or payload inspection.
- Do not add custom query language.
- Do not add export, alerting, or ML anomaly detection in this release.
- Do not add a new audit_viewer role in this release.
- Do not change node enforcement semantics in this release.

## Requirements

1. Persist traffic decision evidence on every network audit session where available:
   - governance mode
   - policy revision
   - route rule id
   - route rule type
   - route rule pattern
   - chain id
   - scope id
   - entry node id
   - exit node id
   - normalized deny reason
2. Show traffic governance as the primary network audit experience.
3. Provide risk summary cards for:
   - total sessions
   - denied sessions
   - unique targets
   - total traffic
4. Provide focused views or presets for:
   - all sessions
   - denied traffic
   - high volume targets
5. Provide URL-backed filters for time range, actor, node, target host, route, chain, decision, and deny reason.
6. Network session rows must open a detail drawer with request, decision, path, and evidence sections.
7. Dashboard drill-down must let admins click denied traffic and top targets into the traffic audit page with filters applied.
8. The UI must remain dense and operational, consistent with the existing console design.

## Acceptance Criteria

1. Network audit rows expose policy revision, governance mode, rule evidence, and deny reason when present.
2. The traffic audit page can filter denied sessions without typing raw query strings.
3. Opening a session shows actor, source, target, decision, path, policy revision, matched rule, chain, scope, timing, and traffic bytes.
4. Dashboard denied and top target widgets link to prefiltered traffic audit queries.
5. Existing business audit remains functional.
6. Existing audit APIs remain compatible for current callers.
