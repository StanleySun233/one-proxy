package store

import (
	"context"
	"database/sql/driver"
	"reflect"
	"testing"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/store/deleteplan"
)

func TestDeleteChainDeletesRelationshipsBeforeChain(t *testing.T) {
	record := &nodeDeleteRecord{}
	db := openNodeDeleteTestDB(t, record)
	defer db.Close()

	store := &MySQLStore{db: db}
	plan, err := store.proxyRepository().buildChainDeletePlan(context.Background(), "chain-1", false)
	if err != nil {
		t.Fatalf("buildChainDeletePlan: %v", err)
	}
	if err := store.DeleteChain("chain-1"); err != nil {
		t.Fatalf("DeleteChain: %v", err)
	}

	args := []driver.Value{"chain-1"}
	want := []nodeDeleteCall{
		{Query: "BEGIN"},
		{Query: "DELETE FROM route_rules WHERE chain_id = ?", Args: args},
		{Query: "DELETE FROM node_onboarding_tasks WHERE path_id IN (SELECT id FROM node_access_paths WHERE chain_id = ?)", Args: args},
		{Query: "DELETE FROM tenant_access_paths WHERE access_path_id IN (SELECT id FROM node_access_paths WHERE chain_id = ?)", Args: args},
		{Query: "DELETE FROM node_access_paths WHERE chain_id = ?", Args: args},
		{Query: "DELETE FROM chain_probe_results WHERE chain_id = ?", Args: args},
		{Query: "DELETE FROM tenant_chains WHERE chain_id = ?", Args: args},
		{Query: "DELETE FROM chain_hops WHERE chain_id = ?", Args: args},
		{Query: "DELETE FROM chains WHERE id = ?", Args: args},
		{Query: "COMMIT"},
	}
	if got := record.snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %#v, want %#v", got, want)
	}
	if got := deleteCallsFromPlan(plan); !reflect.DeepEqual(got, want[1:len(want)-1]) {
		t.Fatalf("plan calls = %#v, want %#v", got, want[1:len(want)-1])
	}
}

func TestDeleteNodeAccessPathDeletesRelationshipsBeforePath(t *testing.T) {
	record := &nodeDeleteRecord{}
	db := openNodeDeleteTestDB(t, record)
	defer db.Close()

	store := &MySQLStore{db: db}
	plan, err := store.proxyRepository().buildNodeAccessPathDeletePlan(context.Background(), "path-1", false)
	if err != nil {
		t.Fatalf("buildNodeAccessPathDeletePlan: %v", err)
	}
	if err := store.DeleteNodeAccessPath("path-1"); err != nil {
		t.Fatalf("DeleteNodeAccessPath: %v", err)
	}

	args := []driver.Value{"path-1"}
	want := []nodeDeleteCall{
		{Query: "BEGIN"},
		{Query: "DELETE FROM node_onboarding_tasks WHERE path_id = ?", Args: args},
		{Query: "DELETE FROM tenant_access_paths WHERE access_path_id = ?", Args: args},
		{Query: "DELETE FROM node_access_paths WHERE id = ?", Args: args},
		{Query: "COMMIT"},
	}
	if got := record.snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %#v, want %#v", got, want)
	}
	if got := deleteCallsFromPlan(plan); !reflect.DeepEqual(got, want[1:len(want)-1]) {
		t.Fatalf("plan calls = %#v, want %#v", got, want[1:len(want)-1])
	}
}

func TestDeletePlanPreviewAndExecutionStepsMatch(t *testing.T) {
	record := &nodeDeleteRecord{}
	db := openNodeDeleteTestDB(t, record)
	defer db.Close()

	store := &MySQLStore{db: db}
	chainExecutionPlan, err := store.proxyRepository().buildChainDeletePlan(context.Background(), "chain-1", false)
	if err != nil {
		t.Fatalf("build execution chain plan: %v", err)
	}
	chainPreviewPlan, err := store.proxyRepository().buildChainDeletePlan(context.Background(), "chain-1", true)
	if err != nil {
		t.Fatalf("build preview chain plan: %v", err)
	}
	if !reflect.DeepEqual(stripExpectedImpact(chainPreviewPlan.Steps), chainExecutionPlan.Steps) {
		t.Fatalf("chain preview steps = %#v, execution steps = %#v", stripExpectedImpact(chainPreviewPlan.Steps), chainExecutionPlan.Steps)
	}

	pathExecutionPlan, err := store.proxyRepository().buildNodeAccessPathDeletePlan(context.Background(), "path-1", false)
	if err != nil {
		t.Fatalf("build execution path plan: %v", err)
	}
	pathPreviewPlan, err := store.proxyRepository().buildNodeAccessPathDeletePlan(context.Background(), "path-1", true)
	if err != nil {
		t.Fatalf("build preview path plan: %v", err)
	}
	if !reflect.DeepEqual(stripExpectedImpact(pathPreviewPlan.Steps), pathExecutionPlan.Steps) {
		t.Fatalf("path preview steps = %#v, execution steps = %#v", stripExpectedImpact(pathPreviewPlan.Steps), pathExecutionPlan.Steps)
	}
}

func TestDeleteTenantClearsPolicyReferencesBeforeTenant(t *testing.T) {
	record := &nodeDeleteRecord{}
	db := openNodeDeleteTestDB(t, record)
	defer db.Close()

	store := &MySQLStore{db: db}
	if err := store.DeleteTenant("tenant-1"); err != nil {
		t.Fatalf("DeleteTenant: %v", err)
	}

	args := []driver.Value{"tenant-1"}
	want := []nodeDeleteCall{
		{Query: "BEGIN"},
		{Query: "UPDATE node_health_snapshots SET policy_revision_id = NULL WHERE policy_revision_id IN (SELECT id FROM policy_revisions WHERE tenant_id = ?)", Args: args},
		{Query: "DELETE FROM node_policy_assignments WHERE tenant_id = ?", Args: args},
		{Query: "DELETE FROM policy_revisions WHERE tenant_id = ?", Args: args},
		{Query: "DELETE FROM tenants WHERE id = ?", Args: args},
		{Query: "COMMIT"},
	}
	if got := record.snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %#v, want %#v", got, want)
	}
}

func TestDeleteAccountReassignsReferencesBeforeAccount(t *testing.T) {
	accountQuery := `SELECT a.id, a.account, r.name, a.status, a.must_rotate_password
		 FROM accounts a
		 JOIN roles r ON r.id = a.role_id
		 WHERE a.id = ?`
	replacementQuery := `SELECT a.id
		 FROM accounts a
		 JOIN roles r ON r.id = a.role_id
		 WHERE a.id <> ? AND a.status = ? AND r.name = ?
		 ORDER BY a.id
		 LIMIT 1`
	record := &nodeDeleteRecord{
		results: map[string]nodeDeleteQueryResult{
			normalizedQuery(accountQuery): {
				columns: []string{"id", "account", "role", "status", "must_rotate_password"},
				values:  [][]driver.Value{{"account-1", "alice", domain.AccountRoleUser, domain.AccountStatusActive, int64(0)}},
			},
			normalizedQuery(replacementQuery): {
				columns: []string{"id"},
				values:  [][]driver.Value{{"account-2"}},
			},
		},
	}
	db := openNodeDeleteTestDB(t, record)
	defer db.Close()

	store := &MySQLStore{db: db}
	if err := store.DeleteAccount("account-1"); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}

	deleteArgs := []driver.Value{"account-1"}
	reassignArgs := []driver.Value{"account-2", "account-1"}
	want := []nodeDeleteCall{
		{Query: "BEGIN"},
		{Query: "DELETE FROM sessions WHERE account_id = ?", Args: deleteArgs},
		{Query: "UPDATE tenant_memberships SET create_id = ? WHERE create_id = ?", Args: reassignArgs},
		{Query: "UPDATE nodes SET create_id = ? WHERE create_id = ?", Args: reassignArgs},
		{Query: "UPDATE nodes SET owner_id = ? WHERE owner_id = ?", Args: reassignArgs},
		{Query: "UPDATE node_links SET create_id = ? WHERE create_id = ?", Args: reassignArgs},
		{Query: "UPDATE node_links SET owner_id = ? WHERE owner_id = ?", Args: reassignArgs},
		{Query: "UPDATE scopes SET create_id = ? WHERE create_id = ?", Args: reassignArgs},
		{Query: "UPDATE scopes SET owner_id = ? WHERE owner_id = ?", Args: reassignArgs},
		{Query: "UPDATE chains SET create_id = ? WHERE create_id = ?", Args: reassignArgs},
		{Query: "UPDATE chains SET owner_id = ? WHERE owner_id = ?", Args: reassignArgs},
		{Query: "UPDATE route_rule_groups SET create_id = ? WHERE create_id = ?", Args: reassignArgs},
		{Query: "UPDATE route_rule_groups SET owner_id = ? WHERE owner_id = ?", Args: reassignArgs},
		{Query: "UPDATE route_rules SET create_id = ? WHERE create_id = ?", Args: reassignArgs},
		{Query: "UPDATE route_rules SET owner_id = ? WHERE owner_id = ?", Args: reassignArgs},
		{Query: "UPDATE policy_revisions SET created_by_account_id = ? WHERE created_by_account_id = ?", Args: reassignArgs},
		{Query: "UPDATE tenant_nodes SET create_id = ? WHERE create_id = ?", Args: reassignArgs},
		{Query: "UPDATE tenant_node_links SET create_id = ? WHERE create_id = ?", Args: reassignArgs},
		{Query: "UPDATE tenant_chains SET create_id = ? WHERE create_id = ?", Args: reassignArgs},
		{Query: "UPDATE tenant_route_rule_groups SET create_id = ? WHERE create_id = ?", Args: reassignArgs},
		{Query: "UPDATE tenant_scopes SET create_id = ? WHERE create_id = ?", Args: reassignArgs},
		{Query: "UPDATE tenant_access_paths SET create_id = ? WHERE create_id = ?", Args: reassignArgs},
		{Query: "UPDATE node_onboarding_tasks SET requested_by_account_id = ? WHERE requested_by_account_id = ?", Args: reassignArgs},
		{Query: "DELETE FROM accounts WHERE id = ?", Args: deleteArgs},
		{Query: "COMMIT"},
	}
	got := callsFrom(record.snapshot(), "BEGIN")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %#v, want %#v", got, want)
	}
}

func callsFrom(calls []nodeDeleteCall, query string) []nodeDeleteCall {
	for index, call := range calls {
		if call.Query == query {
			return calls[index:]
		}
	}
	return nil
}

func deleteCallsFromPlan(plan deleteplan.DeletePlan) []nodeDeleteCall {
	calls := make([]nodeDeleteCall, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		args := make([]driver.Value, 0, len(step.Args))
		for _, arg := range step.Args {
			args = append(args, driver.Value(arg))
		}
		calls = append(calls, nodeDeleteCall{Query: "DELETE FROM " + step.Table + " WHERE " + step.WhereSQL, Args: args})
	}
	return calls
}

func stripExpectedImpact(steps []deleteplan.DeletePlanStep) []deleteplan.DeletePlanStep {
	clean := make([]deleteplan.DeletePlanStep, len(steps))
	copy(clean, steps)
	for index := range clean {
		clean[index].ExpectedImpact = nil
	}
	return clean
}
