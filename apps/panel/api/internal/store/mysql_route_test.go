package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"reflect"
	"sync"
	"testing"
)

type routeRuleDeleteCall struct {
	Query string
	Args  []driver.Value
}

type routeRuleDeleteRecord struct {
	mu    sync.Mutex
	calls []routeRuleDeleteCall
}

func (r *routeRuleDeleteRecord) add(query string, args []driver.Value) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, routeRuleDeleteCall{Query: query, Args: args})
}

func (r *routeRuleDeleteRecord) snapshot() []routeRuleDeleteCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	calls := make([]routeRuleDeleteCall, len(r.calls))
	copy(calls, r.calls)
	return calls
}

var routeRuleDeleteDriverOnce sync.Once
var routeRuleDeleteRecords sync.Map

type routeRuleDeleteDriver struct{}

func (routeRuleDeleteDriver) Open(name string) (driver.Conn, error) {
	value, ok := routeRuleDeleteRecords.Load(name)
	if !ok {
		return nil, errors.New("missing route rule delete record")
	}
	return &routeRuleDeleteConn{record: value.(*routeRuleDeleteRecord)}, nil
}

type routeRuleDeleteConn struct {
	record *routeRuleDeleteRecord
}

func (c *routeRuleDeleteConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare not supported")
}

func (c *routeRuleDeleteConn) Close() error {
	return nil
}

func (c *routeRuleDeleteConn) Begin() (driver.Tx, error) {
	c.record.add("BEGIN", nil)
	return &routeRuleDeleteTx{record: c.record}, nil
}

func (c *routeRuleDeleteConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	values := make([]driver.Value, 0, len(args))
	for _, arg := range args {
		values = append(values, arg.Value)
	}
	c.record.add(query, values)
	return driver.RowsAffected(1), nil
}

type routeRuleDeleteTx struct {
	record *routeRuleDeleteRecord
}

func (tx *routeRuleDeleteTx) Commit() error {
	tx.record.add("COMMIT", nil)
	return nil
}

func (tx *routeRuleDeleteTx) Rollback() error {
	tx.record.add("ROLLBACK", nil)
	return nil
}

func openRouteRuleDeleteTestDB(t *testing.T, record *routeRuleDeleteRecord) *sql.DB {
	t.Helper()
	routeRuleDeleteDriverOnce.Do(func() {
		sql.Register("route_rule_delete_test", routeRuleDeleteDriver{})
	})
	name := t.Name()
	routeRuleDeleteRecords.Store(name, record)
	t.Cleanup(func() {
		routeRuleDeleteRecords.Delete(name)
	})
	db, err := sql.Open("route_rule_delete_test", name)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return db
}

func TestDeleteRouteRuleDeletesTenantBindingsBeforeRouteRule(t *testing.T) {
	record := &routeRuleDeleteRecord{}
	db := openRouteRuleDeleteTestDB(t, record)
	defer db.Close()

	store := &MySQLStore{db: db}
	if err := store.DeleteRouteRule("route-rule-1"); err != nil {
		t.Fatalf("DeleteRouteRule: %v", err)
	}

	want := []routeRuleDeleteCall{
		{Query: "BEGIN"},
		{Query: "DELETE FROM tenant_route_rules WHERE route_rule_id = ?", Args: []driver.Value{"route-rule-1"}},
		{Query: "DELETE FROM route_rules WHERE id = ?", Args: []driver.Value{"route-rule-1"}},
		{Query: "COMMIT"},
	}
	if got := record.snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %#v, want %#v", got, want)
	}
}
