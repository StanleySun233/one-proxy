package linkservice

import (
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (s *Service) AccessPaths(tenantCtx domain.TenantAuthContext) []domain.NodeAccessPath {
	return s.store.ListNodeAccessPathsForTenant(tenantCtx)
}

func (s *Service) CreateAccessPath(tenantCtx domain.TenantAuthContext, input domain.CreateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	if err := requireActiveTenant(tenantCtx); err != nil {
		return domain.NodeAccessPath{}, err
	}
	if !tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.Role != domain.TenantRoleAdmin {
		return domain.NodeAccessPath{}, newError(http.StatusForbidden, "tenant_role_forbidden")
	}
	input = normalizeCreateAccessPathInput(input)
	if err := s.validateAccessPath(tenantCtx, input.ChainID, input.Name, input.Mode, input.Protocol, input.ServiceType, input.TargetHost, input.TargetPort, input.ListenPort, input.TLSMode, input.AuthMode); err != nil {
		return domain.NodeAccessPath{}, err
	}
	return s.store.CreateNodeAccessPathForTenant(tenantCtx, input)
}

func (s *Service) UpdateAccessPath(tenantCtx domain.TenantAuthContext, pathID string, input domain.UpdateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	if pathID == "" {
		return domain.NodeAccessPath{}, invalidInput("missing_path_id")
	}
	if err := s.requireTenantResourceManage(tenantCtx, func() (domain.BindingPermission, bool) {
		return s.store.NodeAccessPathBindingPermission(tenantCtx, pathID)
	}); err != nil {
		return domain.NodeAccessPath{}, err
	}
	input = normalizeUpdateAccessPathInput(input)
	if err := s.validateAccessPath(tenantCtx, input.ChainID, input.Name, input.Mode, input.Protocol, input.ServiceType, input.TargetHost, input.TargetPort, input.ListenPort, input.TLSMode, input.AuthMode); err != nil {
		return domain.NodeAccessPath{}, err
	}
	return s.store.UpdateNodeAccessPath(pathID, input)
}

func (s *Service) DeleteAccessPath(tenantCtx domain.TenantAuthContext, pathID string) error {
	if pathID == "" {
		return invalidInput("missing_path_id")
	}
	if err := s.requireTenantResourceManage(tenantCtx, func() (domain.BindingPermission, bool) {
		return s.store.NodeAccessPathBindingPermission(tenantCtx, pathID)
	}); err != nil {
		return err
	}
	if !(tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.TenantID == "") && s.store.CountNodeAccessPathBindings(pathID) > 1 {
		return newError(http.StatusConflict, "shared_resource_delete_forbidden")
	}
	return s.store.DeleteNodeAccessPath(pathID)
}

func normalizeCreateAccessPathInput(input domain.CreateNodeAccessPathInput) domain.CreateNodeAccessPathInput {
	if input.Protocol == "" {
		input.Protocol = domain.AccessProtocolHTTP
	}
	if input.ServiceType == "" {
		input.ServiceType = domain.AccessServiceHTTP
	}
	if input.TargetProtocol == "" {
		input.TargetProtocol = input.Protocol
	}
	if input.TLSMode == "" {
		input.TLSMode = domain.TLSModeNone
	}
	if input.AuthMode == "" {
		input.AuthMode = domain.AccessAuthProxyToken
	}
	return input
}

func normalizeUpdateAccessPathInput(input domain.UpdateNodeAccessPathInput) domain.UpdateNodeAccessPathInput {
	if input.Protocol == "" {
		input.Protocol = domain.AccessProtocolHTTP
	}
	if input.ServiceType == "" {
		input.ServiceType = domain.AccessServiceHTTP
	}
	if input.TargetProtocol == "" {
		input.TargetProtocol = input.Protocol
	}
	if input.TLSMode == "" {
		input.TLSMode = domain.TLSModeNone
	}
	if input.AuthMode == "" {
		input.AuthMode = domain.AccessAuthProxyToken
	}
	return input
}

func (s *Service) validateAccessPath(tenantCtx domain.TenantAuthContext, chainID string, name string, mode string, protocol string, serviceType string, targetHost string, targetPort int, listenPort int, tlsMode string, authMode string) error {
	if chainID == "" || name == "" || mode == "" {
		return invalidInput("invalid_access_path_payload")
	}
	if _, ok := chainByID(s.store.ListChainsForTenant(tenantCtx), chainID); !ok {
		return invalidInput("chain_not_found")
	}
	if !s.isValidEnum("path_mode", mode) {
		return invalidInput("invalid_access_path_payload")
	}
	if !s.isValidEnum("access_protocol", protocol) || !s.isValidEnum("access_service_type", serviceType) {
		return invalidInput("invalid_access_path_payload")
	}
	if tlsMode != "" && !s.isValidEnum("tls_mode", tlsMode) {
		return invalidInput("invalid_access_path_payload")
	}
	if authMode != "" && !s.isValidEnum("access_auth_mode", authMode) {
		return invalidInput("invalid_access_path_payload")
	}
	switch mode {
	case domain.PathModeDirect, domain.PathModeRelayChain:
		if targetHost == "" || targetPort <= 0 {
			return invalidInput("invalid_access_path_payload")
		}
	}
	switch protocol {
	case domain.AccessProtocolTCP, domain.AccessProtocolTLS, domain.AccessProtocolSSH, domain.AccessProtocolRDP, domain.AccessProtocolSocks5, domain.AccessProtocolSS5, domain.AccessProtocolUDP:
		if listenPort < 0 || targetPort <= 0 {
			return invalidInput("invalid_access_path_payload")
		}
	}
	return nil
}
