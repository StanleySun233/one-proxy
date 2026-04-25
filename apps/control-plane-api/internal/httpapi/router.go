package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/control-plane-api/internal/config"
)

type Router struct {
	mux *http.ServeMux
}

func NewRouter(cfg config.Config) http.Handler {
	router := &Router{mux: http.NewServeMux()}
	router.routes(cfg)
	return router.mux
}

func (r *Router) routes(cfg config.Config) {
	r.mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"httpAddr": cfg.HTTPAddr,
			"sqlitePath": cfg.SQLitePath,
		})
	})

	r.mux.HandleFunc("/api/v1/overview", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"nodes": map[string]int{
				"healthy": 4,
				"degraded": 1,
			},
			"policies": map[string]string{
				"activeRevision": "rev-0007",
				"publishedAt": "2026-04-25T00:00:00Z",
			},
		})
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
