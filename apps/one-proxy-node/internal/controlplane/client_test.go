package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
)

func TestReportProxySessionsPostsNodeAgentMetrics(t *testing.T) {
	var received domain.ProxySessionMetricsInput
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v1/node-agent/proxy-sessions" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		if req.Header.Get("Authorization") != "Bearer node-token" {
			t.Fatalf("authorization = %q", req.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(req.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := New(server.URL, "node-token")
	err := client.ReportProxySessions(context.Background(), domain.ProxySessionMetricsInput{
		Sessions: []domain.ProxySessionMetric{{
			ID:         "session-1",
			TenantID:   "tenant-1",
			NodeID:     "node-1",
			TargetHost: "example.com",
			Protocol:   domain.ProxySessionProtocolHTTP,
			Status:     domain.ProxySessionStatusOK,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(received.Sessions) != 1 || received.Sessions[0].ID != "session-1" {
		t.Fatalf("received = %+v", received)
	}
}
