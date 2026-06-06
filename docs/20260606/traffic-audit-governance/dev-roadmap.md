# Dev Roadmap: Traffic Audit Governance

**Date:** 20260606
**Status:** implementation complete, deployment pending
**Product document:** ./product-requirements.md

## Summary

Build a traffic-first audit and governance experience in the panel. The first release extends network audit sessions with policy evidence and upgrades the network audit page into a traffic investigation surface focused on denied traffic, target risk, actor attribution, and per-session detail.

## Tasks

- [x] backend-audit: extend network audit domain, schema, MySQL store, and audit query parsing with governance evidence fields
- [x] backend-audit: enrich proxy session ingest with policy revision and route evidence available at ingest time
- [x] frontend-audit: extend panel web audit types and API query parameters
- [x] frontend-audit: redesign network audit page as traffic governance workbench with presets, summary cards, URL filters, and detail modal
- [x] frontend-audit: add audit traffic i18n strings and supporting styles
- [x] test-audit: run Go tests for panel API and Node/TypeScript checks for panel web

## Verification

- `go test ./...` in `apps/panel/api`
- `go test ./...` in `apps/node/api`
- `tsc --project <temporary tsconfig excluding .next>` in `apps/panel/web`
- Independent review findings addressed:
  - locale-aware audit drilldown links
  - route-not-found and local-proxy fallback audit reporting
  - node-origin policy and rule evidence
  - upgrade indexes for new audit filters
  - Suspense boundary for URL-backed filters
  - WebSocket upgrade audit reporting

The repository `apps/panel/web/tsconfig.json` still includes stale `.next/types` entries that reference removed routes, so raw `tsc --noEmit` fails before checking source files.

## Constraints

- Existing business audit must remain functional.
- Existing audit API callers must remain compatible.
- Do not add packet-level logging, content inspection, custom query language, exports, or alerting.
- Do not change node enforcement semantics in this release.
