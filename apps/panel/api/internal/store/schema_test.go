package store

import (
	"context"
	"strings"
	"testing"
)

func TestInitSchemaRunsFinalSchemaForEmptyDatabase(t *testing.T) {
	record := &tokenSecurityRecord{}
	db := openTokenSecurityTestDB(t, record)
	defer db.Close()

	store := &MySQLStore{db: db}
	if err := store.initSchema(context.Background()); err != nil {
		t.Fatalf("initSchema: %v", err)
	}

	findTokenSecurityCall(t, record, "CREATE TABLE IF NOT EXISTS roles")
	findTokenSecurityCall(t, record, "CREATE TABLE IF NOT EXISTS tenant_access_paths")
	findTokenSecurityCall(t, record, "INSERT IGNORE INTO field_enum")
}

func TestInitSchemaSkipsNonEmptyDatabase(t *testing.T) {
	record := &tokenSecurityRecord{tableCount: 1}
	db := openTokenSecurityTestDB(t, record)
	defer db.Close()

	store := &MySQLStore{db: db}
	if err := store.initSchema(context.Background()); err != nil {
		t.Fatalf("initSchema: %v", err)
	}

	for _, call := range record.snapshot() {
		if strings.HasPrefix(normalizedQuery(call.Query), "CREATE TABLE") {
			t.Fatalf("unexpected schema statement: %s", normalizedQuery(call.Query))
		}
	}
}
