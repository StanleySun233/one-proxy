package proxyservice

import (
	maindomain "github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	proxy "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/domain"
	"net/http"
	"strings"
)

func (s *Service) Scopes(tenantCtx maindomain.TenantAuthContext) []proxy.Scope {
	return s.store.ListScopesForTenant(tenantCtx)
}

func (s *Service) CreateScope(tenantCtx maindomain.TenantAuthContext, input proxy.CreateScopeInput) (proxy.Scope, error) {
	if err := requireActiveTenant(tenantCtx); err != nil {
		return proxy.Scope{}, err
	}
	if !tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.Role != maindomain.TenantRoleAdmin {
		return proxy.Scope{}, newError(http.StatusForbidden, "tenant_role_forbidden")
	}
	input.ID = ""
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return proxy.Scope{}, invalidInput("invalid_scope_payload")
	}
	return s.store.CreateScopeForTenant(tenantCtx, input)
}

func (s *Service) UpdateScope(tenantCtx maindomain.TenantAuthContext, scopeID string, input proxy.UpdateScopeInput) (proxy.Scope, error) {
	if scopeID == "" || strings.TrimSpace(input.Name) == "" {
		return proxy.Scope{}, invalidInput("invalid_scope_payload")
	}
	if err := s.requireTenantResourceManage(tenantCtx, func() (maindomain.BindingPermission, bool) {
		return s.store.ScopeBindingPermission(tenantCtx, scopeID)
	}); err != nil {
		return proxy.Scope{}, err
	}
	return s.store.UpdateScope(scopeID, input)
}

func (s *Service) DeleteScope(tenantCtx maindomain.TenantAuthContext, scopeID string) error {
	if scopeID == "" {
		return invalidInput("missing_scope_id")
	}
	if err := s.requireTenantResourceManage(tenantCtx, func() (maindomain.BindingPermission, bool) {
		return s.store.ScopeBindingPermission(tenantCtx, scopeID)
	}); err != nil {
		return err
	}
	if !tenantCtx.SuperAdmin && s.store.CountScopeBindings(scopeID) > 1 {
		return newError(http.StatusConflict, "shared_resource_delete_forbidden")
	}
	if s.scopeInUse(scopeID) {
		return invalidInput("scope_in_use")
	}
	return s.store.DeleteScope(scopeID)
}

func (s *Service) tenantScopeExists(tenantCtx maindomain.TenantAuthContext, scopeID string) bool {
	for _, scope := range s.store.ListScopesForTenant(tenantCtx) {
		if scope.ID == scopeID {
			return true
		}
	}
	return false
}

func (s *Service) scopeExists(scopeID string) bool {
	return s.ScopeExists(scopeID)
}

func (s *Service) ScopeExists(scopeID string) bool {
	for _, scope := range s.store.ListScopes() {
		if scope.ID == scopeID {
			return true
		}
	}
	return false
}

func (s *Service) scopeInUse(scopeID string) bool {
	for _, node := range s.store.ListNodes() {
		if node.ScopeKey == scopeID {
			return true
		}
	}
	for _, chain := range s.store.ListChains() {
		if chain.DestinationScope == scopeID {
			return true
		}
	}
	for _, rule := range s.store.ListRouteRules() {
		if rule.DestinationScope == scopeID {
			return true
		}
	}
	groups, err := s.store.ListGroups()
	if err != nil {
		return true
	}
	for _, group := range groups {
		scopes, err := s.store.GetGroupScopes(group.ID)
		if err != nil {
			return true
		}
		for _, item := range scopes {
			if item == scopeID {
				return true
			}
		}
	}
	return false
}
