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

func TestNodeRouteSplitHandlerProxiesAbsoluteRootURL(t *testing.T) {
	consoleCalled := false
	proxyCalled := false
	consoleWeb := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		consoleCalled = true
		w.WriteHeader(http.StatusOK)
	})
	proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		proxyCalled = true
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "http://172.20.116.91:2333/", nil)
	resp := httptest.NewRecorder()

	nodeRouteSplitHandler(consoleWeb, proxyHandler).ServeHTTP(resp, req)

	if consoleCalled {
		t.Fatal("console handler was called")
	}
	if !proxyCalled {
		t.Fatal("proxy handler was not called")
	}
	if resp.Code != http.StatusNoContent {
		t.Fatalf("status = %d", resp.Code)
	}
}

func TestNodeRouteSplitHandlerProxiesLocalRoot(t *testing.T) {
	consoleCalled := false
	proxyCalled := false
	consoleWeb := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		consoleCalled = true
		w.WriteHeader(http.StatusOK)
	})
	proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		proxyCalled = true
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()

	nodeRouteSplitHandler(consoleWeb, proxyHandler).ServeHTTP(resp, req)

	if consoleCalled {
		t.Fatal("console handler was called")
	}
	if !proxyCalled {
		t.Fatal("proxy handler was not called")
	}
	if resp.Code != http.StatusNoContent {
		t.Fatalf("status = %d", resp.Code)
	}
}

func TestNodeRouteSplitHandlerServesConsoleForLocalConsole(t *testing.T) {
	consoleCalled := false
	proxyCalled := false
	consoleWeb := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		consoleCalled = true
		w.WriteHeader(http.StatusOK)
	})
	proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		proxyCalled = true
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/console", nil)
	resp := httptest.NewRecorder()

	nodeRouteSplitHandler(consoleWeb, proxyHandler).ServeHTTP(resp, req)

	if !consoleCalled {
		t.Fatal("console handler was not called")
	}
	if proxyCalled {
		t.Fatal("proxy handler was called")
	}
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d", resp.Code)
	}
}

func TestNodeRouteSplitHandlerProxiesAbsoluteConsoleURL(t *testing.T) {
	consoleCalled := false
	proxyCalled := false
	consoleWeb := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		consoleCalled = true
		w.WriteHeader(http.StatusOK)
	})
	proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		proxyCalled = true
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "http://172.20.116.91:2333/console", nil)
	resp := httptest.NewRecorder()

	nodeRouteSplitHandler(consoleWeb, proxyHandler).ServeHTTP(resp, req)

	if consoleCalled {
		t.Fatal("console handler was called")
	}
	if !proxyCalled {
		t.Fatal("proxy handler was not called")
	}
	if resp.Code != http.StatusNoContent {
		t.Fatalf("status = %d", resp.Code)
	}
}
