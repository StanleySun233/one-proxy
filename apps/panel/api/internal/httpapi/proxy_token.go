package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	proxy "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/domain"
)

type proxyTokenValidateRequest struct {
	TokenHash    string `json:"tokenHash"`
	Token        string `json:"token"`
	AccessPathID string `json:"accessPathId"`
	TargetHost   string `json:"targetHost"`
	TargetPort   int    `json:"targetPort"`
	Protocol     string `json:"protocol"`
	RouteID      string `json:"routeId"`
}

type proxyTokenValidateResponse struct {
	Valid           bool     `json:"valid"`
	TenantID        string   `json:"tenantId"`
	AccountID       string   `json:"accountId"`
	ExpiresAt       string   `json:"expiresAt"`
	CacheTTLSeconds int      `json:"cacheTtlSeconds"`
	AllowLocalProxy bool     `json:"allowLocalProxy"`
	Scopes          []string `json:"scopes"`
	AccessPathIDs   []string `json:"accessPathIds"`
	RouteIDs        []string `json:"routeIds"`
}

func (r *Router) handleProxyTokenAuthenticate(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	var payload proxyTokenValidateRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	payload.TokenHash = strings.TrimSpace(payload.TokenHash)
	payload.Token = strings.TrimSpace(payload.Token)
	if payload.Token != "" || !validProxyTokenHash(payload.TokenHash) {
		writeError(w, http.StatusBadRequest, "invalid_proxy_token_payload")
		return
	}
	nodeID, ok := nodeIDFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_node_token")
		return
	}
	result := r.service.ValidateProxyTokenHash(payload.TokenHash, nodeID)
	if !result.Valid || !result.AllowLocalProxy {
		writeError(w, http.StatusUnauthorized, "invalid_proxy_token")
		return
	}
	_, tenantID, ok := proxyTokenTenantContext(result)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_proxy_token")
		return
	}
	writeSuccess(w, http.StatusOK, proxyTokenValidateResponse{
		Valid:           true,
		TenantID:        tenantID,
		AccountID:       result.Account.ID,
		ExpiresAt:       result.ExpiresAt,
		CacheTTLSeconds: result.CacheTTLSeconds,
		AllowLocalProxy: true,
		Scopes:          []string{},
		AccessPathIDs:   []string{},
		RouteIDs:        []string{},
	})
}

func (r *Router) handleProxyTokenValidate(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	var payload proxyTokenValidateRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	payload.TokenHash = strings.TrimSpace(payload.TokenHash)
	payload.Token = strings.TrimSpace(payload.Token)
	payload.AccessPathID = strings.TrimSpace(payload.AccessPathID)
	payload.TargetHost = strings.TrimSpace(payload.TargetHost)
	payload.Protocol = strings.TrimSpace(payload.Protocol)
	payload.RouteID = strings.TrimSpace(payload.RouteID)
	if payload.Token != "" || !validProxyTokenHash(payload.TokenHash) || payload.TargetHost == "" || payload.TargetPort < 1 || payload.TargetPort > 65535 || payload.Protocol == "" || (payload.AccessPathID == "" && payload.RouteID == "") {
		writeError(w, http.StatusBadRequest, "invalid_proxy_token_payload")
		return
	}
	nodeID, ok := nodeIDFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_node_token")
		return
	}
	result := r.service.ValidateProxyTokenHash(payload.TokenHash, nodeID)
	if !result.Valid || !result.AllowLocalProxy {
		writeError(w, http.StatusUnauthorized, "invalid_proxy_token")
		return
	}
	tenantCtx, tenantID, ok := proxyTokenTenantContext(result)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_proxy_token")
		return
	}
	scopes := []string{}
	accessPathIDs := []string{}
	routeIDs := []string{}
	if payload.AccessPathID != "" {
		accessPath, ok := proxyTokenAccessPath(r.service.Proxy().AccessPaths(tenantCtx), payload.AccessPathID, nodeID)
		if !ok {
			writeError(w, http.StatusUnauthorized, "invalid_proxy_token")
			return
		}
		accessPathIDs = append(accessPathIDs, accessPath.ID)
		scopes, ok = proxyTokenScopes(r.service.Proxy().Chains(tenantCtx), accessPath.ChainID)
		if !ok {
			writeError(w, http.StatusUnauthorized, "invalid_proxy_token")
			return
		}
		if payload.RouteID != "" {
			if !proxyTokenRouteInChain(r.service.Proxy().RouteRules(tenantCtx), payload.RouteID, accessPath.ChainID) {
				writeError(w, http.StatusUnauthorized, "invalid_proxy_token")
				return
			}
			routeIDs = append(routeIDs, payload.RouteID)
		}
	} else {
		route, ok := proxyTokenDirectRoute(r.service.Proxy().RouteRules(tenantCtx), payload.RouteID)
		if !ok {
			writeError(w, http.StatusUnauthorized, "invalid_proxy_token")
			return
		}
		if route.DestinationScope != "" {
			scopes = append(scopes, route.DestinationScope)
		}
		routeIDs = append(routeIDs, route.ID)
	}
	writeSuccess(w, http.StatusOK, proxyTokenValidateResponse{
		Valid:           true,
		TenantID:        tenantID,
		AccountID:       result.Account.ID,
		ExpiresAt:       result.ExpiresAt,
		CacheTTLSeconds: result.CacheTTLSeconds,
		AllowLocalProxy: true,
		Scopes:          scopes,
		AccessPathIDs:   accessPathIDs,
		RouteIDs:        routeIDs,
	})
}

func validProxyTokenHash(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func proxyTokenTenantContext(result domain.ProxyTokenValidation) (domain.TenantAuthContext, string, bool) {
	if result.ActiveTenantID == nil || *result.ActiveTenantID == "" {
		return domain.TenantAuthContext{}, "", false
	}
	for _, membership := range result.TenantMemberships {
		if membership.TenantID == *result.ActiveTenantID {
			return domain.TenantAuthContext{
				Account:      result.Account,
				ActiveTenant: membership,
				SuperAdmin:   result.Account.Role == domain.AccountRoleSuperAdmin,
			}, membership.TenantID, true
		}
	}
	return domain.TenantAuthContext{}, "", false
}

func proxyTokenAccessPath(paths []domain.NodeAccessPath, accessPathID string, nodeID string) (domain.NodeAccessPath, bool) {
	for _, path := range paths {
		if path.ID == accessPathID && path.Enabled && proxyTokenAccessPathIncludesNode(path, nodeID) {
			return path, true
		}
	}
	return domain.NodeAccessPath{}, false
}

func proxyTokenAccessPathIncludesNode(path domain.NodeAccessPath, nodeID string) bool {
	if path.EntryNodeID == nodeID || path.TargetNodeID == nodeID {
		return true
	}
	for _, relayNodeID := range path.RelayNodeIDs {
		if relayNodeID == nodeID {
			return true
		}
	}
	return false
}

func proxyTokenScopes(chains []proxy.Chain, chainID string) ([]string, bool) {
	for _, chain := range chains {
		if chain.ID == chainID && chain.Enabled {
			scopes := []string{}
			if chain.DestinationScope != "" {
				scopes = append(scopes, chain.DestinationScope)
			}
			return scopes, true
		}
	}
	return nil, false
}

func proxyTokenRouteInChain(routes []proxy.RouteRule, routeID string, chainID string) bool {
	for _, route := range routes {
		if route.ID == routeID && route.Enabled && route.ActionType == domain.ActionTypeChain && route.ChainID == chainID {
			return true
		}
	}
	return false
}

func proxyTokenDirectRoute(routes []proxy.RouteRule, routeID string) (proxy.RouteRule, bool) {
	for _, route := range routes {
		if route.ID == routeID && route.Enabled && route.ActionType == domain.ActionTypeDirect {
			return route, true
		}
	}
	return proxy.RouteRule{}, false
}
