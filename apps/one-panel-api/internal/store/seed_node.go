package store

import (
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (s *SeedStore) ListNodes() []domain.Node {
	return []domain.Node{}
}

func (s *SeedStore) ListNodeTransports() []domain.NodeTransport {
	return []domain.NodeTransport{}
}

func (s *SeedStore) UpsertNodeTransport(input domain.UpsertNodeTransportInput) (domain.NodeTransport, error) {
	return domain.NodeTransport{
		ID:              s.nextID("node_transport"),
		NodeID:          input.NodeID,
		TransportType:   input.TransportType,
		Direction:       input.Direction,
		Address:         input.Address,
		Status:          input.Status,
		ParentNodeID:    input.ParentNodeID,
		ConnectedAt:     input.ConnectedAt,
		LastHeartbeatAt: input.LastHeartbeatAt,
		LatencyMs:       input.LatencyMs,
		Details:         input.Details,
	}, nil
}

func (s *SeedStore) CreateNode(input domain.CreateNodeInput) (domain.Node, error) {
	return domain.Node{
		ID:           s.nextID("node"),
		Name:         input.Name,
		Mode:         input.Mode,
		ScopeKey:     input.ScopeKey,
		ParentNodeID: input.ParentNodeID,
		Enabled:      true,
		Status:       "healthy",
		PublicHost:   input.PublicHost,
		PublicPort:   input.PublicPort,
	}, nil
}

func (s *SeedStore) ProvisionNodeAccess(nodeID string) (domain.ApproveNodeEnrollmentResult, error) {
	return domain.ApproveNodeEnrollmentResult{
		Node: domain.Node{
			ID:       nodeID,
			Name:     "seed-node",
			Mode:     "relay",
			ScopeKey: "seed-scope",
			Enabled:  true,
			Status:   "healthy",
		},
		AccessToken:   "seed-node-token",
		TrustMaterial: "seed-shared-secret",
		ExpiresAt:     time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339),
	}, nil
}

func (s *SeedStore) UpdateNode(nodeID string, input domain.UpdateNodeInput) (domain.Node, error) {
	return domain.Node{
		ID:           nodeID,
		Name:         input.Name,
		Mode:         input.Mode,
		ScopeKey:     input.ScopeKey,
		ParentNodeID: input.ParentNodeID,
		Enabled:      input.Enabled,
		Status:       input.Status,
		PublicHost:   input.PublicHost,
		PublicPort:   input.PublicPort,
	}, nil
}

func (s *SeedStore) DeleteNode(nodeID string) error {
	_ = nodeID
	return nil
}

func (s *SeedStore) AuthenticateNodeToken(accessToken string) (string, bool) {
	_ = accessToken
	return "", false
}

func (s *SeedStore) GetNodeAgentPolicy(nodeID string) (domain.NodeAgentPolicy, bool) {
	_ = nodeID
	return domain.NodeAgentPolicy{}, false
}
