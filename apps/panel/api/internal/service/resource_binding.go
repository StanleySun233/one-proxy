package service

import (
	"database/sql"
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func (c *ControlPlane) ResourceBindings(account domain.Account, tenantCtx domain.TenantAuthContext, resourceType domain.ResourceType, resourceID string) ([]domain.TenantResourceBinding, error) {
	if err := c.requireResourceGrantManage(account, tenantCtx, resourceType, resourceID); err != nil {
		return nil, err
	}
	return c.store.ListTenantResourceBindings(resourceType, resourceID)
}

func (c *ControlPlane) UpsertResourceBinding(account domain.Account, tenantCtx domain.TenantAuthContext, resourceType domain.ResourceType, resourceID string, tenantID string, input domain.UpsertTenantResourceBindingInput) (domain.TenantResourceBinding, error) {
	if input.Permission == "" {
		input.Permission = domain.BindingPermissionUse
	}
	if input.Permission != domain.BindingPermissionUse && input.Permission != domain.BindingPermissionManage {
		return domain.TenantResourceBinding{}, invalidInput("invalid_binding_permission")
	}
	if _, ok := c.store.GetTenant(tenantID); !ok {
		return domain.TenantResourceBinding{}, invalidInput("tenant_invalid")
	}
	if err := c.requireResourceGrantManage(account, tenantCtx, resourceType, resourceID); err != nil {
		return domain.TenantResourceBinding{}, err
	}
	if input.Permission != domain.BindingPermissionManage && c.wouldRemoveLastManageBinding(resourceType, resourceID, tenantID) {
		return domain.TenantResourceBinding{}, newError(http.StatusConflict, "resource_manage_binding_required")
	}
	return c.store.UpsertTenantResourceBinding(resourceType, resourceID, tenantID, input.Permission, account.ID)
}

func (c *ControlPlane) DeleteResourceBinding(account domain.Account, tenantCtx domain.TenantAuthContext, resourceType domain.ResourceType, resourceID string, tenantID string) error {
	if err := c.requireResourceGrantManage(account, tenantCtx, resourceType, resourceID); err != nil {
		return err
	}
	if c.wouldRemoveLastManageBinding(resourceType, resourceID, tenantID) {
		return newError(http.StatusConflict, "resource_manage_binding_required")
	}
	return resourceBindingNotFound(c.store.DeleteTenantResourceBinding(resourceType, resourceID, tenantID))
}

func (c *ControlPlane) requireResourceGrantManage(account domain.Account, tenantCtx domain.TenantAuthContext, resourceType domain.ResourceType, resourceID string) error {
	if resourceID == "" || !validResourceType(resourceType) {
		return invalidInput("invalid_resource_binding_payload")
	}
	if account.Role == domain.AccountRoleSuperAdmin {
		return nil
	}
	if tenantCtx.ActiveTenant.Role != domain.TenantRoleAdmin {
		return newError(http.StatusForbidden, "tenant_role_forbidden")
	}
	permission, ok := c.store.TenantResourceBindingPermission(tenantCtx, resourceType, resourceID)
	if !ok || permission != domain.BindingPermissionManage {
		return newError(http.StatusForbidden, "resource_binding_forbidden")
	}
	return nil
}

func (c *ControlPlane) wouldRemoveLastManageBinding(resourceType domain.ResourceType, resourceID string, tenantID string) bool {
	bindings, err := c.store.ListTenantResourceBindings(resourceType, resourceID)
	if err != nil {
		return false
	}
	manageCount := 0
	targetIsManage := false
	for _, binding := range bindings {
		if binding.Permission != domain.BindingPermissionManage {
			continue
		}
		manageCount++
		if binding.TenantID == tenantID {
			targetIsManage = true
		}
	}
	return targetIsManage && manageCount <= 1
}

func validResourceType(resourceType domain.ResourceType) bool {
	switch resourceType {
	case domain.ResourceTypeNode, domain.ResourceTypeNodeLink, domain.ResourceTypeScope, domain.ResourceTypeChain, domain.ResourceTypeRouteRule, domain.ResourceTypeAccessPath:
		return true
	default:
		return false
	}
}

func resourceBindingNotFound(err error) error {
	if err == sql.ErrNoRows {
		return invalidInput("resource_binding_not_found")
	}
	return err
}
