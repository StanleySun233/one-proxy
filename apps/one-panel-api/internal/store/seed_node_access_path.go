package store

import "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"

func (s *SeedStore) ListNodeAccessPaths() []domain.NodeAccessPath {
	return []domain.NodeAccessPath{}
}

func (s *SeedStore) CreateNodeAccessPath(input domain.CreateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	return domain.NodeAccessPath{
		ID:             s.nextID("node_access_path"),
		Name:           input.Name,
		Mode:           input.Mode,
		Protocol:       input.Protocol,
		ServiceType:    input.ServiceType,
		TargetNodeID:   input.TargetNodeID,
		EntryNodeID:    input.EntryNodeID,
		RelayNodeIDs:   normalizeStringSlice(input.RelayNodeIDs),
		ListenHost:     input.ListenHost,
		ListenPort:     input.ListenPort,
		TargetProtocol: input.TargetProtocol,
		TargetHost:     input.TargetHost,
		TargetPort:     input.TargetPort,
		TargetSNI:      input.TargetSNI,
		TLSMode:        input.TLSMode,
		AuthMode:       input.AuthMode,
		Options:        input.Options,
		Enabled:        true,
	}, nil
}

func (s *SeedStore) UpdateNodeAccessPath(pathID string, input domain.UpdateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	return domain.NodeAccessPath{
		ID:             pathID,
		Name:           input.Name,
		Mode:           input.Mode,
		Protocol:       input.Protocol,
		ServiceType:    input.ServiceType,
		TargetNodeID:   input.TargetNodeID,
		EntryNodeID:    input.EntryNodeID,
		RelayNodeIDs:   normalizeStringSlice(input.RelayNodeIDs),
		ListenHost:     input.ListenHost,
		ListenPort:     input.ListenPort,
		TargetProtocol: input.TargetProtocol,
		TargetHost:     input.TargetHost,
		TargetPort:     input.TargetPort,
		TargetSNI:      input.TargetSNI,
		TLSMode:        input.TLSMode,
		AuthMode:       input.AuthMode,
		Options:        input.Options,
		Enabled:        input.Enabled,
	}, nil
}

func (s *SeedStore) DeleteNodeAccessPath(pathID string) error {
	_ = pathID
	return nil
}
