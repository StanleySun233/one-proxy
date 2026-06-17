package main

import (
	"net"
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

func TestParseBytesOrDefault(t *testing.T) {
	if parseBytesOrDefault("512mb", 1) != 512*1024*1024 {
		t.Fatalf("512mb parsed incorrectly")
	}
	if parseBytesOrDefault("2gb", 1) != 2*1024*1024*1024 {
		t.Fatalf("2gb parsed incorrectly")
	}
	if parseBytesOrDefault("", 99) != 99 {
		t.Fatalf("fallback not used")
	}
}

func TestPublicHostFromAddrsPrefersPublicIPv4(t *testing.T) {
	got := publicHostFromAddrs([]net.Addr{
		ipNetAddr("10.10.0.102"),
		ipNetAddr("103.214.172.211"),
		ipNetAddr("172.18.0.1"),
	})
	if got != "103.214.172.211" {
		t.Fatalf("public host = %q", got)
	}
}

func TestPublicHostFromAddrsSkipsNonPublicIPv4(t *testing.T) {
	got := publicHostFromAddrs([]net.Addr{
		ipNetAddr("127.0.0.1"),
		ipNetAddr("10.0.0.8"),
		ipNetAddr("172.18.0.1"),
		ipNetAddr("192.168.1.8"),
		ipNetAddr("100.64.0.1"),
	})
	if got != "" {
		t.Fatalf("public host = %q", got)
	}
}

func ipNetAddr(value string) net.Addr {
	return &net.IPNet{IP: net.ParseIP(value), Mask: net.CIDRMask(32, 32)}
}
