package store

import (
	"database/sql"
	"fmt"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func (s *MySQLStore) ListAllTenants() []domain.Tenant {
	rows, err := s.db.Query(
		`SELECT id, name, created_at, updated_at
		 FROM tenants
		 ORDER BY name`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.Tenant, 0)
	for rows.Next() {
		var item domain.Tenant
		if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt, &item.UpdatedAt); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) ListTenants(account domain.Account) []domain.Tenant {
	if account.Role == domain.AccountRoleSuperAdmin {
		return s.ListAllTenants()
	}
	rows, err := s.db.Query(
		`SELECT t.id, t.name, t.created_at, t.updated_at
		 FROM tenants t
		 JOIN tenant_memberships tm ON tm.tenant_id = t.id
		 WHERE tm.account_id = ?
		 ORDER BY t.name`,
		account.ID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.Tenant, 0)
	for rows.Next() {
		var item domain.Tenant
		if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt, &item.UpdatedAt); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) GetTenant(tenantID string) (domain.Tenant, bool) {
	var item domain.Tenant
	err := s.db.QueryRow(
		`SELECT id, name, created_at, updated_at
		 FROM tenants
		 WHERE id = ?`,
		tenantID,
	).Scan(&item.ID, &item.Name, &item.CreatedAt, &item.UpdatedAt)
	return item, err == nil
}

func (s *MySQLStore) CreateTenant(name string, initialAdminAccountID string, createID string) (domain.Tenant, error) {
	tenantID, err := s.nextTenantID()
	if err != nil {
		return domain.Tenant{}, err
	}
	now := nowRFC3339()
	item := domain.Tenant{
		ID:        tenantID,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	tx, err := s.db.Begin()
	if err != nil {
		return domain.Tenant{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`INSERT INTO tenants (id, name, created_at, updated_at)
		 VALUES (?, ?, ?, ?)`,
		item.ID, item.Name, item.CreatedAt, item.UpdatedAt,
	); err != nil {
		return domain.Tenant{}, err
	}
	if _, err := tx.Exec(
		`INSERT INTO tenant_memberships (tenant_id, account_id, role, create_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		item.ID, initialAdminAccountID, domain.TenantRoleAdmin, createID, now, now,
	); err != nil {
		return domain.Tenant{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Tenant{}, err
	}
	return item, nil
}

func (s *MySQLStore) UpdateTenant(tenantID string, name string) (domain.Tenant, error) {
	now := nowRFC3339()
	result, err := s.db.Exec(
		`UPDATE tenants SET name = ?, updated_at = ? WHERE id = ?`,
		name, now, tenantID,
	)
	if err != nil {
		return domain.Tenant{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return domain.Tenant{}, err
	}
	if affected == 0 {
		return domain.Tenant{}, sql.ErrNoRows
	}
	item, _ := s.GetTenant(tenantID)
	return item, nil
}

func (s *MySQLStore) DeleteTenant(tenantID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	statements := []string{
		"UPDATE node_health_snapshots SET policy_revision_id = NULL WHERE policy_revision_id IN (SELECT id FROM policy_revisions WHERE tenant_id = ?)",
		"DELETE FROM node_policy_assignments WHERE tenant_id = ?",
		"DELETE FROM policy_revisions WHERE tenant_id = ?",
	}
	for _, statement := range statements {
		if _, err := tx.Exec(statement, tenantID); err != nil {
			return err
		}
	}
	result, err := tx.Exec("DELETE FROM tenants WHERE id = ?", tenantID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

func (s *MySQLStore) ListTenantMemberships(accountID string) []domain.TenantMembership {
	rows, err := s.db.Query(
		`SELECT tm.tenant_id, t.name, tm.role, tm.created_at
		 FROM tenant_memberships tm
		 JOIN tenants t ON t.id = tm.tenant_id
		 WHERE tm.account_id = ?
		 ORDER BY t.name`,
		accountID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.TenantMembership, 0)
	for rows.Next() {
		var item domain.TenantMembership
		if err := rows.Scan(&item.TenantID, &item.TenantName, &item.Role, &item.JoinedAt); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) ListTenantMembers(tenantID string) []domain.TenantMembership {
	rows, err := s.db.Query(
		`SELECT tm.tenant_id, t.name, tm.role, tm.created_at
		 FROM tenant_memberships tm
		 JOIN tenants t ON t.id = tm.tenant_id
		 WHERE tm.tenant_id = ?
		 ORDER BY tm.account_id`,
		tenantID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.TenantMembership, 0)
	for rows.Next() {
		var item domain.TenantMembership
		if err := rows.Scan(&item.TenantID, &item.TenantName, &item.Role, &item.JoinedAt); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) GetTenantMembership(accountID string, tenantID string) (domain.TenantMembership, bool) {
	var item domain.TenantMembership
	err := s.db.QueryRow(
		`SELECT tm.tenant_id, t.name, tm.role, tm.created_at
		 FROM tenant_memberships tm
		 JOIN tenants t ON t.id = tm.tenant_id
		 WHERE tm.account_id = ? AND tm.tenant_id = ?`,
		accountID, tenantID,
	).Scan(&item.TenantID, &item.TenantName, &item.Role, &item.JoinedAt)
	return item, err == nil
}

func (s *MySQLStore) UpsertTenantMembership(tenantID string, accountID string, role domain.TenantRole, createID string) (domain.TenantMembership, error) {
	now := nowRFC3339()
	_, err := s.db.Exec(
		`INSERT INTO tenant_memberships (tenant_id, account_id, role, create_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE role = VALUES(role), updated_at = VALUES(updated_at)`,
		tenantID, accountID, role, createID, now, now,
	)
	if err != nil {
		return domain.TenantMembership{}, err
	}
	item, ok := s.GetTenantMembership(accountID, tenantID)
	if !ok {
		return domain.TenantMembership{}, sql.ErrNoRows
	}
	return item, nil
}

func (s *MySQLStore) DeleteTenantMembership(tenantID string, accountID string) error {
	result, err := s.db.Exec(
		`DELETE FROM tenant_memberships
		 WHERE tenant_id = ? AND account_id = ?`,
		tenantID, accountID,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *MySQLStore) nextTenantID() (string, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	now := nowRFC3339()
	var maxExisting int64
	if err := tx.QueryRow("SELECT COALESCE(MAX(CAST(id AS UNSIGNED)), 0) FROM tenants WHERE id REGEXP '^[0-9]+$'").Scan(&maxExisting); err != nil {
		return "", err
	}
	_, err = tx.Exec(
		`INSERT INTO id_sequences (name, current_value, updated_at)
		 VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE current_value = GREATEST(current_value + 1, VALUES(current_value)), updated_at = ?`,
		"tenant", maxExisting+1, now, now,
	)
	if err != nil {
		return "", err
	}

	var nextID int64
	err = tx.QueryRow(`SELECT current_value FROM id_sequences WHERE name = ?`, "tenant").Scan(&nextID)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", nextID), nil
}
