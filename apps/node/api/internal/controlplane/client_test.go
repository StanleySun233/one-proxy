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

func TestValidateProxyTokenPostsRouteContext(t *testing.T) {
	var received ProxyTokenValidationRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/node/agent/proxy/token/validate" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		if req.Header.Get("X-One-Proxy-Node-Token") != "node-token" {
			t.Fatalf("node token = %q", req.Header.Get("X-One-Proxy-Node-Token"))
		}
		if err := json.NewDecoder(req.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"valid":true,"tenantId":"tenant-1","allowLocalProxy":true,"expiresAt":"2026-06-17T12:00:00Z","cacheTtlSeconds":60}}`))
	}))
	defer server.Close()

	result, err := New(server.URL, "node-token").ValidateProxyToken(context.Background(), ProxyTokenValidationRequest{
		TokenHash:    "hash",
		AccessPathID: "path-1",
		TargetHost:   "example.invalid",
		TargetPort:   2333,
		Protocol:     "http",
		RouteID:      "route-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if received.TokenHash != "hash" || received.AccessPathID != "path-1" || received.TargetHost != "example.invalid" || received.TargetPort != 2333 || received.Protocol != "http" || received.RouteID != "route-1" {
		t.Fatalf("received = %+v", received)
	}
	if !result.Valid || result.TenantID != "tenant-1" || !result.AllowLocalProxy {
		t.Fatalf("result = %+v", result)
	}
}

func TestAuthenticateProxyTokenPostsTokenHashOnly(t *testing.T) {
	var received ProxyTokenValidationRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/node/agent/proxy/token/authenticate" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		if req.Header.Get("X-One-Proxy-Node-Token") != "node-token" {
			t.Fatalf("node token = %q", req.Header.Get("X-One-Proxy-Node-Token"))
		}
		if err := json.NewDecoder(req.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"valid":true,"tenantId":"tenant-1","allowLocalProxy":true,"expiresAt":"2026-06-17T12:00:00Z","cacheTtlSeconds":60}}`))
	}))
	defer server.Close()

	result, err := New(server.URL, "node-token").AuthenticateProxyToken(context.Background(), "hash")
	if err != nil {
		t.Fatal(err)
	}
	if received.TokenHash != "hash" || received.AccessPathID != "" || received.TargetHost != "" || received.TargetPort != 0 || received.Protocol != "" || received.RouteID != "" {
		t.Fatalf("received = %+v", received)
	}
	if !result.Valid || result.TenantID != "tenant-1" || !result.AllowLocalProxy {
		t.Fatalf("result = %+v", result)
	}
}
