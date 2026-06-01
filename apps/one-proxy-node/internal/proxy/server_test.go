package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/policystore"
)

func TestForwardDirectUsesForwardProxySemantics(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("X-Forwarded-For") != "" {
			t.Fatalf("unexpected X-Forwarded-For header")
		}
		if req.URL.Path != "/health" {
			t.Fatalf("unexpected path %q", req.URL.Path)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer origin.Close()

	store := policystore.New("")
	payload, err := json.Marshal(policystore.Snapshot{
		RouteRules: []domain.RouteRule{
			{
				ID:         "rule-local-cidr",
				MatchType:  domain.MatchTypeIPCIDR,
				MatchValue: "127.0.0.0/8",
				ActionType: domain.ActionTypeDirect,
				Enabled:    true,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update("test", string(payload)); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, origin.URL+"/health", nil)
	resp := httptest.NewRecorder()
	NewServer(store, func() string { return "node-1" }, nil).ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", resp.Code, resp.Body.String())
	}
	if resp.Body.String() != "ok" {
		t.Fatalf("body = %q", resp.Body.String())
	}
}

func TestMatchSupportsDefaultRoute(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://www.baidu.com/", nil)
	match := Match(policystore.Snapshot{
		RouteRules: []domain.RouteRule{
			{
				ID:         "1",
				MatchType:  domain.MatchTypeDefault,
				ActionType: domain.ActionTypeDirect,
				Enabled:    true,
			},
		},
	}, req)

	if !match.Found {
		t.Fatal("default route did not match")
	}
	if match.Rule.ID != "1" {
		t.Fatalf("rule id = %q", match.Rule.ID)
	}
}
