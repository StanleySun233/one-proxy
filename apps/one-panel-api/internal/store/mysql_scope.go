package store

import (
	"database/sql"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (s *MySQLStore) ListScopes() []domain.Scope {
	rows, err := s.db.Query(
		`SELECT id, name, COALESCE(description, ''), created_at, updated_at
		 FROM scopes ORDER BY name`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.Scope, 0)
	for rows.Next() {
		var item domain.Scope
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) CreateScope(input domain.CreateScopeInput) (domain.Scope, error) {
	scopeID := strings.TrimSpace(input.ID)
	if scopeID == "" {
		var err error
		scopeID, err = s.nextID("scope")
		if err != nil {
			return domain.Scope{}, err
		}
	}
	now := nowRFC3339()
	item := domain.Scope{
		ID:          scopeID,
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err := s.db.Exec(
		`INSERT INTO scopes (id, name, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		item.ID, item.Name, item.Description, item.CreatedAt, item.UpdatedAt,
	)
	return item, err
}

func (s *MySQLStore) UpdateScope(scopeID string, input domain.UpdateScopeInput) (domain.Scope, error) {
	now := nowRFC3339()
	_, err := s.db.Exec(
		`UPDATE scopes SET name = ?, description = ?, updated_at = ? WHERE id = ?`,
		strings.TrimSpace(input.Name), strings.TrimSpace(input.Description), now, scopeID,
	)
	if err != nil {
		return domain.Scope{}, err
	}
	var item domain.Scope
	err = s.db.QueryRow(
		`SELECT id, name, COALESCE(description, ''), created_at, updated_at
		 FROM scopes WHERE id = ?`,
		scopeID,
	).Scan(&item.ID, &item.Name, &item.Description, &item.CreatedAt, &item.UpdatedAt)
	if err == sql.ErrNoRows {
		return domain.Scope{}, sql.ErrNoRows
	}
	return item, err
}

func (s *MySQLStore) DeleteScope(scopeID string) error {
	_, err := s.db.Exec("DELETE FROM scopes WHERE id = ?", scopeID)
	return err
}
