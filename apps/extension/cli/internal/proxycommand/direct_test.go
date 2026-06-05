package proxycommand

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDialDirectRequestsClientSession(t *testing.T) {
	seen := make(chan directSessionRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/proxy/extension/direct/session" {
			t.Fatalf("path = %s", req.URL.Path)
		}
		if req.Header.Get("X-One-Proxy-Access-Token") != "access-token" {
			t.Fatalf("access token = %q", req.Header.Get("X-One-Proxy-Access-Token"))
		}
		if req.Header.Get("X-One-Proxy-Tenant-ID") != "tenant-1" {
			t.Fatalf("tenant = %q", req.Header.Get("X-One-Proxy-Tenant-ID"))
		}
		var payload directSessionRequest
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		seen <- payload
		_ = json.NewEncoder(w).Encode(struct {
			Code int           `json:"code"`
			Data directSession `json:"data"`
		}{
			Data: directSession{
				SessionID:  "session-1",
				TargetHost: "10.0.0.2",
				TargetPort: 22,
				PunchToken: "punch-1",
			},
		})
	}))
	defer server.Close()

	t.Setenv("ONEPROXY_ACCESS_TOKEN", "access-token")
	err := func() error {
		conn, dialErr := DialDirect(Config{
			PanelURL:       server.URL,
			TenantID:       "tenant-1",
			AccessPathID:   "path-1",
			TargetHost:     "10.0.0.2",
			TargetPort:     22,
			ConnectTimeout: time.Second,
		})
		if conn != nil {
			_ = conn.Close()
		}
		return dialErr
	}()
	if err == nil || err.Error() != "direct_candidates_unavailable" {
		t.Fatalf("error = %v", err)
	}
	payload := <-seen
	if payload.AccessPathID != "path-1" || payload.TargetHost != "10.0.0.2" || payload.TargetPort != 22 {
		t.Fatalf("payload = %+v", payload)
	}
}
