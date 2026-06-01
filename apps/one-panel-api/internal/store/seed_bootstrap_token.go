package store

import (
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/auth"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (s *SeedStore) CreateBootstrapToken(input domain.CreateBootstrapTokenInput) (domain.BootstrapToken, error) {
	token, _ := auth.RandomToken()
	return domain.BootstrapToken{
		ID:           s.nextID("bootstrap_token"),
		Token:        token,
		TargetType:   input.TargetType,
		TargetID:     input.TargetID,
		NodeName:     input.NodeName,
		NodeMode:     input.NodeMode,
		ScopeKey:     input.ScopeKey,
		ParentNodeID: input.ParentNodeID,
		PublicHost:   input.PublicHost,
		PublicPort:   input.PublicPort,
		ExpiresAt:    time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339),
		CreatedAt:    nowRFC3339(),
	}, nil
}

func (s *SeedStore) ListUnconsumedBootstrapTokens() []domain.BootstrapToken {
	return []domain.BootstrapToken{}
}

func (s *SeedStore) DeleteBootstrapToken(tokenID string) error {
	_ = tokenID
	return nil
}
