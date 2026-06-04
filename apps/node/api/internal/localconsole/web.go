package localconsole

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var consoleRoutes = map[string]string{
	"/console":             "Overview",
	"/console/login":       "Login",
	"/console/overview":    "Overview",
	"/console/health":      "Health",
	"/console/audit":       "Audit",
	"/console/policy":      "Policy",
	"/console/diagnostics": "Diagnostics",
}

func ConsoleRoute(path string) bool {
	_, ok := consoleRoutes[path]
	return ok
}

func StaticAssetRoute(path string) bool {
	return strings.HasPrefix(path, "/console/assets/") || path == "/console/favicon.ico" || path == "/console/robots.txt"
}

func WebHandler(webRoot string, authenticated func(*http.Request) bool) http.Handler {
	files := http.FileServer(http.Dir(webRoot))
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet && req.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if req.URL.Path == "/console" {
			if authenticated(req) {
				http.Redirect(w, req, "/console/overview", http.StatusFound)
				return
			}
			http.Redirect(w, req, "/console/login", http.StatusFound)
			return
		}
		if StaticAssetRoute(req.URL.Path) && fileExists(webRoot, strings.TrimPrefix(req.URL.Path, "/console")) {
			http.StripPrefix("/console", files).ServeHTTP(w, req)
			return
		}
		if ConsoleRoute(req.URL.Path) {
			if fileExists(webRoot, "/index.html") {
				http.ServeFile(w, req, filepath.Join(webRoot, "index.html"))
				return
			}
			serveFallbackShell(w, consoleRoutes[req.URL.Path])
			return
		}
		http.NotFound(w, req)
	})
}

func fileExists(root string, path string) bool {
	cleaned := strings.TrimPrefix(filepath.Clean(path), string(filepath.Separator))
	info, err := os.Stat(filepath.Join(root, cleaned))
	return err == nil && !info.IsDir()
}

func serveFallbackShell(w http.ResponseWriter, title string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte("<!doctype html><html><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><title>One Proxy Node Console</title></head><body><main id=\"root\"><h1>One Proxy Node Console</h1><h2>" + title + "</h2></main></body></html>"))
}
