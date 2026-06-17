package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
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

	server := newAuthenticatedReverseServer(t, origin.URL+"/lab")
	req := httptest.NewRequest(http.MethodGet, "http://proxy.local/api", nil)
	req.URL.Scheme = ""
	req.URL.Host = ""
	setReverseProxyToken(req)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", resp.Code, resp.Body.String())
	}
	if resp.Body.String() != "proxy.local" {
		t.Fatalf("forwarded host = %q", resp.Body.String())
	}
}

func TestReverseTargetRejectsNilValidator(t *testing.T) {
	server, err := NewServerWithReverseTarget(policystore.New(""), func() string { return "node-1" }, nil, "http://example.com")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://proxy.local/api", nil)
	req.URL.Scheme = ""
	req.URL.Host = ""
	setReverseProxyToken(req)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", resp.Code)
	}
	if resp.Header().Get("X-One-Proxy-Authenticate") == "" {
		t.Fatal("missing challenge")
	}
}

func TestReverseTargetRequiresOneProxyTokenHeader(t *testing.T) {
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
	if resp.Header().Get("X-One-Proxy-Authenticate") == "" {
		t.Fatal("missing challenge")
	}
	req.Header.Set(reverseHeaderName, "reverse-token")
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
		if req.Header.Get(reverseHeaderName) != "" {
			t.Fatal("one-proxy header leaked")
		}
		if req.Header.Get("Authorization") != "Bearer business-token" {
			t.Fatalf("authorization = %q", req.Header.Get("Authorization"))
		}
		if _, err := req.Cookie(reverseCookieName); err == nil {
			t.Fatal("proxy cookie leaked")
		}
		if req.URL.Query().Get("one_proxy_token") != "business-query" {
			t.Fatalf("business query = %q", req.URL.Query().Get("one_proxy_token"))
		}
		if cookie, err := req.Cookie("one_proxy_token"); err != nil || cookie.Value != "business-cookie" {
			t.Fatalf("business cookie = %v err=%v", cookie, err)
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
	req := httptest.NewRequest(http.MethodGet, "http://proxy.local/api?one_proxy_auth=query-token&one_proxy_token=business-query&x=1", nil)
	req.URL.Scheme = ""
	req.URL.Host = ""
	req.Header.Set("Authorization", "Bearer business-token")
	req.AddCookie(&http.Cookie{Name: reverseCookieName, Value: "cookie-token"})
	req.AddCookie(&http.Cookie{Name: "one_proxy_token", Value: "business-cookie"})
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

func TestReverseTargetReportsProxySessionMetrics(t *testing.T) {
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
	validator := &recordingTokenValidator{valid: true, expiresAt: time.Now().UTC().Add(time.Hour), tenantID: "tenant-1"}
	server, err := NewServerWithOptions(policystore.New(""), func() string { return "node-1" }, nil, origin.URL, AuthConfig{
		Validator: validator,
	})
	if err != nil {
		t.Fatal(err)
	}
	reporter := recordingSessionReporter{sessions: make(chan domain.ProxySessionMetric, 1)}
	server.SetProxySessionReporter(reporter)

	req := httptest.NewRequest(http.MethodPost, "http://proxy.local/api", strings.NewReader("upload"))
	req.URL.Scheme = ""
	req.URL.Host = ""
	req.Header.Set(reverseHeaderName, "reverse-token")
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%q", resp.Code, resp.Body.String())
	}

	session := receiveSession(t, reporter.sessions)
	if session.TenantID != "tenant-1" || session.NodeID != "node-1" {
		t.Fatalf("session identity = %+v", session)
	}
	if session.Protocol != domain.ProxySessionProtocolHTTP || session.UploadBytes != 6 || session.DownloadBytes != 5 || session.Status != domain.ProxySessionStatusOK {
		t.Fatalf("session metrics = %+v", session)
	}
}

func TestReverseTargetRetriesContentLengthMismatch(t *testing.T) {
	attempts := 0
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		attempts += 1
		if req.URL.Path != "/api/scenario/id/track" {
			t.Fatalf("unexpected path %q", req.URL.Path)
		}
		if attempts == 1 {
			w.Header().Set("Content-Length", "6")
			_, _ = w.Write([]byte("bad"))
			return
		}
		_, _ = w.Write([]byte("loaded"))
	}))
	defer origin.Close()

	server := newAuthenticatedReverseServer(t, origin.URL)
	req := httptest.NewRequest(http.MethodGet, "http://proxy.local/api/scenario/id/track", nil)
	req.URL.Scheme = ""
	req.URL.Host = ""
	setReverseProxyToken(req)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%q", resp.Code, resp.Body.String())
	}
	if resp.Body.String() != "loaded" {
		t.Fatalf("body = %q", resp.Body.String())
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d", attempts)
	}
}

func TestReverseTargetRetriesPostRequest(t *testing.T) {
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

	server := newAuthenticatedReverseServer(t, origin.URL)
	req := httptest.NewRequest(http.MethodPost, "http://proxy.local/api/save", strings.NewReader("upload"))
	req.URL.Scheme = ""
	req.URL.Host = ""
	setReverseProxyToken(req)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%q", resp.Code, resp.Body.String())
	}
	if resp.Body.String() != "saved" {
		t.Fatalf("body = %q", resp.Body.String())
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d", attempts)
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

	server := newAuthenticatedReverseServer(t, origin.URL)
	proxy := httptest.NewServer(server)
	defer proxy.Close()

	targetURL := "ws" + proxy.URL[len("http"):] + "/terminals/websocket/1"
	conn, _, err := websocket.DefaultDialer.Dial(targetURL, http.Header{
		"Origin":          []string{"http://proxy.local"},
		reverseHeaderName: []string{"reverse-token"},
	})
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
