package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

func main() {
	webProxy := reverseProxy(mustParseURL(envOrDefault("EDGE_WEB_URL", "http://127.0.0.1:2885")))
	apiProxy := reverseProxy(mustParseURL(envOrDefault("EDGE_API_URL", "http://127.0.0.1:2887")))
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if routeToAPI(req.URL.Path) {
			apiProxy.ServeHTTP(w, req)
			return
		}
		webProxy.ServeHTTP(w, req)
	})
	addr := envOrDefault("EDGE_ADDR", ":2886")
	log.Printf("panel edge listening addr=%s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}

func routeToAPI(path string) bool {
	if path == "/healthz" {
		return true
	}
	if path == "/api/node-release-tags" || strings.HasPrefix(path, "/api/node-release-tags/") {
		return false
	}
	return path == "/api" || strings.HasPrefix(path, "/api/")
}

func reverseProxy(target *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		log.Printf("edge proxy error path=%s err=%v", req.URL.Path, err)
		http.Error(w, "upstream_unavailable", http.StatusBadGateway)
	}
	return proxy
}

func mustParseURL(raw string) *url.URL {
	parsed, err := url.Parse(raw)
	if err != nil {
		log.Fatalf("invalid upstream url %q: %v", raw, err)
	}
	return parsed
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
