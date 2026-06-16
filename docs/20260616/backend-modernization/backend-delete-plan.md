# Backend Progress: delete-plan

**Engineer:** delete-plan
**Scope:** Build the shared DeletePlan model and executor.

## Tasks

- [ ] Create `apps/panel/api/internal/store/deleteplan/plan.go` with DeletePlan, DeletePlanStep, DeleteImpactItem, and execution result types.
  - Commit:
- [ ] Create `apps/panel/api/internal/store/deleteplan/mysql_executor.go` to execute DeletePlan steps inside one SQL transaction.
  - Commit:
- [ ] Create `apps/panel/api/internal/store/deleteplan/mysql_executor_test.go` covering ordered execution, rollback, and affected-row reporting.
  - Commit:

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
