package service

import "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"

func (c *ControlPlane) NodeAccessPaths() []domain.NodeAccessPath {
	return c.store.ListNodeAccessPaths()
}

func (c *ControlPlane) CreateNodeAccessPath(input domain.CreateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	input = normalizeCreateNodeAccessPathInput(input)
	if err := c.validateNodeAccessPath(input.Name, input.Mode, input.Protocol, input.ServiceType, input.TargetHost, input.TargetPort, input.ListenPort, input.TLSMode, input.AuthMode); err != nil {
		return domain.NodeAccessPath{}, err
	}
	return c.store.CreateNodeAccessPath(input)
}

func (c *ControlPlane) UpdateNodeAccessPath(pathID string, input domain.UpdateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	if pathID == "" {
		return domain.NodeAccessPath{}, invalidInput("missing_path_id")
	}
	input = normalizeUpdateNodeAccessPathInput(input)
	if err := c.validateNodeAccessPath(input.Name, input.Mode, input.Protocol, input.ServiceType, input.TargetHost, input.TargetPort, input.ListenPort, input.TLSMode, input.AuthMode); err != nil {
		return domain.NodeAccessPath{}, err
	}
	return c.store.UpdateNodeAccessPath(pathID, input)
}

func (c *ControlPlane) DeleteNodeAccessPath(pathID string) error {
	if pathID == "" {
		return invalidInput("missing_path_id")
	}
	return c.store.DeleteNodeAccessPath(pathID)
}

func normalizeCreateNodeAccessPathInput(input domain.CreateNodeAccessPathInput) domain.CreateNodeAccessPathInput {
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

func normalizeUpdateNodeAccessPathInput(input domain.UpdateNodeAccessPathInput) domain.UpdateNodeAccessPathInput {
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

func (c *ControlPlane) validateNodeAccessPath(name string, mode string, protocol string, serviceType string, targetHost string, targetPort int, listenPort int, tlsMode string, authMode string) error {
	if name == "" || mode == "" {
		return invalidInput("invalid_node_access_path_payload")
	}
	if !c.isValidEnum("path_mode", mode) {
		return invalidInput("invalid_node_access_path_payload")
	}
	if !c.isValidEnum("access_protocol", protocol) || !c.isValidEnum("access_service_type", serviceType) {
		return invalidInput("invalid_node_access_path_payload")
	}
	if tlsMode != "" && !c.isValidEnum("tls_mode", tlsMode) {
		return invalidInput("invalid_node_access_path_payload")
	}
	if authMode != "" && !c.isValidEnum("access_auth_mode", authMode) {
		return invalidInput("invalid_node_access_path_payload")
	}
	switch mode {
	case domain.PathModeDirect, domain.PathModeRelayChain:
		if targetHost == "" || targetPort <= 0 {
			return invalidInput("invalid_node_access_path_payload")
		}
	}
	switch protocol {
	case domain.AccessProtocolTCP, domain.AccessProtocolTLS, domain.AccessProtocolSSH, domain.AccessProtocolRDP, domain.AccessProtocolSocks5, domain.AccessProtocolSS5, domain.AccessProtocolUDP:
		if listenPort < 0 || targetPort <= 0 {
			return invalidInput("invalid_node_access_path_payload")
		}
	}
	return nil
}
