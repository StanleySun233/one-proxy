# Todo List

## Current Status

- [x] repo split into `apps`, `docs`, and `prototypes`
- [x] Python demo moved out of the main product path
- [x] initial product architecture documented
- [x] documentation split into numbered design files
- [x] backend, frontend, extension, and node-agent design docs split
- [x] Go control-plane skeleton created
- [x] Next.js control-plane dashboard skeleton created
- [x] Chrome extension server-backed state contract drafted
- [ ] commit current repo restructuring

## Phase 1

- [ ] freeze V1 scope
- [ ] review and adjust route rule model
- [ ] review and adjust `destination_scope` model
- [ ] review and adjust SQLite DDL
- [ ] review and adjust API contract

## Phase 2

- [ ] initialize Go module under `apps/control-plane-api`
- [ ] scaffold Go control-plane code layout
- [ ] define API directory structure
- [ ] implement bootstrap admin flow
- [ ] implement account login and session flow
- [ ] implement node CRUD
- [ ] implement chain CRUD
- [ ] implement route rule CRUD
- [ ] implement policy publish flow

## Phase 3

- [ ] define Go node agent project layout
- [ ] implement node enrollment handshake
- [ ] implement heartbeat
- [ ] implement policy sync
- [ ] implement CONNECT-based chain forwarding
- [ ] implement public cert renewal flow

## Phase 4

- [ ] initialize Next.js app under `apps/control-plane-web`
- [ ] build design-system-based dashboard skeleton
- [ ] implement login page
- [ ] implement node list and detail pages
- [ ] implement chain editor
- [ ] implement route rule editor
- [ ] implement logs and health views

## Phase 5

- [ ] define Chrome extension login flow
- [ ] redesign extension state model around server-backed profiles
- [ ] define extension storage model
- [ ] implement node selection
- [ ] implement profile sync
- [ ] implement status display
