package store

import (
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (s *SeedStore) ListPolicyRevisions() []domain.PolicyRevision {
	return []domain.PolicyRevision{}
}

func (s *SeedStore) PublishPolicy(accountID string) (domain.PolicyRevision, error) {
	_ = accountID
	policyID := s.nextID("policy_revision")
	return domain.PolicyRevision{
		ID:            policyID,
		Version:       policyID,
		Status:        "published",
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		AssignedNodes: 0,
	}, nil
}
