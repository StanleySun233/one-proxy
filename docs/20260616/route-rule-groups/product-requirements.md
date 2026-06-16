# Product Requirements: Route Rule Groups

**Date:** 20260616

## Requirement

Create route rule groups. A route rule group contains multiple route rules, and tenant permission grants for routing must be based on the route rule group rather than individual route rules.

## Operating Constraints

- No backward compatibility with the old route-rule grant model is required.
- The camelbot production database can be updated directly with SQL after the new code path is implemented and verified.
- Existing production data should be transformed into the new model without keeping the old runtime authorization path.

## Current Production Shape

- `route_rules`: 7 rows.
- `tenant_route_rules`: 15 rows.
- Current route rule bindings are distributed across tenant IDs `1`, `2`, and `3`.

## Target Model

- `route_rule_groups` is the grantable routing container.
- `route_rules` belongs to a route rule group through `group_id`.
- `tenant_route_rule_groups` replaces `tenant_route_rules`.
- Policy compilation, extension bootstrap, route visibility, create, edit, delete, and grants all use route rule group permissions.
