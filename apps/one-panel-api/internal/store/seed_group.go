package store

import (
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (s *SeedStore) CreateGroup(input domain.CreateGroupInput) (domain.Group, error) {
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	now := time.Now().UTC().Format(time.RFC3339)
	return domain.Group{
		ID:          s.nextID("group"),
		Name:        input.Name,
		Description: input.Description,
		Enabled:     enabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (s *SeedStore) UpdateGroup(id string, input domain.UpdateGroupInput) (domain.Group, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	group := domain.Group{
		ID:          id,
		Name:        "updated",
		Description: "updated",
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if input.Name != nil {
		group.Name = *input.Name
	}
	if input.Description != nil {
		group.Description = *input.Description
	}
	if input.Enabled != nil {
		group.Enabled = *input.Enabled
	}
	return group, nil
}

func (s *SeedStore) DeleteGroup(id string) error {
	_ = id
	return nil
}

func (s *SeedStore) GetGroup(id string) (domain.Group, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	return domain.Group{
		ID:          id,
		Name:        "seed-group",
		Description: "",
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (s *SeedStore) ListGroups() ([]domain.Group, error) {
	return []domain.Group{}, nil
}

func (s *SeedStore) ListAccountGroups(accountID string) ([]domain.Group, error) {
	return []domain.Group{}, nil
}

func (s *SeedStore) GetGroupScopes(groupID string) ([]string, error) {
	return []string{}, nil
}

func (s *SeedStore) AddAccountToGroup(accountID, groupID string) error {
	return nil
}

func (s *SeedStore) RemoveAccountFromGroup(accountID, groupID string) error {
	return nil
}

func (s *SeedStore) ListGroupAccounts(groupID string) ([]domain.Account, error) {
	return []domain.Account{}, nil
}

func (s *SeedStore) SetGroupAccounts(groupID string, accountIDs []string) error {
	return nil
}

func (s *SeedStore) SetGroupScopes(groupID string, scopeKeys []string) error {
	return nil
}
