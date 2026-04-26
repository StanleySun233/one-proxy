package bootstrap

import (
	"encoding/json"
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/network"
	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/runtime"
)

type Handler struct {
	joinPassword    string
	listenAddr      string
	httpsListenAddr string
	manager         *runtime.Manager
}

type responseEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data,omitempty"`
}

func New(joinPassword string, listenAddr string, httpsListenAddr string, manager *runtime.Manager) *Handler {
	return &Handler{
		joinPassword:    joinPassword,
		listenAddr:      listenAddr,
		httpsListenAddr: httpsListenAddr,
		manager:         manager,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if h.joinPassword == "" {
		writeError(w, http.StatusForbidden, "node_join_password_not_configured")
		return
	}
	var payload domain.NodeBootstrapAttachInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if payload.Password == "" || payload.Password != h.joinPassword {
		writeError(w, http.StatusUnauthorized, "invalid_join_password")
		return
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
	writeSuccess(w, http.StatusOK, domain.NodeBootstrapAttachResult{
		ConnectionStatus:    "connected",
		LocalIPs:            network.LocalIPs(),
		NodeListenAddr:      h.listenAddr,
		NodeHTTPSListenAddr: h.httpsListenAddr,
		ControlPlaneBound:   h.manager.Bound(),
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
