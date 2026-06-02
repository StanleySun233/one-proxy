package httpapi

import (
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/network"
)

func (r *Router) handleSetupStatus(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{"configured": r.service.IsInitialized()})
}

func (r *Router) handleHealthz(cfg HTTPConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeSuccess(w, http.StatusOK, map[string]any{
			"status":    "ok",
			"httpAddr":  cfg.HTTPAddr,
			"dbBackend": cfg.DBBackend,
			"localIPs":  network.LocalIPs(),
		})
	}
}
