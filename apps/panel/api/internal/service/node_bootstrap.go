package service

import "github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"

func (c *ControlPlane) CreateBootstrapToken(tenantCtx domain.TenantAuthContext, input domain.CreateBootstrapTokenInput) (domain.BootstrapToken, error) {
	if tenantCtx.ActiveTenant.TenantID == "" {
		return domain.BootstrapToken{}, invalidInput("tenant_required")
	}
	if !tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.Role != domain.TenantRoleAdmin {
		return domain.BootstrapToken{}, newError(403, "tenant_role_forbidden")
	}
	if input.TargetType == "" {
		return domain.BootstrapToken{}, invalidInput("invalid_bootstrap_payload")
	}
	if input.TargetID == "" {
		if err := validateNodeInput(input.NodeName, input.NodeMode, input.ScopeKey); err != nil {
			return domain.BootstrapToken{}, err
		}
		if !c.tenantScopeExists(tenantCtx, input.ScopeKey) {
			return domain.BootstrapToken{}, invalidInput("scope_not_found")
		}
		if input.ParentNodeID != "" && !c.tenantNodeExists(tenantCtx, input.ParentNodeID) {
			return domain.BootstrapToken{}, invalidInput("node_not_found")
		}
	} else if !c.tenantNodeExists(tenantCtx, input.TargetID) {
		return domain.BootstrapToken{}, invalidInput("node_not_found")
	}
	if input.NodeMode != "" && !c.isValidEnum("node_mode", input.NodeMode) {
		return domain.BootstrapToken{}, invalidInput("invalid_node_payload")
	}
	return c.store.CreateBootstrapTokenForTenant(tenantCtx, input)
}

func (c *ControlPlane) UnconsumedBootstrapTokens(tenantCtx domain.TenantAuthContext) ([]domain.BootstrapToken, error) {
	if !tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.Role != domain.TenantRoleAdmin {
		return nil, newError(403, "tenant_role_forbidden")
	}
	items := c.store.ListUnconsumedBootstrapTokens()
	if tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.TenantID == "" {
		return items, nil
	}
	filtered := make([]domain.BootstrapToken, 0)
	for _, item := range items {
		if _, ok := c.store.NodeBindingPermission(tenantCtx, item.TargetID); ok {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func (c *ControlPlane) DeleteBootstrapToken(tenantCtx domain.TenantAuthContext, tokenID string) error {
	if tokenID == "" {
		return invalidInput("missing_bootstrap_token_id")
	}
	if !tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.Role != domain.TenantRoleAdmin {
		return newError(403, "tenant_role_forbidden")
	}
	if !tenantCtx.SuperAdmin {
		found := false
		for _, item := range c.store.ListUnconsumedBootstrapTokens() {
			if item.ID != tokenID {
				continue
			}
			_, found = c.store.NodeBindingPermission(tenantCtx, item.TargetID)
			break
		}
		if !found {
			return newError(403, "resource_binding_forbidden")
		}
	}
	return c.store.DeleteBootstrapToken(tokenID)
}
