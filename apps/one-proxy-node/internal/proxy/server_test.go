package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/policystore"
	"github.com/gorilla/websocket"
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
