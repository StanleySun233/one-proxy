package store

import (
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (s *SeedStore) ListNodeHealth() []domain.NodeHealth {
	return []domain.NodeHealth{}
}

func (s *SeedStore) ListNodeHealthHistory(nodeID string, window time.Duration) ([]domain.NodeHealth, error) {
	return []domain.NodeHealth{}, nil
}

func (s *SeedStore) UpsertNodeHeartbeat(input domain.NodeHeartbeatInput) (domain.NodeHealth, error) {
	return domain.NodeHealth{
		NodeID:           input.NodeID,
		HeartbeatAt:      time.Now().UTC().Format(time.RFC3339),
		PolicyRevisionID: input.PolicyRevisionID,
		ListenerStatus:   input.ListenerStatus,
		CertStatus:       input.CertStatus,
	}, nil
}

func (s *SeedStore) RenewNodeCertificate(input domain.NodeCertRenewInput) (domain.NodeCertRenewResult, error) {
	return domain.NodeCertRenewResult{
		NodeID:   input.NodeID,
		CertType: input.CertType,
		Status:   "renewed",
		NotAfter: time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339),
	}, nil
}
