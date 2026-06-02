package linkservice

import "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"

func (s *Service) AccessPaths() []domain.NodeAccessPath {
	return s.store.ListNodeAccessPaths()
}

func (s *Service) CreateAccessPath(input domain.CreateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	input = normalizeCreateAccessPathInput(input)
	if err := s.validateAccessPath(input.ChainID, input.Name, input.Mode, input.Protocol, input.ServiceType, input.TargetHost, input.TargetPort, input.ListenPort, input.TLSMode, input.AuthMode); err != nil {
		return domain.NodeAccessPath{}, err
	}
	return s.store.CreateNodeAccessPath(input)
}

func (s *Service) UpdateAccessPath(pathID string, input domain.UpdateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	if pathID == "" {
		return domain.NodeAccessPath{}, invalidInput("missing_path_id")
	}
	input = normalizeUpdateAccessPathInput(input)
	if err := s.validateAccessPath(input.ChainID, input.Name, input.Mode, input.Protocol, input.ServiceType, input.TargetHost, input.TargetPort, input.ListenPort, input.TLSMode, input.AuthMode); err != nil {
		return domain.NodeAccessPath{}, err
	}
	return s.store.UpdateNodeAccessPath(pathID, input)
}

func (s *Service) DeleteAccessPath(pathID string) error {
	if pathID == "" {
		return invalidInput("missing_path_id")
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

func (s *Service) validateAccessPath(chainID string, name string, mode string, protocol string, serviceType string, targetHost string, targetPort int, listenPort int, tlsMode string, authMode string) error {
	if chainID == "" || name == "" || mode == "" {
		return invalidInput("invalid_access_path_payload")
	}
	if _, ok := chainByID(s.store.ListChains(), chainID); !ok {
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
