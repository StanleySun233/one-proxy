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
- [x] control-plane implementation docs added
- [x] read-only Go control-plane APIs added
- [x] seed login API added for frontend integration shaping
- [x] SQLite-backed store skeleton added
- [x] create APIs for nodes, chains, and route rules added
- [x] refresh/logout, bootstrap token, enroll, and policy publish APIs added
- [x] node-agent sync APIs added
- [x] control-plane CRUD and sync route set completed
- [x] node link and certificate APIs added
- [x] API response envelope standardized
- [x] control-plane Bearer auth added
- [x] node-agent Bearer auth added
- [x] Go node-agent skeleton added
- [x] standalone `apps/proxy-node` extracted from control plane
- [x] scheduler skeleton added
- [x] node onboarding requirement doc added
- [x] node onboarding implementation doc added
- [x] node onboarding API doc added
- [ ] commit current repo restructuring

## Phase 1

- [x] freeze V1 scope
- [x] review and adjust route rule model
- [x] review and adjust `destination_scope` model
- [x] review and adjust SQLite DDL
- [x] review and adjust API contract

## Phase 2

- [x] initialize Go module under `apps/control-plane-api`
- [x] scaffold Go control-plane code layout
- [x] define API directory structure
- [x] replace seed store with SQLite-backed store
- [x] implement bootstrap admin flow
- [x] implement account login and session flow
- [x] complete node CRUD
- [x] complete chain CRUD
- [x] complete route rule CRUD
- [x] implement policy publish flow
- [x] implement node agent policy pull
- [x] implement node agent heartbeat
- [x] implement node agent cert renew

## Phase 3

- [x] define Go node agent project layout
- [x] implement node enrollment handshake
- [x] implement enrollment approval flow
- [x] implement heartbeat
- [x] implement policy sync
- [x] implement CONNECT-based chain forwarding
- [x] persist node local policy state
- [x] support CIDR and protocol route matching
- [ ] implement public cert renewal flow
- [x] implement node access path CRUD
- [x] implement panel onboarding task APIs
- [x] implement node upstream-pull forwarding via existing online node
- [x] implement direct onboarding task reachability probe
- [x] implement relay-chain onboarding executor

## Phase 4

 - [x] initialize Next.js app under `apps/control-plane-web`
- [x] build design-system-based dashboard skeleton
- [x] define frontend tech stack
- [x] define frontend information architecture
- [x] add `next-intl` locale routing skeleton
- [x] add `next-themes` and token-based theme shell
- [x] add React Query provider shell
- [x] add left navigation and overview page skeleton
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

## Phase 6

- [x] define reverse tunnel requirements
- [x] define reverse tunnel architecture
- [x] add transport-aware node runtime model
- [x] add transport list API
- [x] add chain probe API with blocking-hop diagnostics
- [x] add parent-child reverse ws tunnel in proxy node
- [x] add tunnel-backed multi-hop probe execution
- [x] add tunnel-backed data stream forwarding
