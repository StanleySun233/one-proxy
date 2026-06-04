package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestProxyAwareHandlerBypassesServeMuxRedirectForConnect(t *testing.T) {
	proxyCalled := false
	proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		proxyCalled = true
		w.WriteHeader(http.StatusNoContent)
	})
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := &http.Request{
		Method: http.MethodConnect,
		Host:   "www.baidu.com:443",
		URL:    &url.URL{Host: "www.baidu.com:443"},
		Header: http.Header{},
	}
	resp := httptest.NewRecorder()
	proxyAwareHandler(proxyHandler, mux).ServeHTTP(resp, req)

	if !proxyCalled {
		t.Fatal("proxy handler was not called")
	}
	if resp.Code != http.StatusNoContent {
		t.Fatalf("status = %d", resp.Code)
	}
}
