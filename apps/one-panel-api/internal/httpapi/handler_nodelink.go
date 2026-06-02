package httpapi

import "net/http"

func (r *Router) handleNodeTransports(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	writeSuccess(w, http.StatusOK, r.service.NodeTransports())
}
