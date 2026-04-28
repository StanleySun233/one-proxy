package service

import "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"

func (c *ControlPlane) ListGroups() ([]domain.Group, error) {
	return c.store.ListGroups()
}

func (c *ControlPlane) CreateGroup(input domain.CreateGroupInput) (domain.Group, error) {
	if input.Name == "" {
		return domain.Group{}, invalidInput("invalid_group_payload")
	}
	return c.store.CreateGroup(input)
}

func (c *ControlPlane) GetGroup(id string) (domain.GroupDetail, error) {
	if id == "" {
		return domain.GroupDetail{}, invalidInput("missing_group_id")
	}
	group, err := c.store.GetGroup(id)
	if err != nil {
		return domain.GroupDetail{}, err
	}
	scopes, _ := c.store.GetGroupScopes(id)
	detail := domain.GroupDetail{
		Group:  group,
		Scopes: scopes,
	}
	return detail, nil
}

func (c *ControlPlane) UpdateGroup(id string, input domain.UpdateGroupInput) (domain.Group, error) {
	if id == "" {
		return domain.Group{}, invalidInput("missing_group_id")
	}
	return c.store.UpdateGroup(id, input)
}

func (c *ControlPlane) DeleteGroup(id string) error {
	if id == "" {
		return invalidInput("missing_group_id")
	}
	return c.store.DeleteGroup(id)
}

func (c *ControlPlane) ListGroupAccounts(groupID string) ([]domain.Account, error) {
	if groupID == "" {
		return nil, invalidInput("missing_group_id")
	}
	return c.store.ListGroupAccounts(groupID)
}

func (c *ControlPlane) SetGroupAccounts(groupID string, input domain.SetGroupAccountsInput) error {
	if groupID == "" {
		return invalidInput("missing_group_id")
	}
	return c.store.SetGroupAccounts(groupID, input.AccountIDs)
}

func (c *ControlPlane) SetGroupScopes(groupID string, input domain.SetGroupScopesInput) error {
	if groupID == "" {
		return invalidInput("missing_group_id")
	}
	return c.store.SetGroupScopes(groupID, input.ScopeKeys)
}
