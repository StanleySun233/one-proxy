package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/auth"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

type tokenSecurityCall struct {
	Query string
	Args  []driver.Value
}

type tokenSecurityRecord struct {
	mu         sync.Mutex
	calls      []tokenSecurityCall
	sequences  map[string]int64
	nodeStatus string
}

func (r *tokenSecurityRecord) add(query string, args []driver.Value) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, tokenSecurityCall{Query: query, Args: args})
}

func (r *tokenSecurityRecord) nextSequence(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sequences == nil {
		r.sequences = map[string]int64{}
	}
	r.sequences[name]++
	if r.sequences[name] == 0 {
		r.sequences[name] = 1
	}
}

func (r *tokenSecurityRecord) sequence(name string) int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sequences == nil || r.sequences[name] == 0 {
		return 1
	}
	return r.sequences[name]
}

func (r *tokenSecurityRecord) snapshot() []tokenSecurityCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	calls := make([]tokenSecurityCall, len(r.calls))
	copy(calls, r.calls)
	return calls
}

var tokenSecurityDriverOnce sync.Once
var tokenSecurityRecords sync.Map

type tokenSecurityDriver struct{}

func (tokenSecurityDriver) Open(name string) (driver.Conn, error) {
	value, ok := tokenSecurityRecords.Load(name)
	if !ok {
		return nil, errors.New("missing token security record")
	}
	return &tokenSecurityConn{record: value.(*tokenSecurityRecord)}, nil
}

type tokenSecurityConn struct {
	record *tokenSecurityRecord
}

func (c *tokenSecurityConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare not supported")
}

func (c *tokenSecurityConn) Close() error {
	return nil
}

func (c *tokenSecurityConn) Begin() (driver.Tx, error) {
	c.record.add("BEGIN", nil)
	return &tokenSecurityTx{record: c.record}, nil
}

func (c *tokenSecurityConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	values := tokenSecurityNamedValues(args)
	c.record.add(query, values)
	if strings.HasPrefix(normalizedQuery(query), "INSERT INTO id_sequences") && len(values) > 0 {
		if name, ok := values[0].(string); ok {
			c.record.nextSequence(name)
		}
	}
	return driver.RowsAffected(1), nil
}

func (c *tokenSecurityConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	values := tokenSecurityNamedValues(args)
	c.record.add(query, values)
	return c.record.rows(query, values), nil
}

type tokenSecurityTx struct {
	record *tokenSecurityRecord
}

func (tx *tokenSecurityTx) Commit() error {
	tx.record.add("COMMIT", nil)
	return nil
}

func (tx *tokenSecurityTx) Rollback() error {
	tx.record.add("ROLLBACK", nil)
	return nil
}

type tokenSecurityRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (r *tokenSecurityRows) Columns() []string {
	return r.columns
}

func (r *tokenSecurityRows) Close() error {
	return nil
}

func (r *tokenSecurityRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

func (r *tokenSecurityRecord) rows(query string, args []driver.Value) driver.Rows {
	normalized := normalizedQuery(query)
	switch {
	case strings.HasPrefix(normalized, "SELECT COALESCE(MAX(CAST(id AS UNSIGNED)), 0) FROM"):
		return tokenSecurityRow([]string{"max"}, 0)
	case normalized == "SELECT current_value FROM id_sequences WHERE name = ?":
		name, _ := args[0].(string)
		return tokenSecurityRow([]string{"current_value"}, r.sequence(name))
	case strings.HasPrefix(normalized, "SELECT id, target_id, node_name, node_mode, scope_key, parent_node_id, public_host, public_port, expires_at, consumed_at FROM bootstrap_tokens WHERE token_hash = ?"):
		return tokenSecurityRow([]string{"id", "target_id", "node_name", "node_mode", "scope_key", "parent_node_id", "public_host", "public_port", "expires_at", "consumed_at"}, "bootstrap-1", "node-1", "local-node", "relay", "default", nil, "127.0.0.1", int64(2988), time.Now().UTC().Add(time.Hour).Format(time.RFC3339), nil)
	case strings.HasPrefix(normalized, "SELECT id, name, mode, scope_key, COALESCE(parent_node_id, ''), enabled, status, COALESCE(public_host, ''), COALESCE(public_port, 0) FROM nodes WHERE id = ?"):
		status := r.nodeStatus
		if status == "" {
			status = domain.NodeStatusHealthy
		}
		return tokenSecurityRow([]string{"id", "name", "mode", "scope_key", "parent_node_id", "enabled", "status", "public_host", "public_port"}, "node-1", "local-node", "relay", "default", "", int64(1), status, "127.0.0.1", int64(2988))
	case strings.HasPrefix(normalized, "SELECT COUNT(1) FROM node_trust_materials WHERE node_id = ? AND material_type = 'enrollment_secret'"):
		return tokenSecurityRow([]string{"count"}, int64(1))
	case strings.HasPrefix(normalized, "SELECT material_value FROM node_trust_materials WHERE node_id = ? AND material_type = 'shared_secret'"):
		return tokenSecurityRow([]string{"material_value"}, "shared-secret")
	case strings.HasPrefix(normalized, "SELECT t.node_id, t.expires_at, n.status, n.enabled FROM node_api_tokens t JOIN nodes n ON n.id = t.node_id WHERE t.token_hash = ?"):
		return tokenSecurityRow([]string{"node_id", "expires_at", "status", "enabled"}, "node-1", time.Now().UTC().Add(time.Hour).Format(time.RFC3339), domain.NodeStatusHealthy, int64(1))
	default:
		return &tokenSecurityRows{columns: []string{"empty"}}
	}
}

func tokenSecurityRow(columns []string, values ...driver.Value) driver.Rows {
	return &tokenSecurityRows{columns: columns, values: [][]driver.Value{values}}
}

func tokenSecurityNamedValues(args []driver.NamedValue) []driver.Value {
	values := make([]driver.Value, 0, len(args))
	for _, arg := range args {
		values = append(values, arg.Value)
	}
	return values
}

func openTokenSecurityTestDB(t *testing.T, record *tokenSecurityRecord) *sql.DB {
	t.Helper()
	tokenSecurityDriverOnce.Do(func() {
		sql.Register("token_security_test", tokenSecurityDriver{})
	})
	name := t.Name()
	tokenSecurityRecords.Store(name, record)
	t.Cleanup(func() {
		tokenSecurityRecords.Delete(name)
	})
	db, err := sql.Open("token_security_test", name)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return db
}

func TestCreateBootstrapTokenForTenantStoresTokenHash(t *testing.T) {
	record := &tokenSecurityRecord{}
	db := openTokenSecurityTestDB(t, record)
	defer db.Close()

	store := &MySQLStore{db: db}
	result, err := store.CreateBootstrapTokenForTenant(domain.TenantAuthContext{
		Account:      domain.Account{ID: "account-1"},
		ActiveTenant: domain.TenantMembership{TenantID: "tenant-1"},
	}, domain.CreateBootstrapTokenInput{
		TargetType: domain.BootstrapTargetTypeNode,
		TargetID:   "node-1",
		NodeName:   "local-node",
		NodeMode:   "relay",
		ScopeKey:   "default",
		PublicHost: "127.0.0.1",
		PublicPort: 2988,
	})
	if err != nil {
		t.Fatalf("CreateBootstrapTokenForTenant: %v", err)
	}

	call := findTokenSecurityCall(t, record, "INSERT INTO bootstrap_tokens")
	assertStoredTokenHash(t, call.Args[1], result.Token)
}

func TestEnrollNodeLooksUpBootstrapTokenHash(t *testing.T) {
	rawToken := "bootstrap-token"
	record := &tokenSecurityRecord{nodeStatus: domain.NodeStatusPending}
	db := openTokenSecurityTestDB(t, record)
	defer db.Close()

	store := &MySQLStore{db: db}
	result, err := store.EnrollNode(domain.EnrollNodeInput{Token: rawToken})
	if err != nil {
		t.Fatalf("EnrollNode: %v", err)
	}
	if result.EnrollmentSecret == "" {
		t.Fatalf("EnrollmentSecret is empty")
	}

	call := findTokenSecurityCall(t, record, "FROM bootstrap_tokens WHERE token_hash = ?")
	if got := call.Args[0]; got != auth.TokenHash(rawToken) {
		t.Fatalf("bootstrap lookup arg = %v, want %s", got, auth.TokenHash(rawToken))
	}
}

func TestApproveNodeEnrollmentStoresNodeTokenHash(t *testing.T) {
	record := &tokenSecurityRecord{nodeStatus: domain.NodeStatusPending}
	db := openTokenSecurityTestDB(t, record)
	defer db.Close()

	store := &MySQLStore{db: db}
	result, err := store.ApproveNodeEnrollment("node-1", "account-1")
	if err != nil {
		t.Fatalf("ApproveNodeEnrollment: %v", err)
	}

	call := findTokenSecurityCall(t, record, "INSERT INTO node_api_tokens")
	assertStoredTokenHash(t, call.Args[2], result.AccessToken)
}

func TestExchangeNodeEnrollmentIssuesHashedNodeToken(t *testing.T) {
	record := &tokenSecurityRecord{nodeStatus: domain.NodeStatusHealthy}
	db := openTokenSecurityTestDB(t, record)
	defer db.Close()

	store := &MySQLStore{db: db}
	result, err := store.ExchangeNodeEnrollment(domain.ExchangeNodeEnrollmentInput{
		NodeID:           "node-1",
		EnrollmentSecret: "enrollment-secret",
	})
	if err != nil {
		t.Fatalf("ExchangeNodeEnrollment: %v", err)
	}

	call := findTokenSecurityCall(t, record, "INSERT INTO node_api_tokens")
	assertStoredTokenHash(t, call.Args[2], result.AccessToken)
	for _, item := range record.snapshot() {
		if strings.Contains(normalizedQuery(item.Query), "SELECT token_hash") {
			t.Fatalf("exchange read persisted token: %s", normalizedQuery(item.Query))
		}
	}
}

func TestAuthenticateNodeTokenLooksUpTokenHash(t *testing.T) {
	rawToken := "node-token"
	record := &tokenSecurityRecord{}
	db := openTokenSecurityTestDB(t, record)
	defer db.Close()

	store := &MySQLStore{db: db}
	nodeID, ok := store.AuthenticateNodeToken(rawToken)
	if !ok || nodeID != "node-1" {
		t.Fatalf("AuthenticateNodeToken = %q, %v", nodeID, ok)
	}

	call := findTokenSecurityCall(t, record, "FROM node_api_tokens t JOIN nodes n ON n.id = t.node_id WHERE t.token_hash = ?")
	if got := call.Args[0]; got != auth.TokenHash(rawToken) {
		t.Fatalf("node token lookup arg = %v, want %s", got, auth.TokenHash(rawToken))
	}
}

func findTokenSecurityCall(t *testing.T, record *tokenSecurityRecord, pattern string) tokenSecurityCall {
	t.Helper()
	for _, call := range record.snapshot() {
		if strings.Contains(normalizedQuery(call.Query), pattern) {
			return call
		}
	}
	t.Fatalf("missing call containing %q in %#v", pattern, record.snapshot())
	return tokenSecurityCall{}
}

func assertStoredTokenHash(t *testing.T, stored driver.Value, raw string) {
	t.Helper()
	value, ok := stored.(string)
	if !ok {
		t.Fatalf("stored token value type = %T", stored)
	}
	if value == raw {
		t.Fatalf("stored raw token")
	}
	if value != auth.TokenHash(raw) {
		t.Fatalf("stored token = %s, want %s", value, auth.TokenHash(raw))
	}
	if len(value) != 64 {
		t.Fatalf("stored token hash length = %d", len(value))
	}
}
