package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/policystore"
	"github.com/gorilla/websocket"
)

func TestReverseTargetForwardsOriginFormHTTP(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Host != req.URL.Host && req.Host == "proxy.local" {
			t.Fatalf("request was forwarded with proxy host")
		}
		if req.URL.Path != "/lab/api" {
			t.Fatalf("unexpected path %q", req.URL.Path)
		}
		_, _ = w.Write([]byte(req.Header.Get("X-Forwarded-Host")))
	}))
	defer origin.Close()

	server, err := NewServerWithReverseTarget(policystore.New(""), func() string { return "node-1" }, nil, origin.URL+"/lab")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://proxy.local/api", nil)
	req.URL.Scheme = ""
	req.URL.Host = ""
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", resp.Code, resp.Body.String())
	}
	if resp.Body.String() != "proxy.local" {
		t.Fatalf("forwarded host = %q", resp.Body.String())
	}
}

func TestReverseTargetRequiresAuthorizationToken(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer origin.Close()
	validator := &recordingTokenValidator{valid: true, expiresAt: time.Now().UTC().Add(time.Hour)}
	server, err := NewServerWithOptions(policystore.New(""), func() string { return "node-1" }, nil, origin.URL, AuthConfig{
		Validator: validator,
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://proxy.local/api", nil)
	req.URL.Scheme = ""
	req.URL.Host = ""
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", resp.Code)
	}
	if resp.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("missing challenge")
	}
	req.Header.Set("Authorization", "Bearer reverse-token")
	resp = httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("authorized status = %d body=%q", resp.Code, resp.Body.String())
	}
	if len(validator.validations) != 1 || validator.validations[0] != sha256Hex("reverse-token") {
		t.Fatalf("validation hashes = %v", validator.validations)
	}
}

func TestReverseQueryTokenSetsCookieAndStripsCredentials(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Query().Get(reverseQueryTokenKey) != "" {
			t.Fatal("proxy token leaked in query")
		}
		if req.Header.Get("Authorization") != "" {
			t.Fatal("authorization leaked")
		}
		if _, err := req.Cookie(reverseCookieName); err == nil {
			t.Fatal("proxy cookie leaked")
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer origin.Close()
	validator := &recordingTokenValidator{valid: true, expiresAt: time.Now().UTC().Add(time.Hour)}
	server, err := NewServerWithOptions(policystore.New(""), func() string { return "node-1" }, nil, origin.URL, AuthConfig{
		Validator: validator,
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://proxy.local/api?proxy_token=query-token&x=1", nil)
	req.URL.Scheme = ""
	req.URL.Host = ""
	req.AddCookie(&http.Cookie{Name: reverseCookieName, Value: "cookie-token"})
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%q", resp.Code, resp.Body.String())
	}
	cookies := resp.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != reverseCookieName {
		t.Fatalf("cookies = %v", cookies)
	}
	if len(validator.validations) != 1 || validator.validations[0] != sha256Hex("query-token") {
		t.Fatalf("validation hashes = %v", validator.validations)
	}
}

func TestReverseTargetForwardsOriginFormWebSocket(t *testing.T) {
	upgrader := websocket.Upgrader{}
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/terminals/websocket/1" {
			t.Fatalf("unexpected path %q", req.URL.Path)
		}
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

	server, err := NewServerWithReverseTarget(policystore.New(""), func() string { return "node-1" }, nil, origin.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxy := httptest.NewServer(server)
	defer proxy.Close()

	targetURL := "ws" + proxy.URL[len("http"):] + "/terminals/websocket/1"
	conn, _, err := websocket.DefaultDialer.Dial(targetURL, http.Header{"Origin": []string{"http://proxy.local"}})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.TextMessage, []byte("terminal")); err != nil {
		t.Fatal(err)
	}
	_, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != "terminal" {
		t.Fatalf("payload = %q", payload)
	}
}
