package bootstrap

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/network"
	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/runtime"
)

type Handler struct {
	listenAddr      string
	httpsListenAddr string
	manager         *runtime.Manager
}

type responseEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data,omitempty"`
}

func New(listenAddr string, httpsListenAddr string, manager *runtime.Manager) *Handler {
	return &Handler{
		listenAddr:      listenAddr,
		httpsListenAddr: httpsListenAddr,
		manager:         manager,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		h.handleAttach(w, req)
	case http.MethodPatch:
		h.handleRotatePassword(w, req)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
	}
}

func (h *Handler) handleAttach(w http.ResponseWriter, req *http.Request) {
	joinPassword := h.manager.JoinPassword()
	if joinPassword == "" {
		writeError(w, http.StatusForbidden, "node_join_password_not_configured")
		return
	}
	var payload domain.NodeBootstrapAttachInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if payload.Password == "" || payload.Password != joinPassword {
		writeError(w, http.StatusUnauthorized, "invalid_join_password")
		return
	}
	if h.manager.MustRotatePassword() {
		if payload.NewPassword == "" {
			writeError(w, http.StatusBadRequest, "node_password_rotation_required")
			return
		}
		if payload.NewPassword == joinPassword {
			writeError(w, http.StatusBadRequest, "invalid_new_join_password")
			return
		}
	}
	if payload.ControlPlaneURL == "" || payload.NodeID == "" || payload.NodeAccessToken == "" {
		writeError(w, http.StatusBadRequest, "invalid_attach_payload")
		return
	}
	if err := h.manager.Attach(runtime.Binding{
		ControlPlaneURL: payload.ControlPlaneURL,
		NodeID:          payload.NodeID,
		NodeAccessToken: payload.NodeAccessToken,
		NodeName:        payload.NodeName,
		NodeMode:        payload.NodeMode,
		NodeScopeKey:    payload.NodeScopeKey,
		NodeParentID:    payload.NodeParentID,
		NodePublicHost:  payload.NodePublicHost,
		NodePublicPort:  payload.NodePublicPort,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if h.manager.MustRotatePassword() {
		if err := h.manager.RotateJoinPassword(joinPassword, payload.NewPassword); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeSuccess(w, http.StatusOK, domain.NodeBootstrapAttachResult{
		ConnectionStatus:    "connected",
		LocalIPs:            network.LocalIPs(),
		NodeListenAddr:      h.listenAddr,
		NodeHTTPSListenAddr: h.httpsListenAddr,
		ControlPlaneBound:   h.manager.Bound(),
		MustRotatePassword:  h.manager.MustRotatePassword(),
	})
}

func (h *Handler) handleRotatePassword(w http.ResponseWriter, req *http.Request) {
	var payload domain.NodeBootstrapPasswordRotateInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	err := h.manager.RotateJoinPassword(payload.CurrentPassword, payload.NewPassword)
	if err == os.ErrPermission {
		writeError(w, http.StatusUnauthorized, "invalid_join_password")
		return
	}
	if err == os.ErrInvalid {
		writeError(w, http.StatusBadRequest, "invalid_new_join_password")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{
		"status":             "updated",
		"mustRotatePassword": h.manager.MustRotatePassword(),
	})
}

func writeSuccess[T any](w http.ResponseWriter, status int, data T) {
	writeEnvelope(w, status, responseEnvelope[T]{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeEnvelope(w, status, responseEnvelope[any]{
		Code:    status,
		Message: message,
	})
}

func writeEnvelope[T any](w http.ResponseWriter, status int, payload responseEnvelope[T]) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
