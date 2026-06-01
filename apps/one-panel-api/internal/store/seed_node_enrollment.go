package store

import (
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/auth"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (s *SeedStore) EnrollNode(input domain.EnrollNodeInput) (domain.EnrollNodeResult, error) {
	enrollmentSecret, _ := auth.RandomToken()
	return domain.EnrollNodeResult{
		Node: domain.Node{
			ID:           s.nextID("node"),
			Name:         input.Name,
			Mode:         input.Mode,
			ScopeKey:     input.ScopeKey,
			ParentNodeID: input.ParentNodeID,
			Enabled:      true,
			Status:       "pending",
			PublicHost:   input.PublicHost,
			PublicPort:   input.PublicPort,
		},
		EnrollmentSecret: enrollmentSecret,
		ApprovalState:    "pending",
	}, nil
}

func (s *SeedStore) ApproveNodeEnrollment(nodeID string, reviewedBy string) (domain.ApproveNodeEnrollmentResult, error) {
	accessToken, _ := auth.RandomToken()
	trustMaterial, _ := auth.RandomToken()
	return domain.ApproveNodeEnrollmentResult{
		Node: domain.Node{
			ID:       nodeID,
			Name:     nodeID,
			Mode:     "relay",
			ScopeKey: "seed-scope",
			Enabled:  true,
			Status:   "degraded",
		},
		AccessToken:   accessToken,
		TrustMaterial: trustMaterial,
		ExpiresAt:     time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339),
	}, nil
}

func (s *SeedStore) ExchangeNodeEnrollment(input domain.ExchangeNodeEnrollmentInput) (domain.ApproveNodeEnrollmentResult, error) {
	return s.ApproveNodeEnrollment(input.NodeID, "")
}

func (s *SeedStore) ListPendingNodes() []domain.Node {
	return []domain.Node{}
}

func (s *SeedStore) RejectNodeEnrollment(nodeID string, reviewedBy string, reason string) error {
	return nil
}
