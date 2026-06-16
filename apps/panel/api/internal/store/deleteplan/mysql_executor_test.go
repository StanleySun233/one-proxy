package deleteplan

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
)

type executorCall struct {
	Query string
	Args  []driver.Value
}

type executorRecord struct {
	mu           sync.Mutex
	calls        []executorCall
	rowsAffected map[string]int64
	failQuery    string
}

func (r *executorRecord) add(query string, args []driver.Value) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, executorCall{Query: query, Args: args})
}

func (r *executorRecord) snapshot() []executorCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	calls := make([]executorCall, len(r.calls))
	copy(calls, r.calls)
	return calls
}

var executorDriverOnce sync.Once
var executorRecords sync.Map

type executorDriver struct{}

func (executorDriver) Open(name string) (driver.Conn, error) {
	value, ok := executorRecords.Load(name)
	if !ok {
		return nil, errors.New("missing executor record")
	}
	return &executorConn{record: value.(*executorRecord)}, nil
}

type executorConn struct {
	record *executorRecord
}

func (c *executorConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare not supported")
}

func (c *executorConn) Close() error {
	return nil
}

func (c *executorConn) Begin() (driver.Tx, error) {
	c.record.add("BEGIN", nil)
	return &executorTx{record: c.record}, nil
}

func (c *executorConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	values := make([]driver.Value, 0, len(args))
	for _, arg := range args {
		values = append(values, arg.Value)
	}
	c.record.add(query, values)
	if query == c.record.failQuery {
		return nil, errors.New("forced exec error")
	}
	if c.record.rowsAffected != nil {
		if rowsAffected, ok := c.record.rowsAffected[query]; ok {
			return driver.RowsAffected(rowsAffected), nil
		}
	}
	return driver.RowsAffected(0), nil
}

type executorTx struct {
	record *executorRecord
}

func (tx *executorTx) Commit() error {
	tx.record.add("COMMIT", nil)
	return nil
}

func (tx *executorTx) Rollback() error {
	tx.record.add("ROLLBACK", nil)
	return nil
}

func openExecutorTestDB(t *testing.T, record *executorRecord) *sql.DB {
	t.Helper()
	executorDriverOnce.Do(func() {
		sql.Register("deleteplan_executor_test", executorDriver{})
	})
	name := t.Name()
	executorRecords.Store(name, record)
	t.Cleanup(func() {
		executorRecords.Delete(name)
	})
	db, err := sql.Open("deleteplan_executor_test", name)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return db
}

func TestMySQLExecutorExecutesStepsInOrder(t *testing.T) {
	record := &executorRecord{}
	db := openExecutorTestDB(t, record)
	defer db.Close()

	plan := DeletePlan{
		ResourceType: "chain",
		ResourceID:   "chain-1",
		Steps: []DeletePlanStep{
			{
				Name:      "route-bindings",
				Table:     "tenant_route_rules",
				Operation: OperationDelete,
				WhereSQL:  "route_rule_id IN (SELECT id FROM route_rules WHERE chain_id = ?)",
				Args:      []any{"chain-1"},
			},
			{
				Name:      "chain",
				Table:     "chains",
				Operation: OperationDelete,
				WhereSQL:  "id = ?",
				Args:      []any{"chain-1"},
			},
		},
	}

	if _, err := NewMySQLExecutor(db).Execute(context.Background(), plan); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	args := []driver.Value{"chain-1"}
	want := []executorCall{
		{Query: "BEGIN"},
		{Query: "DELETE FROM tenant_route_rules WHERE route_rule_id IN (SELECT id FROM route_rules WHERE chain_id = ?)", Args: args},
		{Query: "DELETE FROM chains WHERE id = ?", Args: args},
		{Query: "COMMIT"},
	}
	if got := record.snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %#v, want %#v", got, want)
	}
}

func TestMySQLExecutorRollsBackOnStepError(t *testing.T) {
	failQuery := "DELETE FROM chains WHERE id = ?"
	record := &executorRecord{failQuery: failQuery}
	db := openExecutorTestDB(t, record)
	defer db.Close()

	plan := DeletePlan{
		ResourceType: "chain",
		ResourceID:   "chain-1",
		Steps: []DeletePlanStep{
			{
				Name:      "route-rules",
				Table:     "route_rules",
				Operation: OperationDelete,
				WhereSQL:  "chain_id = ?",
				Args:      []any{"chain-1"},
			},
			{
				Name:      "chain",
				Table:     "chains",
				Operation: OperationDelete,
				WhereSQL:  "id = ?",
				Args:      []any{"chain-1"},
			},
		},
	}

	result, err := NewMySQLExecutor(db).Execute(context.Background(), plan)
	if err == nil {
		t.Fatal("Execute succeeded, want error")
	}
	if !strings.Contains(err.Error(), `execute delete step "chain"`) {
		t.Fatalf("error = %q, want step name", err.Error())
	}
	if len(result.Steps) != 1 {
		t.Fatalf("result steps = %d, want 1", len(result.Steps))
	}

	args := []driver.Value{"chain-1"}
	want := []executorCall{
		{Query: "BEGIN"},
		{Query: "DELETE FROM route_rules WHERE chain_id = ?", Args: args},
		{Query: failQuery, Args: args},
		{Query: "ROLLBACK"},
	}
	if got := record.snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %#v, want %#v", got, want)
	}
}

func TestMySQLExecutorReportsRowsAffected(t *testing.T) {
	record := &executorRecord{
		rowsAffected: map[string]int64{
			"DELETE FROM route_rules WHERE chain_id = ?": 3,
			"DELETE FROM chains WHERE id = ?":            1,
		},
	}
	db := openExecutorTestDB(t, record)
	defer db.Close()

	plan := DeletePlan{
		ResourceType: "chain",
		ResourceID:   "chain-1",
		Steps: []DeletePlanStep{
			{
				Name:      "route-rules",
				Table:     "route_rules",
				Operation: OperationDelete,
				WhereSQL:  "chain_id = ?",
				Args:      []any{"chain-1"},
			},
			{
				Name:      "chain",
				Table:     "chains",
				Operation: OperationDelete,
				WhereSQL:  "id = ?",
				Args:      []any{"chain-1"},
			},
		},
	}

	got, err := NewMySQLExecutor(db).Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	want := DeleteExecutionResult{
		PlanResourceType: "chain",
		PlanResourceID:   "chain-1",
		Steps: []DeleteStepResult{
			{Name: "route-rules", Table: "route_rules", RowsAffected: 3},
			{Name: "chain", Table: "chains", RowsAffected: 1},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("result = %#v, want %#v", got, want)
	}
}
