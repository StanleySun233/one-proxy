package localconsole

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/agentconfig"
	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/network"
	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/policystore"
	noderuntime "github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/runtime"
)

const cookieName = "one_proxy_node_console"

type Handler struct {
	manager  *noderuntime.Manager
	store    *policystore.Store
	cfg      agentconfig.Config
	started  time.Time
	sessions map[string]session
	mu       sync.RWMutex
}

type session struct {
	User      user          `json:"user"`
	CreatedAt time.Time     `json:"createdAt"`
	Auth      authorization `json:"authorization"`
}

type user struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type authorization struct {
	Mode                string `json:"mode"`
	ManageAccessChecked bool   `json:"manageAccessChecked"`
}

type envelope struct {
	OK    bool      `json:"ok"`
	Data  any       `json:"data,omitempty"`
	Error *apiError `json:"error,omitempty"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func New(manager *noderuntime.Manager, store *policystore.Store, cfg agentconfig.Config, started time.Time) *Handler {
	return &Handler{
		manager:  manager,
		store:    store,
		cfg:      cfg,
		started:  started,
		sessions: map[string]session{},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/api/local/login":
		h.login(w, req)
	case "/api/local/logout":
		h.logout(w, req)
	case "/api/local/session":
		h.currentSession(w, req)
	case "/api/local/status":
		h.requireSession(w, req, h.status)
	case "/api/local/health":
		h.requireSession(w, req, h.health)
	case "/api/local/audit":
		h.requireSession(w, req, h.audit)
	case "/api/local/policy":
		h.requireSession(w, req, h.policy)
	case "/api/local/diagnostics":
		h.requireSession(w, req, h.diagnostics)
	default:
		writeError(w, http.StatusNotFound, "not_found", "Not found")
	}
}

func (h *Handler) login(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}
	controlPlaneURL := h.manager.Current().ControlPlaneURL
	if controlPlaneURL == "" {
		controlPlaneURL = h.cfg.ControlPlaneURL
	}
	if controlPlaneURL == "" {
		writeError(w, http.StatusServiceUnavailable, "control_plane_unbound", "Control plane is not configured")
		return
	}
	panelUser, auth, err := authenticate(controlPlaneURL, h.manager.Current().NodeID, payload.Username, payload.Password)
	if err != nil {
		if err.Error() == "node_manage_forbidden" {
			writeError(w, http.StatusForbidden, "node_manage_forbidden", "Node manage access is required")
			return
		}
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}
	token, err := sessionToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session_failed", "Session failed")
		return
	}
	item := session{
		User:      panelUser,
		CreatedAt: time.Now().UTC(),
		Auth:      auth,
	}
	h.mu.Lock()
	h.sessions[token] = item
	h.mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	writeOK(w, map[string]any{"user": item.User, "authorization": item.Auth})
}

func (h *Handler) logout(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	if cookie, err := req.Cookie(cookieName); err == nil {
		h.mu.Lock()
		delete(h.sessions, cookie.Value)
		h.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	writeOK(w, map[string]bool{"loggedOut": true})
}

func (h *Handler) currentSession(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	item, ok := h.session(req)
	if !ok {
		writeOK(w, map[string]bool{"authenticated": false})
		return
	}
	writeOK(w, map[string]any{
		"authenticated": true,
		"user":          item.User,
		"authorization": item.Auth,
	})
}

func (h *Handler) status(w http.ResponseWriter, req *http.Request, _ session) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	current := h.manager.Current()
	revision, _ := h.store.Current()
	writeOK(w, map[string]any{
		"node": map[string]any{
			"id":            current.NodeID,
			"name":          current.NodeName,
			"role":          current.NodeMode,
			"version":       "1.0.0",
			"startedAt":     h.started.UTC().Format(time.RFC3339),
			"uptimeSeconds": int64(time.Since(h.started).Seconds()),
		},
		"controlPlane": map[string]any{
			"url":        current.ControlPlaneURL,
			"bound":      h.manager.Bound(),
			"lastSyncAt": "",
		},
		"listeners": map[string]string{
			"http":      h.cfg.ListenAddr,
			"https":     h.cfg.HTTPSListenAddr,
			"tcpAccess": h.cfg.TCPAccessListenAddr,
			"udpAccess": h.cfg.UDPAccessListenAddr,
		},
		"runtime": map[string]any{
			"activeProxySessions": 0,
			"activeTunnels":       0,
			"policyRevision":      revision,
		},
	})
}

func (h *Handler) health(w http.ResponseWriter, req *http.Request, _ session) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	controlStatus := "healthy"
	controlMessage := "connected"
	if !h.manager.Bound() {
		controlStatus = "degraded"
		controlMessage = "not bound"
	}
	revision, _ := h.store.Current()
	policyStatus := "healthy"
	policyMessage := "revision " + revision + " loaded"
	if revision == "" {
		policyStatus = "degraded"
		policyMessage = "no policy revision loaded"
	}
	status := "healthy"
	if controlStatus != "healthy" || policyStatus != "healthy" {
		status = "degraded"
	}
	writeOK(w, map[string]any{
		"status":    status,
		"checkedAt": now,
		"checks": []map[string]string{
			{"name": "control_plane", "status": controlStatus, "message": controlMessage, "lastCheckedAt": now},
			{"name": "policy_store", "status": policyStatus, "message": policyMessage, "lastCheckedAt": now},
		},
	})
}

func (h *Handler) audit(w http.ResponseWriter, req *http.Request, _ session) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	limit := 50
	if value := req.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 200 {
			writeError(w, http.StatusBadRequest, "invalid_limit", "Invalid limit")
			return
		}
		limit = parsed
	}
	_ = limit
	writeOK(w, map[string]any{"items": []any{}, "nextCursor": ""})
}

func (h *Handler) policy(w http.ResponseWriter, req *http.Request, _ session) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	revision, snapshot := h.store.Current()
	nodes := make([]map[string]string, 0, len(snapshot.Nodes))
	for _, item := range snapshot.Nodes {
		nodes = append(nodes, map[string]string{"id": item.ID, "name": item.Name, "role": item.Mode})
	}
	routes := make([]map[string]any, 0, len(snapshot.RouteRules))
	for _, item := range snapshot.RouteRules {
		routes = append(routes, map[string]any{
			"id":      item.ID,
			"name":    item.ID,
			"action":  item.ActionType,
			"enabled": item.Enabled,
			"match": map[string]string{
				"host":     item.MatchValue,
				"protocol": item.MatchType,
			},
		})
	}
	writeOK(w, map[string]any{
		"revision": revision,
		"loadedAt": "",
		"source":   "control_plane",
		"nodes":    nodes,
		"routes":   routes,
	})
}

func (h *Handler) diagnostics(w http.ResponseWriter, req *http.Request, _ session) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	writeOK(w, map[string]any{
		"generatedAt": time.Now().UTC().Format(time.RFC3339),
		"environment": map[string]string{
			"version":   "1.0.0",
			"goVersion": runtime.Version(),
			"os":        runtime.GOOS,
			"arch":      runtime.GOARCH,
		},
		"network": map[string]any{
			"localAddresses": network.LocalIPs(),
			"natType":        "unknown",
		},
		"controlPlane": map[string]any{
			"configured": h.manager.Current().ControlPlaneURL != "",
			"reachable":  h.manager.Bound(),
			"lastError":  "",
		},
		"recentErrors": []any{},
	})
}

func (h *Handler) requireSession(w http.ResponseWriter, req *http.Request, next func(http.ResponseWriter, *http.Request, session)) {
	item, ok := h.session(req)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}
	next(w, req, item)
}

func (h *Handler) session(req *http.Request) (session, bool) {
	cookie, err := req.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return session{}, false
	}
	h.mu.RLock()
	item, ok := h.sessions[cookie.Value]
	h.mu.RUnlock()
	return item, ok
}

func (h *Handler) Authenticated(req *http.Request) bool {
	_, ok := h.session(req)
	return ok
}

func authenticate(controlPlaneURL string, nodeID string, username string, password string) (user, authorization, error) {
	if controlPlaneURL == "" || nodeID == "" || username == "" || password == "" {
		return user{}, authorization{}, errors.New("missing_credentials")
	}
	body, err := json.Marshal(map[string]string{"account": username, "password": password})
	if err != nil {
		return user{}, authorization{}, err
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(controlPlaneURL, "/")+"/api/auth/login", bytes.NewReader(body))
	if err != nil {
		return user{}, authorization{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return user{}, authorization{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return user{}, authorization{}, errors.New("control_plane_login_failed")
	}
	var result struct {
		Data struct {
			AccessToken string `json:"accessToken"`
			Account     struct {
				ID      string `json:"id"`
				Account string `json:"account"`
				Role    string `json:"role"`
			} `json:"account"`
			TenantMemberships []struct {
				TenantID string `json:"tenantId"`
				Role     string `json:"role"`
			} `json:"tenantMemberships"`
			ActiveTenantID *string `json:"activeTenantId"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return user{}, authorization{}, err
	}
	auth, err := validateManageAccess(client, controlPlaneURL, nodeID, result.Data.AccessToken, result.Data.Account.Role, result.Data.ActiveTenantID, result.Data.TenantMemberships)
	if err != nil {
		return user{}, authorization{}, err
	}
	email := ""
	if strings.Contains(result.Data.Account.Account, "@") {
		email = result.Data.Account.Account
	}
	name := result.Data.Account.Account
	if name == "" {
		name = username
	}
	return user{ID: result.Data.Account.ID, Name: name, Email: email}, auth, nil
}

func validateManageAccess(client *http.Client, controlPlaneURL string, nodeID string, accessToken string, accountRole string, activeTenantID *string, memberships []struct {
	TenantID string `json:"tenantId"`
	Role     string `json:"role"`
}) (authorization, error) {
	if accessToken == "" {
		return authorization{}, errors.New("node_manage_forbidden")
	}
	if accountRole == "super_admin" && manageAccessAllowed(client, controlPlaneURL, nodeID, accessToken, "") {
		return authorization{Mode: "panel_manage_access", ManageAccessChecked: true}, nil
	}
	if activeTenantID != nil && *activeTenantID != "" && manageAccessAllowed(client, controlPlaneURL, nodeID, accessToken, *activeTenantID) {
		return authorization{Mode: "panel_manage_access", ManageAccessChecked: true}, nil
	}
	for _, membership := range memberships {
		if membership.Role == "tenant_admin" && manageAccessAllowed(client, controlPlaneURL, nodeID, accessToken, membership.TenantID) {
			return authorization{Mode: "panel_manage_access", ManageAccessChecked: true}, nil
		}
	}
	return authorization{}, errors.New("node_manage_forbidden")
}

func manageAccessAllowed(client *http.Client, controlPlaneURL string, nodeID string, accessToken string, tenantID string) bool {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(controlPlaneURL, "/")+"/api/nodes/"+nodeID+"/access/manage", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if tenantID != "" {
		req.Header.Set("X-Tenant-ID", tenantID)
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func sessionToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func writeOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, envelope{OK: true, Data: data})
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, envelope{OK: false, Error: &apiError{Code: code, Message: message}})
}

func writeMethodNotAllowed(w http.ResponseWriter, method string) {
	w.Header().Set("Allow", method)
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
}

func writeJSON(w http.ResponseWriter, status int, payload envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
