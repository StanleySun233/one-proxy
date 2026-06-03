package proxyservice

import (
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/store"
)

type Service struct {
	store store.Store
}

type Error struct {
	Status  int
	Code    string
	Message string
}

func New(store store.Store) *Service {
	return &Service{store: store}
}

func (e *Error) Error() string {
	return e.Message
}

func invalidInput(code string) *Error {
	return &Error{Status: http.StatusBadRequest, Code: code, Message: code}
}

func newError(status int, code string) *Error {
	return &Error{Status: status, Code: code, Message: code}
}

func (s *Service) requireTenantResourceManage(tenantCtx domain.TenantAuthContext, permission func() (domain.BindingPermission, bool)) error {
	if tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.TenantID == "" {
		return nil
	}
	if !tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.Role != domain.TenantRoleAdmin {
		return newError(http.StatusForbidden, "tenant_role_forbidden")
	}
	bindingPermission, ok := permission()
	if !ok || bindingPermission != domain.BindingPermissionManage {
		return newError(http.StatusForbidden, "resource_binding_forbidden")
	}
	return nil
}

func requireActiveTenant(tenantCtx domain.TenantAuthContext) error {
	if tenantCtx.ActiveTenant.TenantID == "" {
		return invalidInput("tenant_required")
	}
	return nil
}

func (s *Service) isValidEnum(field, value string) bool {
	items, err := s.store.ListFieldEnumsByField(field)
	if err != nil {
		return true
	}
	for _, item := range items {
		if item.Value == value {
			return true
		}
	}
	return false
}

func nodeByID(items []domain.Node, nodeID string) (domain.Node, bool) {
	for _, item := range items {
		if item.ID == nodeID {
			return item, true
		}
	}
	return domain.Node{}, false
}

func (s *Service) tenantNodesExist(tenantCtx domain.TenantAuthContext, nodeIDs []string) bool {
	nodes := s.store.ListNodesForTenant(tenantCtx)
	for _, nodeID := range nodeIDs {
		if _, ok := nodeByID(nodes, nodeID); !ok {
			return false
		}
	}
	return true
}
