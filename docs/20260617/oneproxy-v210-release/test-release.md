# Test Progress: test-release

**Engineer:** test-release
**Scope:** Compile checks, local Docker scenario, camelbot isolated scenario, GitHub Actions image verification, final-schema evidence, replacement deployment decision, database-backed user-flow tests, and release tag.

## Tasks

- [x] Add local v2.1.0 Docker scenario runner in `scripts/test-v210-docker-scenario.sh`
  - Commit: 3df6698
- [x] Add camelbot isolated scenario runner in `scripts/test-camelbot-v210-scenario.sh`
  - Commit: 3df6698
- [x] Upgrade local and camelbot scenario runners to create latest control-plane state, enroll and approve an edge node, validate latest extension bootstrap, validate hashed proxy-token authorization, and query hash-shaped token storage
  - Commit: 58ab36a
- [x] Extend GitHub Actions image workflows for immutable pre-tag test images in `.github/workflows/one-proxy-node-image.yml`
  - Commit: 3df6698
- [x] Extend GitHub Actions image workflows for immutable pre-tag test images in `.github/workflows/one-proxy-panel-image.yml`
  - Commit: 3df6698
- [x] Add release deployment script for local node, camelbot node, and camelbot panel in `scripts/deploy-v210-release-images.sh`
  - Commit: 3df6698
- [x] Document database-backed real-user verification in `docs/20260617/oneproxy-v210-release/release-test-plan.md`
  - Commit: 7679cab
- [x] Verify final-schema-only panel delivery with no goose runtime and no numbered SQL migration chain
  - Commit: 65411e7
  - Evidence:
    - `apps/panel/api/schema/final.sql` is the only panel API schema file.
    - `apps/panel/api/migrations` has been removed.
    - Panel API initializes the schema only when the target database has zero tables.
    - A source scan found no panel API/web/Docker references to `goose`, `goose_db_version`, numbered SQL migrations, removed old access-path enums, old SOCKS-style aliases, or old raw TCP/UDP service names.
- [x] Add final-schema deployment guard to the camelbot panel replacement script
  - Evidence:
    - `bash -n scripts/deploy-v210-release-images.sh`: pass
    - `scripts/deploy-v210-release-images.sh dry-run camelbot-panel v2.1.0-rc.65411e7`: pass; prints required final DB and final volume
    - `scripts/deploy-v210-release-images.sh deploy camelbot-panel v2.1.0-rc.65411e7`: rejects without `ONEPROXY_FINAL_PANEL_DB_NAME`
    - `scripts/deploy-v210-release-images.sh check camelbot-panel v2.1.0-rc.65411e7`: pass; standing DB evidence `tables=40 goose_tables=1 access_paths=0`
- [x] Add final-schema standing cutover script for fresh panel DB and fresh node runtime bindings
  - Evidence:
    - `bash -n scripts/deploy-v210-final-cutover.sh`: pass
    - `scripts/deploy-v210-final-cutover.sh dry-run v2.1.0-rc.65411e7`: pass; prints final panel DB, fresh panel volume, fresh remote node volume, fresh local node volume, and target node names
    - The same dry-run prints `remote_node_parent_url=http://one-proxy-panel:2886` and infers `local_node_parent_url=http://103.214.172.211:2886`
    - `scripts/deploy-v210-final-cutover.sh dry-run latest`: rejects mutable image tag
    - `scripts/deploy-v210-final-cutover.sh check`: pass; local node, camelbot panel, and camelbot node are still on `v2.1.0-rc.38d3a66`; target final DB `one_proxy_v210_final` has `tables=0`
    - `scripts/deploy-v210-final-cutover.sh run v2.1.0-rc.65411e7`: rejects without `ONEPROXY_FINAL_SCHEMA_CONFIRM=deploy-final-schema`
    - Cutover rollback logic restores renamed backup containers if panel, remote-node, local-node, or final control-plane provisioning fails before completion
- [x] Add final-schema post-cutover verification mode
  - Evidence:
    - `bash -n scripts/deploy-v210-final-cutover.sh`: pass
    - `scripts/deploy-v210-final-cutover.sh verify latest`: rejects mutable image tag
    - `scripts/deploy-v210-final-cutover.sh verify <immutable_tag>` is read-only and checks local node, camelbot panel, camelbot node, final DB table shape, no `goose_db_version`, access paths, routes, policy revisions, and token hash shapes
- [x] Add post-manual-setup node bootstrap and evidence script
  - Evidence:
    - `bash -n scripts/deploy-v210-post-setup-nodes.sh`: pass
    - `scripts/deploy-v210-post-setup-nodes.sh dry-run v2.1.0-rc.65411e7`: pass; prints final node image and local/remote parent URLs
    - `scripts/deploy-v210-post-setup-nodes.sh check`: pass; panel is running final image in setup mode with `configured=false`; remote and local node containers are missing after reset
    - `scripts/deploy-v210-post-setup-nodes.sh run v2.1.0-rc.65411e7`: rejects without `ONEPROXY_POST_SETUP_CONFIRM=bootstrap-nodes`
- [x] Reset standing deployment for manual final-schema setup testing
  - Evidence:
    - User requested a full reset and deletion of old configuration so setup can be tested from database creation.
    - Remote standing panel runs `ghcr.io/stanleysun233/oneproxy-panel:v2.1.0-rc.65411e7`.
    - Public `/api/setup/status` returns `configured=false`.
    - Public `/healthz` returns `mode=setup`.
    - Remote node container is missing after reset.
    - Local node container is missing after reset.
- [x] Replace standing panel, remote edge node, and local relay node after manual setup
  - Evidence:
    - User completed manual panel setup against database `one_proxy`.
    - Commit: 3cf4562
    - Panel image workflow: pass, https://github.com/StanleySun233/one-proxy/actions/runs/27682252948
    - Node image workflow: pass, https://github.com/StanleySun233/one-proxy/actions/runs/27682252778
    - Immutable post-UX pre-tag image: `v2.1.0-rc.3cf4562`
    - `ghcr.io/stanleysun233/oneproxy-panel:v2.1.0-rc.3cf4562`: `sha256:ae936f12bd415f7cf264a4162706f739595f43bd0c53185f9f4bd030baaa6f83`
    - `ghcr.io/stanleysun233/oneproxy-panel-base:v2.1.0-rc.3cf4562`: `sha256:e52c16867a20c16c0840270a33843466dfbf30975610477ff8bc8baa8ff7e964`
    - `ghcr.io/stanleysun233/oneproxy-node:v2.1.0-rc.3cf4562`: `sha256:e5c1797e65f021dad46c1550bf2b9ca4d5a14c70748008109266c907c80bd18b`
    - `ghcr.io/stanleysun233/oneproxy-node-base:v2.1.0-rc.3cf4562`: `sha256:ec661e4ae6b2087cbe8cb1a9d8926792c580a4bbb381af6012a9b2d697d8a295`
    - Remote panel container image is `ghcr.io/stanleysun233/oneproxy-panel:v2.1.0-rc.3cf4562`; `/healthz` returns `status=ok`.
    - Remote `camelbot` edge node image is `ghcr.io/stanleysun233/oneproxy-node:v2.1.0-rc.3cf4562`; `/healthz` returns `controlPlaneBound=true`.
    - Local `astar-58` relay node image is `ghcr.io/stanleysun233/oneproxy-node:v2.1.0-rc.3cf4562`; `/healthz` returns `controlPlaneBound=true`.
    - DB nodes: `camelbot edge healthy public_host=103.214.172.211 public_port=2988`; `astar-58 relay healthy parent_node_id=1`.
    - DB health snapshots: nodes `1` and `3` report `{"runtime":"up"}`.
    - DB transport evidence: node `3` has `reverse_ws_parent` connected to `ws://103.214.172.211:2988/api/node/tunnel/connect?parentNodeId=1`.
    - Parent URL probe for `http://103.214.172.211:2988` returns `reachable=true`, `statusCode=200`, `mode=proxy-node`, and `controlPlaneBound=true`.
    - Deployed panel static assets contain `ghcr.io/stanleysun233/oneproxy-node:v2.1.0-rc.3cf4562` and no `oneproxy-node:latest` reference.
- [x] Run compile, unit, extension smoke, local Docker scenario, camelbot isolated scenario, image workflows, and isolated DB evidence
  - Evidence:
    - `bash -n scripts/test-v210-docker-scenario.sh scripts/test-camelbot-v210-scenario.sh scripts/deploy-v210-release-images.sh`: pass
    - `git diff --check`: pass
    - `scripts/test-v210-docker-scenario.sh check`: pass; no containers started
    - `scripts/test-camelbot-v210-scenario.sh check`: pass through SSH; no containers started; remote Docker, Compose, curl, and Python 3 available
    - `scripts/deploy-v210-release-images.sh dry-run all v2.1.0-rc.local`: pass; no deployment performed
    - `cd apps/node/api && go test ./...`: pass
    - `cd apps/panel/api && go test ./...`: pass after `58ab36a`
    - `cd apps/extension/cli && go test ./internal/proxycommand`: pass
    - `cd apps/cli && node ../panel/web/node_modules/typescript/bin/tsc -p tsconfig.json --noEmit`: pass
    - `node --test apps/extension/chrome/tools/domain_suffix_test.mjs`: pass, 3 tests
    - `node apps/extension/chrome/tools/validate_extension.mjs`: pass, `chrome_extension_static_ok`
    - `cd apps/panel/web && node node_modules/typescript/bin/tsc --noEmit --pretty false --incremental false`: fail; `.next/types/validator.ts` still references the removed duplicate audit route
    - Pushed commit: `e3de1ed56bef` to `origin/main`
    - GitHub Actions node image workflow: pass, https://github.com/StanleySun233/one-proxy/actions/runs/27666696189
    - GitHub Actions panel image workflow: pass, https://github.com/StanleySun233/one-proxy/actions/runs/27666696137
    - Immutable pre-tag image: `v2.1.0-rc.e3de1ed56bef`
    - `ghcr.io/stanleysun233/oneproxy-node-base:v2.1.0-rc.e3de1ed56bef`: `sha256:7bca41bcd0b672a0d2c29a388facbd6cb8800f91bc13bb664d06926a8d2c0936`
    - `ghcr.io/stanleysun233/oneproxy-node:v2.1.0-rc.e3de1ed56bef`: `sha256:d50310dca5c012123e8c6af82b3db3c0147a175abdc749801650806cd45908c9`
    - `ghcr.io/stanleysun233/oneproxy-panel-base:v2.1.0-rc.e3de1ed56bef`: `sha256:c5e7acbc1bf418de69d583a57d51a77c639bced5fc935a24af0a803f6e7d19bf`
    - `ghcr.io/stanleysun233/oneproxy-panel:v2.1.0-rc.e3de1ed56bef`: `sha256:323fe5dbb95d579b0be01562cee91c8c652deda1e0e61e323fc98d2a74cc21d0`
    - `shellcheck`: not run; command not installed
    - Final runtime commit: `65411e7`
    - `cd apps/panel/api && go test ./...`: pass after final schema baseline
    - Panel web source TypeScript check excluding stale `.next`: pass after final schema baseline
    - `node --test apps/cli/test/*.mjs`: pass, 50 tests
    - `node apps/extension/chrome/tools/domain_suffix_test.mjs`: pass
    - `node apps/extension/chrome/tools/validate_extension.mjs`: pass, `chrome_extension_static_ok`
    - `node apps/extension/chrome/tools/build_background_bundle.mjs`: pass
    - GitHub Actions panel image workflow: pass, https://github.com/StanleySun233/one-proxy/actions/runs/27671854441
    - GitHub Actions node image workflow: pass, https://github.com/StanleySun233/one-proxy/actions/runs/27671854468
    - Immutable final-schema pre-tag image: `v2.1.0-rc.65411e7`
    - `ghcr.io/stanleysun233/oneproxy-panel:v2.1.0-rc.65411e7`: `sha256:4803a7bf3a1b94be9a0e951c4ff3c13a596d222ce5ab73d0f49fd30c7c9e7d4a`
    - `ghcr.io/stanleysun233/oneproxy-panel-base:v2.1.0-rc.65411e7`: `sha256:dfa63195389d6c82e9152121d43a69d76eacec55525463ff0de592ea2aa00d32`
    - `ghcr.io/stanleysun233/oneproxy-node:v2.1.0-rc.65411e7`: `sha256:0a19998dd5011f0a065889fad1a7dba96bbb1542778514e136af6ffdb369f567`
    - `ghcr.io/stanleysun233/oneproxy-node-base:v2.1.0-rc.65411e7`: `sha256:62fd4e5b070b1fbc2391dd05eb61b0844ded749d2d60b2b7dd727e5a707fe8ce`
    - Local isolated scenario with `ONEPROXY_IMAGE_TAG=v2.1.0-rc.65411e7 scripts/test-v210-docker-scenario.sh run`: pass; bootstrap `schema=v2.1.0 access_paths=1 routes=1`; proxy-token validation ok; session, node-token, and bootstrap-token hashes verified
    - Camelbot isolated scenario with `ONEPROXY_IMAGE_TAG=v2.1.0-rc.65411e7 ONEPROXY_MYSQL_IMAGE=mysql:8.0 ONEPROXY_REDIS_IMAGE=redis:7-alpine scripts/test-camelbot-v210-scenario.sh run`: pass; bootstrap `schema=v2.1.0 access_paths=1 routes=1`; proxy-token validation ok; session, node-token, and bootstrap-token hashes verified
    - Standing replacement deployment against an old non-empty panel database: intentionally not performed after final-schema-only change because the final release does not include old-version migration compatibility
    - Standing reset: complete; panel now runs the final image in setup mode and waits for manual database setup before node bootstrap
- [ ] Create and push tag `v2.1.0` after all final deployment gates pass
  - Evidence:

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
| 2026-06-17 | Raw panel web `tsc --noEmit` includes stale `.next/types/validator.ts` generated output that references a removed duplicate audit route. | Source TypeScript check passes with `.next` excluded; generated cache must be refreshed before using raw `.next` as a release artifact. |
| 2026-06-17 | Existing standing panel database was non-empty, had old migration history, and did not contain final access-path rows. | Reset complete; do not use old database compatibility. |
| 2026-06-17 | Standing replacement needed explicit destructive-operation authorization. | User authorized full reset; reset complete. |
| 2026-06-17 | Panel setup is intentionally manual for database-creation testing. | Setup completed by user; standing panel, remote edge, and local relay replacement verified. |
