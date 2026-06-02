package store

import (
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/features/link/domain"
	"time"
)

func (s *SeedStore) ListScopes() []link.Scope {
	return []link.Scope{}
}

func (s *SeedStore) CreateScope(input link.CreateScopeInput) (link.Scope, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	scopeID := input.ID
	if scopeID == "" {
		scopeID = s.nextID("scope")
	}
	return link.Scope{
		ID:          scopeID,
		Name:        input.Name,
		Description: input.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (s *SeedStore) UpdateScope(scopeID string, input link.UpdateScopeInput) (link.Scope, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	return link.Scope{
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
