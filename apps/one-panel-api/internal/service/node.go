package service

import (
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (c *ControlPlane) Nodes(tenantCtx domain.TenantAuthContext) []domain.Node {
	return c.store.ListNodesForTenant(tenantCtx)
}

func (c *ControlPlane) UpdateNode(tenantCtx domain.TenantAuthContext, nodeID string, input domain.UpdateNodeInput) (domain.Node, error) {
	if nodeID == "" {
		return domain.Node{}, invalidInput("missing_node_id")
	}
	if err := c.requireTenantResourceManage(tenantCtx, func() (domain.BindingPermission, bool) {
		return c.store.NodeBindingPermission(tenantCtx, nodeID)
	}); err != nil {
		return domain.Node{}, err
	}
	if err := validateNodeInput(input.Name, input.Mode, input.ScopeKey); err != nil {
		return domain.Node{}, err
	}
	if !c.ScopeExists(input.ScopeKey) {
		return domain.Node{}, invalidInput("scope_not_found")
	}
	return c.store.UpdateNode(nodeID, input)
}

func (c *ControlPlane) DeleteNode(tenantCtx domain.TenantAuthContext, nodeID string) error {
	if err := c.requireTenantResourceManage(tenantCtx, func() (domain.BindingPermission, bool) {
		return c.store.NodeBindingPermission(tenantCtx, nodeID)
	}); err != nil {
		return err
	}
	if !tenantCtx.SuperAdmin && c.store.CountNodeBindings(nodeID) > 1 {
		return newError(http.StatusConflict, "shared_resource_delete_forbidden")
	}
	return c.store.DeleteNode(nodeID)
}

func (c *ControlPlane) requireTenantResourceManage(tenantCtx domain.TenantAuthContext, permission func() (domain.BindingPermission, bool)) error {
	if tenantCtx.SuperAdmin {
		return nil
	}
	if tenantCtx.ActiveTenant.Role != domain.TenantRoleAdmin {
		return newError(http.StatusForbidden, "tenant_role_forbidden")
	}
	bindingPermission, ok := permission()
	if !ok || bindingPermission != domain.BindingPermissionManage {
		return newError(http.StatusForbidden, "resource_binding_forbidden")
	}
	return nil
}

func validateNodeInput(name string, mode string, scopeKey string) error {
	if name == "" || mode == "" || scopeKey == "" {
		return invalidInput("invalid_node_payload")
	}
	return nil
}
