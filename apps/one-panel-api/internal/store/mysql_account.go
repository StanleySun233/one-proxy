package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/auth"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func roleIDForName(name string) string {
	replacer := strings.NewReplacer("_", "-", " ", "-")
	return "role-" + replacer.Replace(name)
}

func (s *MySQLStore) getAccountByID(accountID string) (domain.Account, bool) {
	var item domain.Account
	var mustRotate int
	err := s.db.QueryRow(
		`SELECT a.id, a.account, r.name, a.status, a.must_rotate_password
		 FROM accounts a
		 JOIN roles r ON r.id = a.role_id
		 WHERE a.id = ?`,
		accountID,
	).Scan(&item.ID, &item.Account, &item.Role, &item.Status, &mustRotate)
	if err != nil {
		return domain.Account{}, false
	}
	item.MustRotatePassword = mustRotate == 1
	return item, true
}

func (s *MySQLStore) createSession(accountID string, account string, role string, status string, mustRotate bool) (domain.LoginResult, bool) {
	accessToken, err := auth.RandomToken()
	if err != nil {
		return domain.LoginResult{}, false
	}
	refreshToken, err := auth.RandomToken()
	if err != nil {
		return domain.LoginResult{}, false
	}
	sessionID, err := s.nextID("session")
	if err != nil {
		return domain.LoginResult{}, false
	}
	now := time.Now().UTC()
	expiresAt := now.Add(30 * 24 * time.Hour).Format(time.RFC3339)
	_, _ = s.db.Exec(
		`INSERT INTO sessions (id, account_id, access_token_hash, refresh_token_hash, expires_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sessionID, accountID, accessToken, refreshToken, expiresAt, now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	memberships := s.ListTenantMemberships(accountID)
	return domain.LoginResult{
		Account:            domain.Account{ID: accountID, Account: account, Role: role, Status: status, MustRotatePassword: mustRotate},
		AccessToken:        accessToken,
		RefreshToken:       refreshToken,
		ExpiresAt:          expiresAt,
		MustRotatePassword: mustRotate,
		TenantMemberships:  memberships,
		ActiveTenantID:     activeTenantID(memberships),
	}, true
}

func activeTenantID(memberships []domain.TenantMembership) *string {
	if len(memberships) != 1 {
		return nil
	}
	tenantID := memberships[0].TenantID
	return &tenantID
}

func (s *MySQLStore) listAllTenants() []domain.Tenant {
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
		return s.listAllTenants()
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
	tenantID, err := s.nextID("tenant")
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
	if _, err := s.db.Exec(
		`INSERT INTO tenants (id, name, created_at, updated_at)
		 VALUES (?, ?, ?, ?)`,
		item.ID, item.Name, item.CreatedAt, item.UpdatedAt,
	); err != nil {
		return domain.Tenant{}, err
	}
	if _, err := s.UpsertTenantMembership(item.ID, initialAdminAccountID, domain.TenantRoleAdmin, createID); err != nil {
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
	result, err := s.db.Exec("DELETE FROM tenants WHERE id = ?", tenantID)
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

func (s *MySQLStore) ListAccounts() []domain.Account {
	rows, err := s.db.Query(
		`SELECT a.id, a.account, r.name, a.status, a.must_rotate_password
		 FROM accounts a
		 JOIN roles r ON r.id = a.role_id
		 ORDER BY a.account`,
	)
	if err != nil {
		return nil
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
	return accounts
}

func (s *MySQLStore) CreateAccount(input domain.CreateAccountInput) (domain.Account, error) {
	roleID := roleIDForName(input.Role)
	now := nowRFC3339()
	if err := s.ensureRole(context.Background(), roleID, input.Role, now); err != nil {
		return domain.Account{}, err
	}
	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		return domain.Account{}, err
	}
	accountID, err := s.nextID("account")
	if err != nil {
		return domain.Account{}, err
	}
	item := domain.Account{
		ID:                 accountID,
		Account:            input.Account,
		Role:               input.Role,
		Status:             domain.AccountStatusActive,
		MustRotatePassword: false,
	}
	_, err = s.db.Exec(
		`INSERT INTO accounts (id, account, password_hash, role_id, status, must_rotate_password, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Account, hash, roleID, item.Status, 0, now, now,
	)
	return item, err
}

func (s *MySQLStore) UpdateAccount(accountID string, input domain.UpdateAccountInput) (domain.Account, error) {
	current, ok := s.getAccountByID(accountID)
	if !ok {
		return domain.Account{}, sql.ErrNoRows
	}
	role := current.Role
	if input.Role != "" {
		role = input.Role
	}
	status := current.Status
	if input.Status != "" {
		status = input.Status
	}
	roleID := roleIDForName(role)
	now := nowRFC3339()
	if err := s.ensureRole(context.Background(), roleID, role, now); err != nil {
		return domain.Account{}, err
	}
	if input.Password != "" {
		hash, err := auth.HashPassword(input.Password)
		if err != nil {
			return domain.Account{}, err
		}
		if _, err := s.db.Exec(
			`UPDATE accounts SET password_hash = ?, role_id = ?, status = ?, must_rotate_password = 0, updated_at = ? WHERE id = ?`,
			hash, roleID, status, now, accountID,
		); err != nil {
			return domain.Account{}, err
		}
	} else {
		if _, err := s.db.Exec(
			`UPDATE accounts SET role_id = ?, status = ?, updated_at = ? WHERE id = ?`,
			roleID, status, now, accountID,
		); err != nil {
			return domain.Account{}, err
		}
	}
	item, _ := s.getAccountByID(accountID)
	return item, nil
}

func (s *MySQLStore) DeleteAccount(accountID string) error {
	account, ok := s.getAccountByID(accountID)
	if !ok {
		return sql.ErrNoRows
	}
	if account.Account == "admin" {
		return fmt.Errorf("cannot_delete_admin")
	}
	_, err := s.db.Exec("DELETE FROM accounts WHERE id = ?", accountID)
	return err
}

func (s *MySQLStore) Authenticate(account string, password string) (domain.LoginResult, bool) {
	var (
		id         string
		name       string
		role       string
		status     string
		hash       string
		mustRotate int
	)
	err := s.db.QueryRow(
		`SELECT a.id, a.account, r.name, a.status, a.password_hash, a.must_rotate_password
		 FROM accounts a
		 JOIN roles r ON r.id = a.role_id
		 WHERE a.account = ?`,
		account,
	).Scan(&id, &name, &role, &status, &hash, &mustRotate)
	if err != nil || status != domain.AccountStatusActive || !auth.CheckPassword(hash, password) {
		return domain.LoginResult{}, false
	}
	return s.createSession(id, name, role, status, mustRotate == 1)
}

func (s *MySQLStore) AuthenticateAccessToken(accessToken string) (domain.Account, bool) {
	var (
		accountID string
		expiresAt string
	)
	err := s.db.QueryRow(
		"SELECT account_id, expires_at FROM sessions WHERE access_token_hash = ?",
		accessToken,
	).Scan(&accountID, &expiresAt)
	if err != nil {
		return domain.Account{}, false
	}
	expiry, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().UTC().After(expiry) {
		return domain.Account{}, false
	}
	item, ok := s.getAccountByID(accountID)
	if !ok || item.Status != domain.AccountStatusActive {
		return domain.Account{}, false
	}
	return item, true
}

func (s *MySQLStore) RefreshSession(refreshToken string) (domain.LoginResult, bool) {
	var (
		accountID string
		expiresAt string
	)
	err := s.db.QueryRow(
		"SELECT account_id, expires_at FROM sessions WHERE refresh_token_hash = ?",
		refreshToken,
	).Scan(&accountID, &expiresAt)
	if err != nil {
		return domain.LoginResult{}, false
	}
	expiry, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().UTC().After(expiry) {
		return domain.LoginResult{}, false
	}
	item, ok := s.getAccountByID(accountID)
	if !ok || item.Status != domain.AccountStatusActive {
		return domain.LoginResult{}, false
	}
	return s.createSession(item.ID, item.Account, item.Role, item.Status, item.MustRotatePassword)
}

func (s *MySQLStore) Logout(accessToken string) bool {
	result, err := s.db.Exec("DELETE FROM sessions WHERE access_token_hash = ?", accessToken)
	if err != nil {
		return false
	}
	affected, err := result.RowsAffected()
	return err == nil && affected > 0
}
