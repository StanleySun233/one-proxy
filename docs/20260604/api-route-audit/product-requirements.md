# Product Requirements: API Route Audit

## Scope

Audit and standardize API route naming across the panel backend API, node backend API, panel web API client/proxy, node web, and Chrome extension.

## Requirements

- List all backend API routes.
- List all frontend-requested API routes, including panel web, node web, and Chrome extension.
- Check whether route naming is unified.
- Enforce the rule that each route path segment should be a single word. Dashed route segments such as `/aaa-bbb/` are not allowed unless explicitly justified.
- Produce an implementation checklist and wait for user approval before changing code.
- Treat panel API, node API, panel web, node web, and extension as separate write scopes during implementation.

## Current Constraint

The repository currently has staged and unstaged work from the node-local console and tenant/resource sharing changes. The audit must avoid reverting existing work and should plan changes on top of the current workspace state.
