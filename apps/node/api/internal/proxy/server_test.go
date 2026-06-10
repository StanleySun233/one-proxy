package proxy

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/policystore"
	"github.com/gorilla/websocket"
)

type recordingTokenValidator struct {
	validations     []string
	valid           bool
	expiresAt       time.Time
	allowLocalProxy bool
	tenantID        string
}

type recordingSessionReporter struct {
	sessions chan domain.ProxySessionMetric
}

func (v *recordingTokenValidator) ValidateProxyToken(_ context.Context, tokenHash string) (TokenValidation, error) {
	v.validations = append(v.validations, tokenHash)
	return TokenValidation{Valid: v.valid, ExpiresAt: v.expiresAt, AllowLocalProxy: v.allowLocalProxy, TenantID: v.tenantID}, nil
}

func (r recordingSessionReporter) ReportProxySessions(_ context.Context, input domain.ProxySessionMetricsInput) error {
	for _, session := range input.Sessions {
		r.sessions <- session
	}
	return nil
}

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

func TestForwardDirectRetriesTransientStaticFailure(t *testing.T) {
	attempts := 0
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		attempts += 1
		if req.URL.Path != "/static/js/app.js" {
			t.Fatalf("unexpected path %q", req.URL.Path)
		}
		if attempts == 1 {
			http.Error(w, "bad_gateway", http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte("loaded"))
	}))
	defer origin.Close()

	store := policystore.New("")
	if err := store.Update("test", `{"nodes":[],"links":[],"chains":[],"routeRules":[{"id":"default","matchType":"default","actionType":"direct","enabled":true}]}`); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, origin.URL+"/static/js/app.js", nil)
	resp := httptest.NewRecorder()
	NewServer(store, func() string { return "node-1" }, nil).ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", resp.Code, resp.Body.String())
	}
	if resp.Body.String() != "loaded" {
		t.Fatalf("body = %q", resp.Body.String())
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d", attempts)
	}
}

func TestForwardDirectRetriesPostRequest(t *testing.T) {
	attempts := 0
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		attempts += 1
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "upload" {
			t.Fatalf("body = %q", body)
		}
		if attempts == 1 {
			http.Error(w, "bad_gateway", http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte("saved"))
	}))
	defer origin.Close()

	store := policystore.New("")
	if err := store.Update("test", `{"nodes":[],"links":[],"chains":[],"routeRules":[{"id":"default","matchType":"default","actionType":"direct","enabled":true}]}`); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, origin.URL+"/api/save", strings.NewReader("upload"))
	resp := httptest.NewRecorder()
	NewServer(store, func() string { return "node-1" }, nil).ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", resp.Code, resp.Body.String())
	}
	if resp.Body.String() != "saved" {
		t.Fatalf("body = %q", resp.Body.String())
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d", attempts)
	}
}

func TestForwardDirectRetriesContentLengthMismatch(t *testing.T) {
	attempts := 0
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		attempts += 1
		if attempts == 1 {
			w.Header().Set("Content-Length", "6")
			_, _ = w.Write([]byte("bad"))
			return
		}
		_, _ = w.Write([]byte("loaded"))
	}))
	defer origin.Close()

	store := policystore.New("")
	if err := store.Update("test", `{"nodes":[],"links":[],"chains":[],"routeRules":[{"id":"default","matchType":"default","actionType":"direct","enabled":true}]}`); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, origin.URL+"/api/scenario/id/track", nil)
	resp := httptest.NewRecorder()
	NewServer(store, func() string { return "node-1" }, nil).ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", resp.Code, resp.Body.String())
	}
	if resp.Body.String() != "loaded" {
		t.Fatalf("body = %q", resp.Body.String())
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d", attempts)
	}
}

func TestForwardDirectReportsProxySessionMetrics(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "upload" {
			t.Fatalf("body = %q", body)
		}
		_, _ = w.Write([]byte("reply"))
	}))
	defer origin.Close()

	store := policystore.New("")
	if err := store.Update("test", `{"nodes":[],"links":[],"chains":[],"routeRules":[{"id":"default","tenantId":"tenant-1","matchType":"default","actionType":"direct","destinationScope":"scope-1","enabled":true}]}`); err != nil {
		t.Fatal(err)
	}
	reporter := recordingSessionReporter{sessions: make(chan domain.ProxySessionMetric, 1)}
	server := NewServer(store, func() string { return "node-1" }, nil)
	server.SetProxySessionReporter(reporter)

	req := httptest.NewRequest(http.MethodPost, origin.URL+"/metrics", strings.NewReader("upload"))
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", resp.Code, resp.Body.String())
	}

	session := receiveSession(t, reporter.sessions)
	if session.TenantID != "tenant-1" || session.NodeID != "node-1" || session.RouteID != "default" {
		t.Fatalf("session identity = %+v", session)
	}
	if session.PolicyRevision != "test" || session.ScopeID != "scope-1" || session.MatchedRuleID != "default" || session.MatchedRuleType != domain.MatchTypeDefault || session.MatchedAction != domain.ActionTypeDirect || session.DecisionSource != "policy" {
		t.Fatalf("session evidence = %+v", session)
	}
	if session.Protocol != domain.ProxySessionProtocolHTTP || session.UploadBytes != 6 || session.DownloadBytes != 5 || session.Status != domain.ProxySessionStatusOK {
		t.Fatalf("session metrics = %+v", session)
	}
}

func TestTunnelDirectReportsProxySessionMetrics(t *testing.T) {
	target, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	go func() {
		conn, err := target.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buffer := make([]byte, 4)
		_, _ = io.ReadFull(conn, buffer)
		_, _ = conn.Write([]byte("pong"))
	}()

	store := policystore.New("")
	if err := store.Update("test", `{"nodes":[],"links":[],"chains":[],"routeRules":[{"id":"default","tenantId":"tenant-1","matchType":"default","actionType":"direct","enabled":true}]}`); err != nil {
		t.Fatal(err)
	}
	reporter := recordingSessionReporter{sessions: make(chan domain.ProxySessionMetric, 1)}
	server := NewServer(store, func() string { return "node-1" }, nil)
	server.SetProxySessionReporter(reporter)
	proxyServer := httptest.NewServer(server)
	defer proxyServer.Close()

	proxyURL, err := url.Parse(proxyServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.Dial("tcp", proxyURL.Host)
	if err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(conn)
	targetAddr := target.Addr().String()
	if _, err := fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", targetAddr, targetAddr); err != nil {
		t.Fatal(err)
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(line, "200") {
		t.Fatalf("connect line = %q", line)
	}
	for {
		headerLine, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if headerLine == "\r\n" {
			break
		}
	}
	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	reply := make([]byte, 4)
	if _, err := io.ReadFull(reader, reply); err != nil {
		t.Fatal(err)
	}
	if string(reply) != "pong" {
		t.Fatalf("reply = %q", reply)
	}
	_ = conn.Close()

	session := receiveSession(t, reporter.sessions)
	if session.Protocol != domain.ProxySessionProtocolConnect || session.UploadBytes != 4 || session.DownloadBytes != 4 || session.Status != domain.ProxySessionStatusOK {
		t.Fatalf("session metrics = %+v", session)
	}
}

func TestForwardProxyRequiresProxyAuthorizationToken(t *testing.T) {
	store := policystore.New("")
	payload, err := json.Marshal(policystore.Snapshot{
		RouteRules: []domain.RouteRule{
			{ID: "default", MatchType: domain.MatchTypeDefault, ActionType: domain.ActionTypeDirect, Enabled: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update("test", string(payload)); err != nil {
		t.Fatal(err)
	}
	server, err := NewServerWithOptions(store, func() string { return "node-1" }, nil, "", AuthConfig{
		Validator: &recordingTokenValidator{valid: true, expiresAt: time.Now().UTC().Add(time.Hour)},
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusProxyAuthRequired {
		t.Fatalf("status = %d", resp.Code)
	}
	if resp.Header().Get("Proxy-Authenticate") == "" {
		t.Fatal("missing Proxy-Authenticate")
	}
}

func TestForwardProxyAcceptsBearerAndChromeBasicToken(t *testing.T) {
	store := policystore.New("")
	payload, err := json.Marshal(policystore.Snapshot{
		RouteRules: []domain.RouteRule{
			{ID: "default", MatchType: domain.MatchTypeDefault, ActionType: domain.ActionTypeDirect, Enabled: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update("test", string(payload)); err != nil {
		t.Fatal(err)
	}
	validator := &recordingTokenValidator{valid: true, expiresAt: time.Now().UTC().Add(time.Hour)}
	server, err := NewServerWithOptions(store, func() string { return "node-1" }, nil, "", AuthConfig{
		Validator: validator,
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.invalid/", nil)
	req.Header.Set("Proxy-Authorization", "Bearer proxy-token")
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadGateway {
		t.Fatalf("bearer authorized status = %d", resp.Code)
	}
	expectedHash := sha256Hex("proxy-token")
	if len(validator.validations) != 1 || validator.validations[0] != expectedHash {
		t.Fatalf("validation hashes = %v, want %s", validator.validations, expectedHash)
	}

	req = httptest.NewRequest(http.MethodGet, "http://example.invalid/", nil)
	req.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("token:chrome-token")))
	resp = httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadGateway {
		t.Fatalf("chrome basic authorized status = %d", resp.Code)
	}
	expectedHash = sha256Hex("chrome-token")
	if len(validator.validations) != 2 || validator.validations[1] != expectedHash {
		t.Fatalf("validation hashes = %v, want second %s", validator.validations, expectedHash)
	}
}

func TestForwardProxyAllowsAuthorizedLocalProxyFallback(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/local-proxy" {
			t.Fatalf("path = %q", req.URL.Path)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer origin.Close()

	store := policystore.New("")
	if err := store.Update("test", `{"nodes":[],"links":[],"chains":[],"routeRules":[]}`); err != nil {
		t.Fatal(err)
	}
	server, err := NewServerWithOptions(store, func() string { return "node-1" }, nil, "", AuthConfig{
		Validator: &recordingTokenValidator{valid: true, expiresAt: time.Now().UTC().Add(time.Hour), allowLocalProxy: true, tenantID: "tenant-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	reporter := recordingSessionReporter{sessions: make(chan domain.ProxySessionMetric, 1)}
	server.SetProxySessionReporter(reporter)
	req := httptest.NewRequest(http.MethodGet, origin.URL+"/local-proxy", nil)
	req.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("token:local-proxy-token")))
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", resp.Code, resp.Body.String())
	}
	if resp.Body.String() != "ok" {
		t.Fatalf("body = %q", resp.Body.String())
	}
	session := receiveSession(t, reporter.sessions)
	if session.TenantID != "tenant-1" || session.DecisionSource != "default" || session.MatchedAction != domain.ActionTypeDirect || session.Status != domain.ProxySessionStatusOK {
		t.Fatalf("session = %+v", session)
	}
}

func TestForwardProxyRejectsLocalProxyFallbackWithoutNodeGrant(t *testing.T) {
	store := policystore.New("")
	if err := store.Update("test", `{"nodes":[],"links":[],"chains":[],"routeRules":[]}`); err != nil {
		t.Fatal(err)
	}
	server, err := NewServerWithOptions(store, func() string { return "node-1" }, nil, "", AuthConfig{
		Validator: &recordingTokenValidator{valid: true, expiresAt: time.Now().UTC().Add(time.Hour), tenantID: "tenant-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	reporter := recordingSessionReporter{sessions: make(chan domain.ProxySessionMetric, 1)}
	server.SetProxySessionReporter(reporter)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("token:local-proxy-token")))
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("status = %d", resp.Code)
	}
	session := receiveSession(t, reporter.sessions)
	if session.TenantID != "tenant-1" || session.Status != domain.ProxySessionStatusError || session.ErrorCode != "route_not_found" || session.MatchedAction != domain.ActionTypeDeny {
		t.Fatalf("session = %+v", session)
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

func TestForwardProxyWebSocketUpgrade(t *testing.T) {
	upgrader := websocket.Upgrader{}
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
		defer conn.Close()
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("read failed: %v", err)
			return
		}
		if err := conn.WriteMessage(messageType, payload); err != nil {
			t.Errorf("write failed: %v", err)
		}
	}))
	defer origin.Close()

	store := policystore.New("")
	payload, err := json.Marshal(policystore.Snapshot{
		RouteRules: []domain.RouteRule{
			{
				ID:         "default",
				TenantID:   "tenant-1",
				MatchType:  domain.MatchTypeDefault,
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

	server := NewServer(store, func() string { return "node-1" }, nil)
	reporter := recordingSessionReporter{sessions: make(chan domain.ProxySessionMetric, 1)}
	server.SetProxySessionReporter(reporter)
	proxy := httptest.NewServer(server)
	defer proxy.Close()

	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatal(err)
	}
	dialer := websocket.Dialer{Proxy: http.ProxyURL(proxyURL)}
	targetURL := "ws" + origin.URL[len("http"):] + "/terminal"
	conn, _, err := dialer.Dial(targetURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
		t.Fatal(err)
	}
	_, payload, err = conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != "ping" {
		t.Fatalf("payload = %q", payload)
	}
	session := receiveSession(t, reporter.sessions)
	if session.TenantID != "tenant-1" || session.Status != domain.ProxySessionStatusOK || session.Protocol != domain.ProxySessionProtocolConnect || session.MatchedRuleID != "default" || session.PolicyRevision != "test" {
		t.Fatalf("session = %+v", session)
	}
}

func TestForwardProxyAbsoluteWebSocketUpgradeReportsSessionMetrics(t *testing.T) {
	upgrader := websocket.Upgrader{}
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
		_ = conn.Close()
	}))
	defer origin.Close()

	store := policystore.New("")
	if err := store.Update("test", `{"nodes":[],"links":[],"chains":[],"routeRules":[{"id":"default","tenantId":"tenant-1","matchType":"default","actionType":"direct","enabled":true}]}`); err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, func() string { return "node-1" }, nil)
	reporter := recordingSessionReporter{sessions: make(chan domain.ProxySessionMetric, 1)}
	server.SetProxySessionReporter(reporter)
	proxy := httptest.NewServer(server)
	defer proxy.Close()
	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxyConn, err := net.Dial("tcp", proxyURL.Host)
	if err != nil {
		t.Fatal(err)
	}
	defer proxyConn.Close()
	targetURL := "ws" + origin.URL[len("http"):] + "/terminal"
	if _, err := fmt.Fprintf(proxyConn, "GET %s HTTP/1.1\r\nHost: %s\r\nConnection: Upgrade\r\nUpgrade: websocket\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n", targetURL, strings.TrimPrefix(origin.URL, "http://")); err != nil {
		t.Fatal(err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(proxyConn), nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	session := receiveSession(t, reporter.sessions)
	if session.TenantID != "tenant-1" || session.Status != domain.ProxySessionStatusOK || session.Protocol != domain.ProxySessionProtocolHTTP || session.MatchedRuleID != "default" || session.PolicyRevision != "test" {
		t.Fatalf("session = %+v", session)
	}
}

func TestTokenValidationCache(t *testing.T) {
	store := policystore.New("")
	payload, err := json.Marshal(policystore.Snapshot{
		RouteRules: []domain.RouteRule{
			{ID: "default", MatchType: domain.MatchTypeDefault, ActionType: domain.ActionTypeDirect, Enabled: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update("test", string(payload)); err != nil {
		t.Fatal(err)
	}
	validator := &recordingTokenValidator{valid: true, expiresAt: time.Now().UTC().Add(time.Hour)}
	server, err := NewServerWithOptions(store, func() string { return "node-1" }, nil, "", AuthConfig{
		Validator: validator,
		CacheTTL:  time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "http://example.invalid/", nil)
		req.Header.Set("Proxy-Authorization", "Bearer cached-token")
		resp := httptest.NewRecorder()
		server.ServeHTTP(resp, req)
		if resp.Code != http.StatusBadGateway {
			t.Fatalf("status[%d] = %d", i, resp.Code)
		}
	}
	if len(validator.validations) != 1 {
		t.Fatalf("validation count = %d", len(validator.validations))
	}
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func receiveSession(t *testing.T, sessions <-chan domain.ProxySessionMetric) domain.ProxySessionMetric {
	t.Helper()
	select {
	case session := <-sessions:
		return session
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for proxy session metric")
	}
	return domain.ProxySessionMetric{}
}
