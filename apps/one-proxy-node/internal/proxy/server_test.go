package proxy

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/policystore"
	"github.com/gorilla/websocket"
)

type recordingTokenValidator struct {
	validations []string
	valid       bool
	expiresAt   time.Time
}

func (v *recordingTokenValidator) ValidateProxyToken(_ context.Context, tokenHash string) (TokenValidation, error) {
	v.validations = append(v.validations, tokenHash)
	return TokenValidation{Valid: v.valid, ExpiresAt: v.expiresAt}, nil
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

	proxy := httptest.NewServer(NewServer(store, func() string { return "node-1" }, nil))
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
