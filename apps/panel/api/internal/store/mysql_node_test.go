package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"reflect"
	"sync"
	"testing"
)

type nodeDeleteCall struct {
	Query string
	Args  []driver.Value
}

type nodeDeleteRecord struct {
	mu       sync.Mutex
	calls    []nodeDeleteCall
	chainIDs []string
}

func (r *nodeDeleteRecord) add(query string, args []driver.Value) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, nodeDeleteCall{Query: query, Args: args})
}

func (r *nodeDeleteRecord) snapshot() []nodeDeleteCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	calls := make([]nodeDeleteCall, len(r.calls))
	copy(calls, r.calls)
	return calls
}

var nodeDeleteDriverOnce sync.Once
var nodeDeleteRecords sync.Map

type nodeDeleteDriver struct{}

func (nodeDeleteDriver) Open(name string) (driver.Conn, error) {
	value, ok := nodeDeleteRecords.Load(name)
	if !ok {
		return nil, errors.New("missing node delete record")
	}
	return &nodeDeleteConn{record: value.(*nodeDeleteRecord)}, nil
}

type nodeDeleteConn struct {
	record *nodeDeleteRecord
}

func (c *nodeDeleteConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare not supported")
}

func (c *nodeDeleteConn) Close() error {
	return nil
}

func (c *nodeDeleteConn) Begin() (driver.Tx, error) {
	c.record.add("BEGIN", nil)
	return &nodeDeleteTx{record: c.record}, nil
}

func (c *nodeDeleteConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	values := namedValues(args)
	c.record.add(query, values)
	return driver.RowsAffected(1), nil
}

func (c *nodeDeleteConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.record.add(query, namedValues(args))
	values := make([][]driver.Value, 0, len(c.record.chainIDs))
	for _, chainID := range c.record.chainIDs {
		values = append(values, []driver.Value{chainID})
	}
	return &nodeDeleteRows{values: values}, nil
}

type nodeDeleteTx struct {
	record *nodeDeleteRecord
}

func (tx *nodeDeleteTx) Commit() error {
	tx.record.add("COMMIT", nil)
	return nil
}

func (tx *nodeDeleteTx) Rollback() error {
	tx.record.add("ROLLBACK", nil)
	return nil
}

type nodeDeleteRows struct {
	values [][]driver.Value
	index  int
}

func (r *nodeDeleteRows) Columns() []string {
	return []string{"chain_id"}
}

func (r *nodeDeleteRows) Close() error {
	return nil
}

func (r *nodeDeleteRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

func namedValues(args []driver.NamedValue) []driver.Value {
	values := make([]driver.Value, 0, len(args))
	for _, arg := range args {
		values = append(values, arg.Value)
	}
	return values
}

func openNodeDeleteTestDB(t *testing.T, record *nodeDeleteRecord) *sql.DB {
	t.Helper()
	nodeDeleteDriverOnce.Do(func() {
		sql.Register("node_delete_test", nodeDeleteDriver{})
	})
	name := t.Name()
	nodeDeleteRecords.Store(name, record)
	t.Cleanup(func() {
		nodeDeleteRecords.Delete(name)
	})
	db, err := sql.Open("node_delete_test", name)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return db
}

func TestDeleteNodeDeletesRelationshipsBeforeNode(t *testing.T) {
	record := &nodeDeleteRecord{chainIDs: []string{"chain-1", "chain-2"}}
	db := openNodeDeleteTestDB(t, record)
	defer db.Close()

	store := &MySQLStore{db: db}
	if err := store.DeleteNode("node-1"); err != nil {
		t.Fatalf("DeleteNode: %v", err)
	}

	chainArgs := []driver.Value{"chain-1", "chain-2"}
	nodeArg := []driver.Value{"node-1"}
	twoNodeArgs := []driver.Value{"node-1", "node-1"}
	threeNodeArgs := []driver.Value{"node-1", "node-1", "node-1"}
	want := []nodeDeleteCall{
		{Query: "BEGIN"},
		{Query: "SELECT DISTINCT chain_id FROM chain_hops WHERE node_id = ? ORDER BY chain_id", Args: nodeArg},
		{Query: "DELETE FROM tenant_route_rules WHERE route_rule_id IN (SELECT id FROM route_rules WHERE chain_id IN (?,?))", Args: chainArgs},
		{Query: "DELETE FROM route_rules WHERE chain_id IN (?,?)", Args: chainArgs},
		{Query: "DELETE FROM node_onboarding_tasks WHERE path_id IN (SELECT id FROM node_access_paths WHERE chain_id IN (?,?))", Args: chainArgs},
		{Query: "DELETE FROM node_access_paths WHERE chain_id IN (?,?)", Args: chainArgs},
		{Query: "DELETE FROM chain_probe_results WHERE chain_id IN (?,?)", Args: chainArgs},
		{Query: "DELETE FROM tenant_chains WHERE chain_id IN (?,?)", Args: chainArgs},
		{Query: "DELETE FROM chain_hops WHERE chain_id IN (?,?)", Args: chainArgs},
		{Query: "DELETE FROM chains WHERE id IN (?,?)", Args: chainArgs},
		{Query: "DELETE FROM node_onboarding_tasks WHERE path_id IN (SELECT id FROM node_access_paths WHERE target_node_id = ? OR entry_node_id = ? OR JSON_CONTAINS(relay_node_ids_json, JSON_QUOTE(?)))", Args: threeNodeArgs},
		{Query: "DELETE FROM node_access_paths WHERE target_node_id = ? OR entry_node_id = ? OR JSON_CONTAINS(relay_node_ids_json, JSON_QUOTE(?))", Args: threeNodeArgs},
		{Query: "DELETE FROM chain_probe_results WHERE blocking_node_id = ?", Args: nodeArg},
		{Query: "DELETE FROM node_transports WHERE node_id = ? OR parent_node_id = ?", Args: twoNodeArgs},
		{Query: "DELETE FROM node_links WHERE source_node_id = ? OR target_node_id = ?", Args: twoNodeArgs},
		{Query: "DELETE FROM node_onboarding_tasks WHERE target_node_id = ?", Args: nodeArg},
		{Query: "DELETE FROM node_policy_assignments WHERE node_id = ?", Args: nodeArg},
		{Query: "DELETE FROM node_health_snapshots WHERE node_id = ?", Args: nodeArg},
		{Query: "DELETE FROM node_sla_minutes WHERE node_id = ?", Args: nodeArg},
		{Query: "DELETE FROM node_api_tokens WHERE node_id = ?", Args: nodeArg},
		{Query: "DELETE FROM node_trust_materials WHERE node_id = ?", Args: nodeArg},
		{Query: "DELETE FROM bootstrap_tokens WHERE target_id = ?", Args: nodeArg},
		{Query: "DELETE FROM tenant_nodes WHERE node_id = ?", Args: nodeArg},
		{Query: "UPDATE nodes SET parent_node_id = NULL WHERE parent_node_id = ?", Args: nodeArg},
		{Query: "DELETE FROM nodes WHERE id = ?", Args: nodeArg},
		{Query: "COMMIT"},
	}
	if got := record.snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %#v, want %#v", got, want)
	}
}
