package store

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func TestNetworkAuditMetadataCacheRoundTrip(t *testing.T) {
	storedAt := time.Date(2026, 6, 16, 10, 30, 0, 0, time.UTC)
	raw := networkAuditMetadataJSON(`{"existing":"ok"}`, "stale", storedAt)
	if !strings.Contains(raw, `"existing":"ok"`) {
		t.Fatalf("metadata lost existing fields: %s", raw)
	}
	status, parsedAt := networkAuditCacheMetadata(raw)
	if status != "stale" {
		t.Fatalf("cache status = %q", status)
	}
	if parsedAt == nil || !parsedAt.Equal(storedAt) {
		t.Fatalf("cache stored-at = %v", parsedAt)
	}
}

func TestNetworkAuditMetadataOmitsZeroCacheStoredAt(t *testing.T) {
	raw := networkAuditMetadataJSON("{}", "stale", time.Time{})
	if strings.Contains(raw, "cacheStoredAt") {
		t.Fatalf("metadata should omit zero cache time: %s", raw)
	}
	status, parsedAt := networkAuditCacheMetadata(raw)
	if status != "stale" || parsedAt != nil {
		t.Fatalf("cache metadata = %q %v", status, parsedAt)
	}
}

func TestNetworkAuditWhereFiltersErrorCode(t *testing.T) {
	where, args := networkAuditWhere(domain.NetworkAuditQuery{
		TenantID:  "tenant-1",
		ErrorCode: "next_hop_connect_failed",
	})
	if !strings.Contains(where, "error_code = ?") {
		t.Fatalf("where = %q", where)
	}
	wantArgs := []any{"tenant-1", "next_hop_connect_failed"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", args, wantArgs)
	}
}
