package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (r *Router) handleChains(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		includeDetails := req.URL.Query().Get("details") == "true"
		if includeDetails {
			writeSuccess(w, http.StatusOK, r.service.ChainsWithDetails())
		} else {
			writeSuccess(w, http.StatusOK, r.service.Chains())
		}
	case http.MethodPost:
		var payload domain.CreateChainInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.CreateChain(payload)
		if err != nil {
			writeServiceError(w, req, err, "create_failed")
			return
		}
		writeSuccess(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (r *Router) handleChainByID(w http.ResponseWriter, req *http.Request) {
	if strings.HasSuffix(req.URL.Path, "/probe") {
		r.handleChainProbe(w, req)
		return
	}
	chainID := resourceID(req.URL.Path, "/api/v1/chains/")
	if chainID == "" {
		writeError(w, http.StatusBadRequest, "missing_chain_id")
		return
	}
	switch req.Method {
	case http.MethodGet:
		item, err := r.service.GetChain(chainID)
		if err != nil {
			writeServiceError(w, req, err, "get_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodPatch:
		var payload domain.UpdateChainInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.UpdateChain(chainID, payload)
		if err != nil {
			writeServiceError(w, req, err, "update_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := r.service.DeleteChain(chainID); err != nil {
			writeServiceError(w, req, err, "delete_failed")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]any{"status": "deleted"})
	default:
		writeMethodNotAllowed(w, "GET, PATCH, DELETE")
	}
}

func (r *Router) handleChainProbe(w http.ResponseWriter, req *http.Request) {
	chainID := strings.TrimSuffix(resourceID(req.URL.Path, "/api/v1/chains/"), "/probe")
	chainID = strings.TrimSuffix(chainID, "/")
	if chainID == "" {
		writeError(w, http.StatusBadRequest, "missing_chain_id")
		return
	}
	switch req.Method {
	case http.MethodGet:
		item, ok := r.service.LatestChainProbe(chainID)
		if !ok {
			writeError(w, http.StatusNotFound, "chain_probe_not_found")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodPost:
		item, err := r.service.ProbeChain(chainID)
		if err != nil {
			writeServiceError(w, req, err, "probe_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (r *Router) handleChainValidate(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	var payload domain.ValidateChainInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	result, err := r.service.ValidateChain(payload)
	if err != nil {
		writeServiceError(w, req, err, "validation_failed")
		return
	}
	writeSuccess(w, http.StatusOK, result)
}

func (r *Router) handleChainPreview(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	var payload domain.PreviewChainInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	result, err := r.service.PreviewChain(payload)
	if err != nil {
		writeServiceError(w, req, err, "preview_failed")
		return
	}
	writeSuccess(w, http.StatusOK, result)
}
