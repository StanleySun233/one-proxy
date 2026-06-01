package store

import (
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (s *SeedStore) ListScopes() []domain.Scope {
	return []domain.Scope{}
}

func (s *SeedStore) CreateScope(input domain.CreateScopeInput) (domain.Scope, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	scopeID := input.ID
	if scopeID == "" {
		scopeID = s.nextID("scope")
	}
	return domain.Scope{
		ID:          scopeID,
		Name:        input.Name,
		Description: input.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (s *SeedStore) UpdateScope(scopeID string, input domain.UpdateScopeInput) (domain.Scope, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	return domain.Scope{
		ID:          scopeID,
		Name:        input.Name,
		Description: input.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (s *SeedStore) DeleteScope(scopeID string) error {
	_ = scopeID
	return nil
}
