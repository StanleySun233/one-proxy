package proxyservice

import (
	"net/http"
	"slices"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	proxy "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/domain"
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
	if err := s.validateCreateAccessPath(tenantCtx, input); err != nil {
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
	if err := s.validateUpdateAccessPath(tenantCtx, input); err != nil {
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
	if !tenantCtx.SuperAdmin && s.store.CountNodeAccessPathBindings(pathID) > 1 {
		return newError(http.StatusConflict, "shared_resource_delete_forbidden")
	}
	return s.store.DeleteNodeAccessPath(pathID)
}

func (s *Service) AccessPathDeleteImpact(tenantCtx domain.TenantAuthContext, pathID string) (proxy.NodeAccessPathDeleteImpact, error) {
	if pathID == "" {
		return proxy.NodeAccessPathDeleteImpact{}, invalidInput("missing_path_id")
	}
	if err := s.requireTenantResourceManage(tenantCtx, func() (domain.BindingPermission, bool) {
		return s.store.NodeAccessPathBindingPermission(tenantCtx, pathID)
	}); err != nil {
		return proxy.NodeAccessPathDeleteImpact{}, err
	}
	if !tenantCtx.SuperAdmin && s.store.CountNodeAccessPathBindings(pathID) > 1 {
		return proxy.NodeAccessPathDeleteImpact{}, newError(http.StatusConflict, "shared_resource_delete_forbidden")
	}
	return s.store.GetNodeAccessPathDeleteImpact(pathID)
}

func accessPathNodeIDs(targetNodeID string, entryNodeID string, relayNodeIDs []string) []string {
	nodeIDs := make([]string, 0, len(relayNodeIDs)+2)
	if targetNodeID != "" {
		nodeIDs = append(nodeIDs, targetNodeID)
	}
	if entryNodeID != "" {
		nodeIDs = append(nodeIDs, entryNodeID)
	}
	for _, nodeID := range relayNodeIDs {
		if nodeID != "" {
			nodeIDs = append(nodeIDs, nodeID)
		}
	}
	return nodeIDs
}

type accessPathValidationInput struct {
	ChainID        string
	Name           string
	Mode           string
	Protocol       string
	ServiceType    string
	TargetNodeID   string
	EntryNodeID    string
	RelayNodeIDs   []string
	ListenHost     string
	ListenPort     int
	TargetProtocol string
	TargetHost     string
	TargetPort     int
	TLSMode        string
	AuthMode       string
}

func (s *Service) validateCreateAccessPath(tenantCtx domain.TenantAuthContext, input domain.CreateNodeAccessPathInput) error {
	return s.validateAccessPath(tenantCtx, accessPathValidationInput{
		ChainID:        input.ChainID,
		Name:           input.Name,
		Mode:           input.Mode,
		Protocol:       input.Protocol,
		ServiceType:    input.ServiceType,
		TargetNodeID:   input.TargetNodeID,
		EntryNodeID:    input.EntryNodeID,
		RelayNodeIDs:   input.RelayNodeIDs,
		ListenHost:     input.ListenHost,
		ListenPort:     input.ListenPort,
		TargetProtocol: input.TargetProtocol,
		TargetHost:     input.TargetHost,
		TargetPort:     input.TargetPort,
		TLSMode:        input.TLSMode,
		AuthMode:       input.AuthMode,
	})
}

func (s *Service) validateUpdateAccessPath(tenantCtx domain.TenantAuthContext, input domain.UpdateNodeAccessPathInput) error {
	return s.validateAccessPath(tenantCtx, accessPathValidationInput{
		ChainID:        input.ChainID,
		Name:           input.Name,
		Mode:           input.Mode,
		Protocol:       input.Protocol,
		ServiceType:    input.ServiceType,
		TargetNodeID:   input.TargetNodeID,
		EntryNodeID:    input.EntryNodeID,
		RelayNodeIDs:   input.RelayNodeIDs,
		ListenHost:     input.ListenHost,
		ListenPort:     input.ListenPort,
		TargetProtocol: input.TargetProtocol,
		TargetHost:     input.TargetHost,
		TargetPort:     input.TargetPort,
		TLSMode:        input.TLSMode,
		AuthMode:       input.AuthMode,
	})
}

func (s *Service) validateAccessPath(tenantCtx domain.TenantAuthContext, input accessPathValidationInput) error {
	if input.ChainID == "" || input.Name == "" || input.Mode == "" || input.Protocol == "" || input.ServiceType == "" || input.TargetNodeID == "" || input.EntryNodeID == "" || input.ListenHost == "" || input.TargetProtocol == "" {
		return invalidInput("invalid_access_path")
	}
	chain, ok := chainByID(s.store.ListChainsForTenant(tenantCtx), input.ChainID)
	if !ok || !chain.Enabled {
		return invalidInput("invalid_access_path")
	}
	if !validLatestPathMode(input.Mode) || !validLatestAccessProtocol(input.Protocol) || !validLatestServiceType(input.ServiceType) {
		return invalidInput("invalid_access_path")
	}
	if !validLatestTLSMode(input.TLSMode) || input.AuthMode != domain.AccessAuthProxyToken {
		return invalidInput("invalid_access_path")
	}
	if !validLatestAccessPathCombination(input.Mode, input.Protocol, input.ServiceType) {
		return invalidInput("invalid_access_path")
	}
	if input.ListenPort < 1 || input.ListenPort > 65535 || input.TargetPort < 1 || input.TargetPort > 65535 {
		return invalidInput("invalid_access_path")
	}
	if input.Mode != "forward" && input.TargetHost == "" {
		return invalidInput("invalid_access_path")
	}
	if !s.tenantEnabledNodesExist(tenantCtx, accessPathNodeIDs(input.TargetNodeID, input.EntryNodeID, input.RelayNodeIDs)) {
		return invalidInput("invalid_access_path")
	}
	return nil
}

func validLatestPathMode(value string) bool {
	return slices.Contains([]string{"forward", "reverse", "direct", "tcp", "udp"}, value)
}

func validLatestAccessProtocol(value string) bool {
	return slices.Contains([]string{"http", "https", "connect", "tcp", "udp", "quic"}, value)
}

func validLatestServiceType(value string) bool {
	return slices.Contains([]string{"http_forward_proxy", "reverse_proxy", "tcp_access", "udp_access", "direct_quic"}, value)
}

func validLatestTLSMode(value string) bool {
	return slices.Contains([]string{"", "passthrough", "terminate", "direct_verify"}, value)
}

func validLatestAccessPathCombination(mode string, protocol string, serviceType string) bool {
	switch mode {
	case "forward":
		return serviceType == "http_forward_proxy" && slices.Contains([]string{"http", "https", "connect"}, protocol)
	case "reverse":
		return serviceType == "reverse_proxy" && slices.Contains([]string{"http", "https"}, protocol)
	case "direct":
		return serviceType == "direct_quic" && protocol == "quic"
	case "tcp":
		return serviceType == "tcp_access" && protocol == "tcp"
	case "udp":
		return serviceType == "udp_access" && protocol == "udp"
	}
	return false
}

func (s *Service) tenantEnabledNodesExist(tenantCtx domain.TenantAuthContext, nodeIDs []string) bool {
	nodes := s.store.ListNodesForTenant(tenantCtx)
	for _, nodeID := range nodeIDs {
		found := false
		for _, node := range nodes {
			if node.ID == nodeID && node.Enabled {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
