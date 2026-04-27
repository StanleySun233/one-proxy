package store

import (
	"database/sql"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (s *MySQLStore) CreateGroup(input domain.CreateGroupInput) (domain.Group, error) {
	now := nowRFC3339()
	enabled := 1
	if input.Enabled != nil && !*input.Enabled {
		enabled = 0
	}
	item := domain.Group{
		ID:          newID("grp"),
		Name:        input.Name,
		Description: input.Description,
		Enabled:     enabled == 1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err := s.db.Exec(
		"INSERT INTO `groups` (id, name, description, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		item.ID, item.Name, item.Description, enabled, now, now,
	)
	return item, err
}

func (s *MySQLStore) UpdateGroup(id string, input domain.UpdateGroupInput) (domain.Group, error) {
	group, err := s.GetGroup(id)
	if err != nil {
		return domain.Group{}, err
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
	now := nowRFC3339()
	group.UpdatedAt = now
	_, err = s.db.Exec(
		"UPDATE `groups` SET name = ?, description = ?, enabled = ?, updated_at = ? WHERE id = ?",
		group.Name, group.Description, boolToInt(group.Enabled), now, id,
	)
	if err != nil {
		return domain.Group{}, err
	}
	return group, nil
}

func (s *MySQLStore) DeleteGroup(id string) error {
	_, err := s.GetGroup(id)
	if err != nil {
		return err
	}
	_, err = s.db.Exec("DELETE FROM `groups` WHERE id = ?", id)
	return err
}

func (s *MySQLStore) GetGroup(id string) (domain.Group, error) {
	var item domain.Group
	var enabled int
	err := s.db.QueryRow(
		"SELECT id, name, COALESCE(description, ''), enabled, created_at, updated_at FROM `groups` WHERE id = ?",
		id,
	).Scan(&item.ID, &item.Name, &item.Description, &enabled, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return domain.Group{}, err
	}
	item.Enabled = enabled == 1
	return item, nil
}

func (s *MySQLStore) ListGroups() ([]domain.Group, error) {
	rows, err := s.db.Query(
		"SELECT id, name, COALESCE(description, ''), enabled, created_at, updated_at FROM `groups` ORDER BY name",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := make([]domain.Group, 0)
	for rows.Next() {
		var item domain.Group
		var enabled int
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &enabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			continue
		}
		item.Enabled = enabled == 1
		groups = append(groups, item)
	}
	return groups, nil
}

func (s *MySQLStore) ListAccountGroups(accountID string) ([]domain.Group, error) {
	rows, err := s.db.Query(
		"SELECT g.id, g.name, COALESCE(g.description, ''), g.enabled, g.created_at, g.updated_at"+
			" FROM `groups` g"+
			" JOIN account_groups ag ON ag.group_id = g.id"+
			" WHERE ag.account_id = ? AND g.enabled = 1"+
			" ORDER BY g.name",
		accountID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := make([]domain.Group, 0)
	for rows.Next() {
		var item domain.Group
		var enabled int
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &enabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			continue
		}
		item.Enabled = enabled == 1
		groups = append(groups, item)
	}
	return groups, nil
}

func (s *MySQLStore) AddAccountToGroup(accountID, groupID string) error {
	_, err := s.db.Exec("INSERT INTO account_groups (account_id, group_id) VALUES (?, ?)", accountID, groupID)
	return err
}

func (s *MySQLStore) RemoveAccountFromGroup(accountID, groupID string) error {
	_, err := s.db.Exec("DELETE FROM account_groups WHERE account_id = ? AND group_id = ?", accountID, groupID)
	return err
}

func (s *MySQLStore) ListGroupAccounts(groupID string) ([]domain.Account, error) {
	rows, err := s.db.Query(
		`SELECT a.id, a.account, r.name, a.status, a.must_rotate_password
		 FROM accounts a
		 JOIN account_groups ag ON ag.account_id = a.id
		 JOIN roles r ON r.id = a.role_id
		 WHERE ag.group_id = ?
		 ORDER BY a.account`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accounts := make([]domain.Account, 0)
	for rows.Next() {
		var item domain.Account
		var mustRotate int
		if err := rows.Scan(&item.ID, &item.Account, &item.Role, &item.Status, &mustRotate); err != nil {
			continue
		}
		item.MustRotatePassword = mustRotate == 1
		accounts = append(accounts, item)
	}
	return accounts, nil
}

func (s *MySQLStore) SetGroupAccounts(groupID string, accountIDs []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM account_groups WHERE group_id = ?", groupID); err != nil {
		return err
	}
	for _, accountID := range accountIDs {
		if _, err := tx.Exec("INSERT INTO account_groups (account_id, group_id) VALUES (?, ?)", accountID, groupID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *MySQLStore) GetGroupScopes(groupID string) ([]string, error) {
	rows, err := s.db.Query(
		`SELECT scope_key FROM group_scopes WHERE group_id = ?`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	scopes := make([]string, 0)
	for rows.Next() {
		var scope string
		if err := rows.Scan(&scope); err != nil {
			continue
		}
		scopes = append(scopes, scope)
	}
	return scopes, nil
}

func (s *MySQLStore) SetGroupScopes(groupID string, scopeKeys []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM group_scopes WHERE group_id = ?", groupID); err != nil {
		return err
	}
	for _, key := range scopeKeys {
		if _, err := tx.Exec("INSERT INTO group_scopes (group_id, scope_key) VALUES (?, ?)", groupID, key); err != nil {
			return err
		}
	}
	return tx.Commit()
}
