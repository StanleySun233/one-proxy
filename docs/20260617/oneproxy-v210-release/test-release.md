# Test Progress: test-release

**Engineer:** test-release
**Scope:** Compile checks, local Docker scenario, camelbot isolated scenario, GitHub Actions image verification, replacement deployment, database-backed user-flow tests, and release tag.

## Tasks

- [ ] Add local v2.1.0 Docker scenario runner in `scripts/test-v210-docker-scenario.sh`
  - Commit:
- [ ] Add camelbot isolated scenario runner in `scripts/test-camelbot-v210-scenario.sh`
  - Commit:
- [ ] Extend GitHub Actions image workflows for immutable pre-tag test images in `.github/workflows/one-proxy-node-image.yml`
  - Commit:
- [ ] Extend GitHub Actions image workflows for immutable pre-tag test images in `.github/workflows/one-proxy-panel-image.yml`
  - Commit:
- [ ] Add release deployment script for local node, camelbot node, and camelbot panel in `scripts/deploy-v210-release-images.sh`
  - Commit:
- [ ] Document database-backed real-user verification in `docs/20260617/oneproxy-v210-release/release-test-plan.md`
  - Commit:
- [ ] Run compile, unit, extension smoke, local Docker scenario, camelbot isolated scenario, replacement deployment, DB queries, and real-user functional tests
  - Evidence:
- [ ] Create and push tag `v2.1.0` after all gates pass
  - Evidence:

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
