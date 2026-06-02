package linkhttpapi

import (
	"encoding/json"
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/httpctx"
)

func (r *Router) handleNodeLinks(w http.ResponseWriter, req *http.Request) {
	tenantCtx, ok := httpctx.TenantAuth(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	switch req.Method {
	case http.MethodGet:
		writeSuccess(w, http.StatusOK, r.service.NodeLinks(tenantCtx))
	case http.MethodPost:
		var payload domain.CreateNodeLinkInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.CreateNodeLink(tenantCtx, payload)
		if err != nil {
			writeServiceError(w, req, err, "create_failed")
			return
		}
		writeSuccess(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (r *Router) handleNodeLinkByID(w http.ResponseWriter, req *http.Request) {
	linkID := resourceID(req.URL.Path, "/api/v1/chains/node-links/")
	if linkID == "" {
		writeError(w, http.StatusBadRequest, "missing_node_link_id")
		return
	}
	tenantCtx, ok := httpctx.TenantAuth(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	switch req.Method {
	case http.MethodPatch:
		var payload domain.UpdateNodeLinkInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.UpdateNodeLink(tenantCtx, linkID, payload)
		if err != nil {
			writeServiceError(w, req, err, "update_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := r.service.DeleteNodeLink(tenantCtx, linkID); err != nil {
			writeServiceError(w, req, err, "delete_failed")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]any{"status": "deleted"})
	default:
		writeMethodNotAllowed(w, "PATCH, DELETE")
	}
}
