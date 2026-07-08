package proxy

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/responsecache"
	"github.com/gorilla/websocket"
)

type recordingTokenValidator struct {
	validations     []TokenValidationRequest
	authentications []string
	valid           bool
	expiresAt       time.Time
	allowLocalProxy bool
	tenantID        string
}

type recordingSessionReporter struct {
	sessions chan domain.ProxySessionMetric
}

type scriptedStreamOpener struct {
	attempts int
}

type cacheFallbackStreamOpener struct {
	attempts int
	fail     bool
}

type fullBodyThenErrorReadCloser struct {
	data []byte
	done bool
}

func (r *fullBodyThenErrorReadCloser) Read(p []byte) (int, error) {
	if r.done {
		return 0, errors.New("websocket close 1006")
	}
	r.done = true
	return copy(p, r.data), errors.New("websocket close 1006")
}

func (r *fullBodyThenErrorReadCloser) Close() error {
	return nil
}

func (s *scriptedStreamOpener) HasDirectPeer(string) bool {
	return true
}

func (s *cacheFallbackStreamOpener) HasDirectPeer(string) bool {
	return true
}

func (s *scriptedStreamOpener) OpenDirectStream(_ context.Context, _ domain.Node, _ []string, _ string, _ int) (net.Conn, error) {
	client, server := net.Pipe()
	s.attempts += 1
	attempt := s.attempts
	go func() {
		defer server.Close()
		req, err := http.ReadRequest(bufio.NewReader(server))
		if err != nil {
			return
		}
		_, _ = io.ReadAll(req.Body)
		if attempt == 1 {
			_, _ = server.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 6\r\n\r\nbad"))
			return
		}
		_, _ = server.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 6\r\n\r\nloaded"))
	}()
	return client, nil
}

func (s *cacheFallbackStreamOpener) OpenDirectStream(_ context.Context, _ domain.Node, _ []string, _ string, _ int) (net.Conn, error) {
	s.attempts += 1
	if s.fail {
		return nil, errors.New("stream_down")
	}
	client, server := net.Pipe()
	go func() {
		defer server.Close()
		req, err := http.ReadRequest(bufio.NewReader(server))
		if err != nil {
			return
		}
		_, _ = io.ReadAll(req.Body)
		_, _ = server.Write([]byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 6\r\n\r\nloaded"))
	}()
	return client, nil
}

func (v *recordingTokenValidator) ValidateProxyToken(_ context.Context, request TokenValidationRequest) (TokenValidation, error) {
	v.validations = append(v.validations, request)
	return TokenValidation{Valid: v.valid, ExpiresAt: v.expiresAt, AllowLocalProxy: v.allowLocalProxy, TenantID: v.tenantID}, nil
}

func (v *recordingTokenValidator) AuthenticateProxyToken(_ context.Context, tokenHash string) (TokenValidation, error) {
	v.authentications = append(v.authentications, tokenHash)
	if !v.allowLocalProxy {
		return TokenValidation{}, nil
	}
	return TokenValidation{Valid: v.valid, ExpiresAt: v.expiresAt, AllowLocalProxy: v.allowLocalProxy, TenantID: v.tenantID}, nil
}

func (r recordingSessionReporter) ReportProxySessions(_ context.Context, input domain.ProxySessionMetricsInput) error {
	for _, session := range input.Sessions {
		r.sessions <- session
	}
	return nil
}

func newAuthenticatedForwardServer(t *testing.T, store *policystore.Store) *Server {
	t.Helper()
	server, err := NewServerWithOptions(store, func() string { return "node-1" }, nil, "", AuthConfig{
		Validator: validRecordingTokenValidator(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func newAuthenticatedReverseServer(t *testing.T, reverseTargetURL string) *Server {
	t.Helper()
	server, err := NewServerWithOptions(policystore.New(""), func() string { return "node-1" }, nil, reverseTargetURL, AuthConfig{
		Validator: validRecordingTokenValidator(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func validRecordingTokenValidator() *recordingTokenValidator {
	return &recordingTokenValidator{valid: true, expiresAt: time.Now().UTC().Add(time.Hour)}
}

func setForwardProxyToken(req *http.Request) {
	req.Header.Set("Proxy-Authorization", "Bearer proxy-token")
}

func setReverseProxyToken(req *http.Request) {
	req.Header.Set(reverseHeaderName, "reverse-token")
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
	server := newAuthenticatedForwardServer(t, store)

	req := httptest.NewRequest(http.MethodGet, origin.URL+"/health", nil)
	setForwardProxyToken(req)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)

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
	server := newAuthenticatedForwardServer(t, store)
	req := httptest.NewRequest(http.MethodGet, origin.URL+"/static/js/app.js", nil)
	setForwardProxyToken(req)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)

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

func TestForwardDirectDoesNotRetryPostRequest(t *testing.T) {
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
	server := newAuthenticatedForwardServer(t, store)
	req := httptest.NewRequest(http.MethodPost, origin.URL+"/api/save", strings.NewReader("upload"))
	setForwardProxyToken(req)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %q", resp.Code, resp.Body.String())
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d", attempts)
	}
}

func TestForwardDirectStreamsContentLengthMismatch(t *testing.T) {
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
	server := newAuthenticatedForwardServer(t, store)
	req := httptest.NewRequest(http.MethodGet, origin.URL+"/api/scenario/id/track", nil)
	setForwardProxyToken(req)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", resp.Code, resp.Body.String())
	}
	if resp.Body.String() != "bad" {
		t.Fatalf("body = %q", resp.Body.String())
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d", attempts)
	}
}

func TestForwardDirectStreamsNDJSONWithResponseCache(t *testing.T) {
	attempts := 0
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		attempts += 1
		w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
		w.Header().Set("Content-Length", "6")
		_, _ = w.Write([]byte("bad"))
	}))
	defer origin.Close()

	store := policystore.New("")
	if err := store.Update("test", `{"nodes":[],"links":[],"chains":[],"routeRules":[{"id":"default","matchType":"default","actionType":"direct","enabled":true}]}`); err != nil {
		t.Fatal(err)
	}
	server := newAuthenticatedForwardServer(t, store)
	cache, err := responsecache.New(responsecache.Config{Dir: t.TempDir(), TTL: time.Hour, MemoryMaxBytes: 1024, DiskMaxBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	server.SetResponseCache(cache)

	req := httptest.NewRequest(http.MethodGet, origin.URL+"/api/cognitive/ais/frame/tiles/stream", nil)
	setForwardProxyToken(req)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", resp.Code, resp.Body.String())
	}
	if resp.Body.String() != "bad" {
		t.Fatalf("body = %q", resp.Body.String())
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d", attempts)
	}
}

func TestForwardDirectDoesNotCacheNDJSONStream(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte("{\"type\":\"complete\"}\n"))
	}))
	store := policystore.New("")
	if err := store.Update("test", `{"nodes":[],"links":[],"chains":[],"routeRules":[{"id":"default","matchType":"default","actionType":"direct","enabled":true}]}`); err != nil {
		t.Fatal(err)
	}
	server := newAuthenticatedForwardServer(t, store)
	cache, err := responsecache.New(responsecache.Config{Dir: t.TempDir(), TTL: time.Hour, MemoryMaxBytes: 1024, DiskMaxBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	server.SetResponseCache(cache)

	req := httptest.NewRequest(http.MethodGet, origin.URL+"/api/cognitive/ais/frame/tiles/stream", nil)
	setForwardProxyToken(req)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || resp.Body.String() != "{\"type\":\"complete\"}\n" {
		t.Fatalf("first response status=%d body=%q", resp.Code, resp.Body.String())
	}
	origin.Close()

	req = httptest.NewRequest(http.MethodGet, origin.URL+"/api/cognitive/ais/frame/tiles/stream", nil)
	setForwardProxyToken(req)
	resp = httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadGateway {
		t.Fatalf("stream should not use cache, status=%d body=%q", resp.Code, resp.Body.String())
	}
	if resp.Header().Get(responseCacheHeader) != "" {
		t.Fatalf("cache header = %q", resp.Header().Get(responseCacheHeader))
	}
}

func TestForwardDirectStreamsUnknownLengthWithResponseCache(t *testing.T) {
	attempts := 0
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		attempts += 1
		w.Header().Set("Content-Type", "application/json")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("missing flusher")
		}
		flusher.Flush()
		_, _ = w.Write([]byte("{\"chunk\":true}\n"))
	}))
	store := policystore.New("")
	if err := store.Update("test", `{"nodes":[],"links":[],"chains":[],"routeRules":[{"id":"default","matchType":"default","actionType":"direct","enabled":true}]}`); err != nil {
		t.Fatal(err)
	}
	server := newAuthenticatedForwardServer(t, store)
	cache, err := responsecache.New(responsecache.Config{Dir: t.TempDir(), TTL: time.Hour, MemoryMaxBytes: 1024, DiskMaxBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	server.SetResponseCache(cache)

	req := httptest.NewRequest(http.MethodGet, origin.URL+"/api/custom/stream", nil)
	setForwardProxyToken(req)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || resp.Body.String() != "{\"chunk\":true}\n" {
		t.Fatalf("first response status=%d body=%q", resp.Code, resp.Body.String())
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d", attempts)
	}
	origin.Close()

	req = httptest.NewRequest(http.MethodGet, origin.URL+"/api/custom/stream", nil)
	setForwardProxyToken(req)
	resp = httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadGateway {
		t.Fatalf("unknown-length response should not use cache, status=%d body=%q", resp.Code, resp.Body.String())
	}
	if resp.Header().Get(responseCacheHeader) != "" {
		t.Fatalf("cache header = %q", resp.Header().Get(responseCacheHeader))
	}
}

func TestForwardDirectServesCacheOnlyAfterForwardError(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", "6")
		_, _ = w.Write([]byte("cached"))
	}))
	store := policystore.New("")
	if err := store.Update("test", `{"nodes":[],"links":[],"chains":[],"routeRules":[{"id":"default","matchType":"default","actionType":"direct","enabled":true}]}`); err != nil {
		t.Fatal(err)
	}
	server := newAuthenticatedForwardServer(t, store)
	cache, err := responsecache.New(responsecache.Config{Dir: t.TempDir(), TTL: time.Hour, MemoryMaxBytes: 1024, DiskMaxBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	server.SetResponseCache(cache)

	req := httptest.NewRequest(http.MethodGet, origin.URL+"/api/data", nil)
	setForwardProxyToken(req)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || resp.Body.String() != "cached" {
		t.Fatalf("first response status=%d body=%q", resp.Code, resp.Body.String())
	}
	if resp.Header().Get(responseCacheHeader) != "" {
		t.Fatalf("first response cache header = %q", resp.Header().Get(responseCacheHeader))
	}
	origin.Close()

	req = httptest.NewRequest(http.MethodGet, origin.URL+"/api/data", nil)
	setForwardProxyToken(req)
	resp = httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("cached status = %d body=%q", resp.Code, resp.Body.String())
	}
	if resp.Body.String() != "cached" {
		t.Fatalf("cached body = %q", resp.Body.String())
	}
	if resp.Header().Get(responseCacheHeader) != "stale" {
		t.Fatalf("cache header = %q", resp.Header().Get(responseCacheHeader))
	}
}

func TestForwardDirectDoesNotServeCacheForPostError(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("saved"))
	}))
	store := policystore.New("")
	if err := store.Update("test", `{"nodes":[],"links":[],"chains":[],"routeRules":[{"id":"default","matchType":"default","actionType":"direct","enabled":true}]}`); err != nil {
		t.Fatal(err)
	}
	server := newAuthenticatedForwardServer(t, store)
	cache, err := responsecache.New(responsecache.Config{Dir: t.TempDir(), TTL: time.Hour, MemoryMaxBytes: 1024, DiskMaxBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	server.SetResponseCache(cache)

	req := httptest.NewRequest(http.MethodPost, origin.URL+"/api/save", strings.NewReader("upload"))
	setForwardProxyToken(req)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("first status = %d body=%q", resp.Code, resp.Body.String())
	}
	origin.Close()

	req = httptest.NewRequest(http.MethodPost, origin.URL+"/api/save", strings.NewReader("upload"))
	setForwardProxyToken(req)
	resp = httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadGateway {
		t.Fatalf("post should not use cache, status=%d body=%q", resp.Code, resp.Body.String())
	}
}

func TestForwardDirectDoesNotServeCacheForUpstreamBadGatewayResponse(t *testing.T) {
	fail := false
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if fail {
			http.Error(w, "upstream_bad_gateway", http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte("cached"))
	}))
	defer origin.Close()
	store := policystore.New("")
	if err := store.Update("test", `{"nodes":[],"links":[],"chains":[],"routeRules":[{"id":"default","matchType":"default","actionType":"direct","enabled":true}]}`); err != nil {
		t.Fatal(err)
	}
	server := newAuthenticatedForwardServer(t, store)
	cache, err := responsecache.New(responsecache.Config{Dir: t.TempDir(), TTL: time.Hour, MemoryMaxBytes: 1024, DiskMaxBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	server.SetResponseCache(cache)

	req := httptest.NewRequest(http.MethodGet, origin.URL+"/api/data", nil)
	setForwardProxyToken(req)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("first status = %d body=%q", resp.Code, resp.Body.String())
	}
	fail = true
	req = httptest.NewRequest(http.MethodGet, origin.URL+"/api/data", nil)
	setForwardProxyToken(req)
	resp = httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadGateway {
		t.Fatalf("upstream status should pass through, got=%d body=%q", resp.Code, resp.Body.String())
	}
	if resp.Header().Get(responseCacheHeader) != "" {
		t.Fatalf("cache header = %q", resp.Header().Get(responseCacheHeader))
	}
}

func TestCachedForwardResponseAnnotatesHTML(t *testing.T) {
	storedAt := time.Date(2026, 6, 16, 10, 30, 0, 0, time.UTC)
	resp := cachedForwardResponse(responsecache.Entry{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/html; charset=utf-8"},
		},
		Body:     []byte("<html><body><main>cached</main></body></html>"),
		StoredAt: storedAt,
	})

	body := string(resp.body)
	if resp.header.Get(responseCacheHeader) != "stale" {
		t.Fatalf("cache header = %q", resp.header.Get(responseCacheHeader))
	}
	if resp.header.Get(responseCacheStoredAtHeader) != storedAt.Format(time.RFC3339) {
		t.Fatalf("cache stored-at header = %q", resp.header.Get(responseCacheStoredAtHeader))
	}
	if resp.header.Get("Content-Length") != fmt.Sprintf("%d", len(resp.body)) {
		t.Fatalf("content-length = %q body=%d", resp.header.Get("Content-Length"), len(resp.body))
	}
	if !strings.Contains(body, `id="one-proxy-cache-banner"`) || !strings.Contains(body, storedAt.Format(time.RFC3339)) {
		t.Fatalf("cached html banner missing: %q", body)
	}
}

func TestForwardChainViaStreamStreamsContentLengthMismatch(t *testing.T) {
	store := policystore.New("")
	payload, err := json.Marshal(policystore.Snapshot{
		Nodes: []domain.Node{
			{ID: "node-2", Name: "next-hop", Enabled: true, Status: "healthy"},
		},
		Chains: []domain.Chain{
			{ID: "chain-1", Name: "chain", Enabled: true, Hops: []string{"node-1", "node-2"}},
		},
		RouteRules: []domain.RouteRule{
			{ID: "default", MatchType: domain.MatchTypeDefault, ActionType: domain.ActionTypeChain, ChainID: "chain-1", Enabled: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update("test", string(payload)); err != nil {
		t.Fatal(err)
	}
	opener := &scriptedStreamOpener{}
	server := newAuthenticatedForwardServer(t, store)
	server.SetDirectStreamOpener(opener)

	req := httptest.NewRequest(http.MethodGet, "http://172.20.116.91:12335/api/scenario/id/track", nil)
	setForwardProxyToken(req)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", resp.Code, resp.Body.String())
	}
	if resp.Body.String() != "bad" {
		t.Fatalf("body = %q", resp.Body.String())
	}
	if opener.attempts != 1 {
		t.Fatalf("attempts = %d", opener.attempts)
	}
}

func TestForwardChainViaStreamServesCacheAfterStreamOpenError(t *testing.T) {
	store := policystore.New("")
	payload, err := json.Marshal(policystore.Snapshot{
		Nodes: []domain.Node{
			{ID: "node-2", Name: "next-hop", Enabled: true, Status: "healthy"},
		},
		Chains: []domain.Chain{
			{ID: "chain-1", Name: "chain", Enabled: true, Hops: []string{"node-1", "node-2"}},
		},
		RouteRules: []domain.RouteRule{
			{ID: "default", MatchType: domain.MatchTypeDefault, ActionType: domain.ActionTypeChain, ChainID: "chain-1", Enabled: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update("test", string(payload)); err != nil {
		t.Fatal(err)
	}
	opener := &cacheFallbackStreamOpener{}
	server := newAuthenticatedForwardServer(t, store)
	server.SetDirectStreamOpener(opener)
	cache, err := responsecache.New(responsecache.Config{Dir: t.TempDir(), TTL: time.Hour, MemoryMaxBytes: 1024, DiskMaxBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	server.SetResponseCache(cache)

	req := httptest.NewRequest(http.MethodGet, "http://172.20.116.91:12335/api/scenario/id/track", nil)
	setForwardProxyToken(req)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || resp.Body.String() != "loaded" {
		t.Fatalf("first response status=%d body=%q", resp.Code, resp.Body.String())
	}
	opener.fail = true
	req = httptest.NewRequest(http.MethodGet, "http://172.20.116.91:12335/api/scenario/id/track", nil)
	setForwardProxyToken(req)
	resp = httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || resp.Body.String() != "loaded" {
		t.Fatalf("cached response status=%d body=%q", resp.Code, resp.Body.String())
	}
	if resp.Header().Get(responseCacheHeader) != "stale" {
		t.Fatalf("cache header = %q", resp.Header().Get(responseCacheHeader))
	}
}

func TestReadForwardResponseAcceptsCompleteBodyWithTransportClose(t *testing.T) {
	resp, err := readForwardResponse(&http.Response{
		StatusCode:    http.StatusOK,
		Header:        http.Header{"Content-Type": []string{"application/javascript"}},
		ContentLength: 6,
		Body:          &fullBodyThenErrorReadCloser{data: []byte("loaded")},
	}, http.MethodGet)
	if err != nil {
		t.Fatal(err)
	}
	if resp.statusCode != http.StatusOK || string(resp.body) != "loaded" {
		t.Fatalf("response status=%d body=%q", resp.statusCode, resp.body)
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
	server := newAuthenticatedForwardServer(t, store)
	server.SetProxySessionReporter(reporter)

	req := httptest.NewRequest(http.MethodPost, origin.URL+"/metrics", strings.NewReader("upload"))
	setForwardProxyToken(req)
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
	server := newAuthenticatedForwardServer(t, store)
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
	if _, err := fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Authorization: Bearer proxy-token\r\n\r\n", targetAddr, targetAddr); err != nil {
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

func TestForwardProxyRejectsNilValidator(t *testing.T) {
	store := policystore.New("")
	if err := store.Update("test", `{"nodes":[],"links":[],"chains":[],"routeRules":[{"id":"default","matchType":"default","actionType":"direct","enabled":true}]}`); err != nil {
		t.Fatal(err)
	}
	server := NewServer(store, func() string { return "node-1" }, nil)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	setForwardProxyToken(req)
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
	if len(validator.validations) != 1 || validator.validations[0].TokenHash != expectedHash {
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
	if len(validator.validations) != 2 || validator.validations[1].TokenHash != expectedHash {
		t.Fatalf("validation hashes = %v, want second %s", validator.validations, expectedHash)
	}
}

func TestForwardProxyTokenValidationIncludesRouteContext(t *testing.T) {
	store := policystore.New("")
	payload, err := json.Marshal(policystore.Snapshot{
		Nodes: []domain.Node{
			{ID: "edge", Enabled: true},
			{ID: "target", Enabled: true},
		},
		Chains: []domain.Chain{
			{ID: "chain-1", Enabled: true, Hops: []string{"edge", "target"}},
		},
		RouteRules: []domain.RouteRule{
			{ID: "route-1", MatchType: domain.MatchTypeDefault, ActionType: domain.ActionTypeChain, ChainID: "chain-1", AccessPathID: "path-1", Enabled: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update("test", string(payload)); err != nil {
		t.Fatal(err)
	}
	validator := &recordingTokenValidator{valid: true, expiresAt: time.Now().UTC().Add(time.Hour)}
	server, err := NewServerWithOptions(store, func() string { return "edge" }, nil, "", AuthConfig{
		Validator: validator,
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.invalid:2333/lab", nil)
	req.Header.Set("Proxy-Authorization", "Bearer proxy-token")
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if len(validator.validations) != 1 {
		t.Fatalf("validation count = %d", len(validator.validations))
	}
	got := validator.validations[0]
	if got.TokenHash != sha256Hex("proxy-token") || got.AccessPathID != "path-1" || got.RouteID != "route-1" {
		t.Fatalf("validation identity = %+v", got)
	}
	if got.TargetHost != "example.invalid" || got.TargetPort != 2333 || got.Protocol != "http" {
		t.Fatalf("validation target = %+v", got)
	}
}

func TestForwardProxyDeniesNoMatchAfterTokenAuthentication(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		t.Fatalf("origin should not receive no-match request path=%q", req.URL.Path)
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
	if resp.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %q", resp.Code, resp.Body.String())
	}
	session := receiveSession(t, reporter.sessions)
	if session.TenantID != "tenant-1" || session.DecisionSource != "default" || session.MatchedAction != domain.ActionTypeDeny || session.Status != domain.ProxySessionStatusError || session.ErrorCode != proxyErrorRouteNotFound {
		t.Fatalf("session = %+v", session)
	}
}

func TestForwardProxyNoMatchUsesAuthenticateOnly(t *testing.T) {
	store := policystore.New("")
	if err := store.Update("test", `{"nodes":[],"links":[],"chains":[],"routeRules":[]}`); err != nil {
		t.Fatal(err)
	}
	validator := &recordingTokenValidator{valid: true, expiresAt: time.Now().UTC().Add(time.Hour), allowLocalProxy: true, tenantID: "tenant-1"}
	server, err := NewServerWithOptions(store, func() string { return "node-1" }, nil, "", AuthConfig{
		Validator: validator,
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	setForwardProxyToken(req)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("status = %d", resp.Code)
	}
	if len(validator.validations) != 0 || len(validator.authentications) != 1 || validator.authentications[0] != sha256Hex("proxy-token") {
		t.Fatalf("validations=%+v authentications=%+v", validator.validations, validator.authentications)
	}
}

func TestForwardRetryableBodyOverLimitStreamsOnce(t *testing.T) {
	attempts := 0
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		attempts += 1
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "larger-than-limit" {
			t.Fatalf("body = %q", body)
		}
		http.Error(w, "bad_gateway", http.StatusBadGateway)
	}))
	defer origin.Close()

	store := policystore.New("")
	payload, err := json.Marshal(policystore.Snapshot{
		RouteRules: []domain.RouteRule{
			{ID: "route-1", MatchType: domain.MatchTypeDefault, MatchValue: "*", ActionType: domain.ActionTypeDirect, DestinationScope: "scope-a", Enabled: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update("test", string(payload)); err != nil {
		t.Fatal(err)
	}
	server := newAuthenticatedForwardServer(t, store)
	server.SetRetryBodyMaxBytes(4)
	req := httptest.NewRequest(http.MethodGet, origin.URL+"/upload", strings.NewReader("larger-than-limit"))
	setForwardProxyToken(req)
	resp := httptest.NewRecorder()
	server.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadGateway {
		t.Fatalf("status = %d body=%q", resp.Code, resp.Body.String())
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d", attempts)
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
	if resp.Code != http.StatusProxyAuthRequired {
		t.Fatalf("status = %d", resp.Code)
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

func TestMatchDomainSuffixSupportsRootAndSubdomains(t *testing.T) {
	for _, value := range []string{".openai.com", "*.openai.com"} {
		for _, target := range []string{"http://openai.com/", "http://api.openai.com/"} {
			req := httptest.NewRequest(http.MethodGet, target, nil)
			match := Match(policystore.Snapshot{
				RouteRules: []domain.RouteRule{
					{
						ID:         "suffix",
						MatchType:  domain.MatchTypeDomainSuffix,
						MatchValue: value,
						ActionType: domain.ActionTypeDirect,
						Enabled:    true,
					},
				},
			}, req)

			if !match.Found {
				t.Fatalf("suffix %q did not match %q", value, target)
			}
			if match.Rule.ID != "suffix" {
				t.Fatalf("rule id = %q", match.Rule.ID)
			}
		}
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

	server := newAuthenticatedForwardServer(t, store)
	reporter := recordingSessionReporter{sessions: make(chan domain.ProxySessionMetric, 1)}
	server.SetProxySessionReporter(reporter)
	proxy := httptest.NewServer(server)
	defer proxy.Close()

	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxyURL.User = url.UserPassword("token", "proxy-token")
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
	server := newAuthenticatedForwardServer(t, store)
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
	if _, err := fmt.Fprintf(proxyConn, "GET %s HTTP/1.1\r\nHost: %s\r\nConnection: Upgrade\r\nUpgrade: websocket\r\nProxy-Authorization: Bearer proxy-token\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n", targetURL, strings.TrimPrefix(origin.URL, "http://")); err != nil {
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
