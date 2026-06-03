package store

import (
	"database/sql"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
	proxy "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/features/proxy/domain"
)

func (s *MySQLStore) ListScopes() []proxy.Scope {
	rows, err := s.db.Query(
		`SELECT id, create_id, owner_id, name, COALESCE(description, ''), created_at, updated_at
		 FROM scopes ORDER BY name`,
	)
	return s.scanScopes(rows, err)
}

func (s *MySQLStore) ListScopesForTenant(tenantCtx domain.TenantAuthContext) []proxy.Scope {
	if tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.TenantID == "" {
		return s.ListScopes()
	}
	rows, err := s.db.Query(
		`SELECT s.id, s.create_id, s.owner_id, s.name, COALESCE(s.description, ''), s.created_at, s.updated_at
		 FROM scopes s
		 JOIN tenant_scopes ts ON ts.scope_id = s.id
		 WHERE ts.tenant_id = ?
		 ORDER BY s.name`,
		tenantCtx.ActiveTenant.TenantID,
	)
	return s.scanScopes(rows, err)
}

func (s *MySQLStore) scanScopes(rows *sql.Rows, err error) []proxy.Scope {
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]proxy.Scope, 0)
	for rows.Next() {
		var item proxy.Scope
		if err := rows.Scan(&item.ID, &item.CreateID, &item.OwnerID, &item.Name, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) CreateScope(input proxy.CreateScopeInput) (proxy.Scope, error) {
	scopeID := strings.TrimSpace(input.ID)
	if scopeID == "" {
		var err error
		scopeID, err = s.nextID("scope")
		if err != nil {
			return proxy.Scope{}, err
		}
	}
	now := nowRFC3339()
	item := proxy.Scope{
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

func (s *MySQLStore) CreateScopeForTenant(tenantCtx domain.TenantAuthContext, input proxy.CreateScopeInput) (proxy.Scope, error) {
	scopeID := strings.TrimSpace(input.ID)
	if scopeID == "" {
		var err error
		scopeID, err = s.nextID("scope")
		if err != nil {
			return proxy.Scope{}, err
		}
	}
	now := nowRFC3339()
	item := proxy.Scope{
		ID:          scopeID,
		CreateID:    tenantCtx.Account.ID,
		OwnerID:     tenantCtx.Account.ID,
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	tx, err := s.db.Begin()
	if err != nil {
		return proxy.Scope{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`INSERT INTO scopes (id, create_id, owner_id, name, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.CreateID, item.OwnerID, item.Name, item.Description, item.CreatedAt, item.UpdatedAt,
	); err != nil {
		return proxy.Scope{}, err
	}
	if tenantCtx.ActiveTenant.TenantID != "" {
		if err := bindTenantResource(tx, "tenant_scopes", "scope_id", tenantCtx.ActiveTenant.TenantID, item.ID, tenantCtx.Account.ID); err != nil {
			return proxy.Scope{}, err
		}
	}
	return item, tx.Commit()
}

func (s *MySQLStore) UpdateScope(scopeID string, input proxy.UpdateScopeInput) (proxy.Scope, error) {
	now := nowRFC3339()
	_, err := s.db.Exec(
		`UPDATE scopes SET name = ?, description = ?, updated_at = ? WHERE id = ?`,
		strings.TrimSpace(input.Name), strings.TrimSpace(input.Description), now, scopeID,
	)
	if err != nil {
		return proxy.Scope{}, err
	}
	var item proxy.Scope
	err = s.db.QueryRow(
		`SELECT id, name, COALESCE(description, ''), created_at, updated_at
		 FROM scopes WHERE id = ?`,
		scopeID,
	).Scan(&item.ID, &item.Name, &item.Description, &item.CreatedAt, &item.UpdatedAt)
	if err == sql.ErrNoRows {
		return proxy.Scope{}, sql.ErrNoRows
	}
	return item, err
}

func (s *MySQLStore) DeleteScope(scopeID string) error {
	_, err := s.db.Exec("DELETE FROM scopes WHERE id = ?", scopeID)
	return err
}

func (s *MySQLStore) ScopeBindingPermission(tenantCtx domain.TenantAuthContext, scopeID string) (domain.BindingPermission, bool) {
	return s.tenantResourcePermission(tenantCtx, "tenant_scopes", "scope_id", scopeID)
}

func (s *MySQLStore) CountScopeBindings(scopeID string) int {
	return s.countTenantResourceBindings("tenant_scopes", "scope_id", scopeID)
}
