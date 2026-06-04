package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
)

func TestReportProxySessionsPostsNodeAgentMetrics(t *testing.T) {
	var received domain.ProxySessionMetricsInput
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/node/agent/proxy/sessions" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		if req.Header.Get("X-One-Proxy-Node-Token") != "node-token" {
			t.Fatalf("node token = %q", req.Header.Get("X-One-Proxy-Node-Token"))
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

func TestValidateNodeAuthUsesNodeAgentEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/node/agent/auth/validate" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		if req.Header.Get("X-One-Proxy-Node-Token") != "child-token" {
			t.Fatalf("node token = %q", req.Header.Get("X-One-Proxy-Node-Token"))
		}
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"nodeId":"child-node"}}`))
	}))
	defer server.Close()

	result, err := New(server.URL, "child-token").ValidateNodeAuth()
	if err != nil {
		t.Fatal(err)
	}
	if result.NodeID != "child-node" {
		t.Fatalf("nodeID = %q", result.NodeID)
	}
}
