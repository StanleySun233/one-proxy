package proxyservice

import (
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (s *Service) NodeLinks(tenantCtx domain.TenantAuthContext) []domain.NodeLink {
	return s.store.ListNodeLinksForTenant(tenantCtx)
}

func (s *Service) CreateNodeLink(tenantCtx domain.TenantAuthContext, input domain.CreateNodeLinkInput) (domain.NodeLink, error) {
	if err := requireActiveTenant(tenantCtx); err != nil {
		return domain.NodeLink{}, err
	}
	if !tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.Role != domain.TenantRoleAdmin {
		return domain.NodeLink{}, newError(http.StatusForbidden, "tenant_role_forbidden")
	}
	if input.SourceNodeID == "" || input.TargetNodeID == "" || input.LinkType == "" || input.TrustState == "" {
		return domain.NodeLink{}, invalidInput("invalid_node_link_payload")
	}
	if !s.tenantNodesExist(tenantCtx, []string{input.SourceNodeID, input.TargetNodeID}) {
		return domain.NodeLink{}, invalidInput("node_not_found")
	}
	return s.store.CreateNodeLinkForTenant(tenantCtx, input)
}

func (s *Service) UpdateNodeLink(tenantCtx domain.TenantAuthContext, linkID string, input domain.UpdateNodeLinkInput) (domain.NodeLink, error) {
	if linkID == "" {
		return domain.NodeLink{}, invalidInput("missing_node_link_id")
	}
	if err := s.requireTenantResourceManage(tenantCtx, func() (domain.BindingPermission, bool) {
		return s.store.NodeLinkBindingPermission(tenantCtx, linkID)
	}); err != nil {
		return domain.NodeLink{}, err
	}
	if input.SourceNodeID == "" || input.TargetNodeID == "" || input.LinkType == "" || input.TrustState == "" {
		return domain.NodeLink{}, invalidInput("invalid_node_link_payload")
	}
	if !s.tenantNodesExist(tenantCtx, []string{input.SourceNodeID, input.TargetNodeID}) {
		return domain.NodeLink{}, invalidInput("node_not_found")
	}
	return s.store.UpdateNodeLink(linkID, input)
}

func (s *Service) DeleteNodeLink(tenantCtx domain.TenantAuthContext, linkID string) error {
	if linkID == "" {
		return invalidInput("missing_node_link_id")
	}
	if err := s.requireTenantResourceManage(tenantCtx, func() (domain.BindingPermission, bool) {
		return s.store.NodeLinkBindingPermission(tenantCtx, linkID)
	}); err != nil {
		return err
	}
	if !tenantCtx.SuperAdmin && s.store.CountNodeLinkBindings(linkID) > 1 {
		return newError(http.StatusConflict, "shared_resource_delete_forbidden")
	}
	return s.store.DeleteNodeLink(linkID)
}
