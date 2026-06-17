package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func TestProbeNodeParentURLReachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/healthz" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		_, _ = w.Write([]byte(`{"status":"ok","mode":"proxy-node","controlPlaneBound":true}`))
	}))
	defer server.Close()

	result, err := (&ControlPlane{}).ProbeNodeParentURL(context.Background(), tenantAdminContext(), domain.ProbeNodeParentURLInput{URL: server.URL})
	if err != nil {
		t.Fatalf("ProbeNodeParentURL: %v", err)
	}
	if !result.Reachable || result.Mode != "proxy-node" || result.ControlPlaneBound == nil || !*result.ControlPlaneBound {
		t.Fatalf("result = %+v", result)
	}
}

func TestNormalizeNodeParentURLAddsHTTP(t *testing.T) {
	result, err := normalizeNodeParentURL("127.0.0.1:2988")
	if err != nil {
		t.Fatalf("normalizeNodeParentURL: %v", err)
	}
	if result != "http://127.0.0.1:2988" {
		t.Fatalf("result = %q", result)
	}
}

func TestProbeNodeParentURLRejectsUnsupportedScheme(t *testing.T) {
	_, err := (&ControlPlane{}).ProbeNodeParentURL(context.Background(), tenantAdminContext(), domain.ProbeNodeParentURLInput{URL: "file:///etc/passwd"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func tenantAdminContext() domain.TenantAuthContext {
	return domain.TenantAuthContext{
		Account: domain.Account{ID: "account-1", Role: domain.AccountRoleSuperAdmin},
		ActiveTenant: domain.TenantMembership{
			TenantID: "tenant-1",
			Role:     domain.TenantRoleAdmin,
		},
		SuperAdmin: true,
	}
}
