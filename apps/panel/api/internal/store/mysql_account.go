package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/auth"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
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
	replacementID, err := s.replacementAccountID(accountID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("cannot_delete_last_account")
	}
	if err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	reassignmentStatements := []string{
		"UPDATE tenant_memberships SET create_id = ? WHERE create_id = ?",
		"UPDATE nodes SET create_id = ? WHERE create_id = ?",
		"UPDATE nodes SET owner_id = ? WHERE owner_id = ?",
		"UPDATE node_links SET create_id = ? WHERE create_id = ?",
		"UPDATE node_links SET owner_id = ? WHERE owner_id = ?",
		"UPDATE scopes SET create_id = ? WHERE create_id = ?",
		"UPDATE scopes SET owner_id = ? WHERE owner_id = ?",
		"UPDATE chains SET create_id = ? WHERE create_id = ?",
		"UPDATE chains SET owner_id = ? WHERE owner_id = ?",
		"UPDATE route_rule_groups SET create_id = ? WHERE create_id = ?",
		"UPDATE route_rule_groups SET owner_id = ? WHERE owner_id = ?",
		"UPDATE route_rules SET create_id = ? WHERE create_id = ?",
		"UPDATE route_rules SET owner_id = ? WHERE owner_id = ?",
		"UPDATE policy_revisions SET created_by_account_id = ? WHERE created_by_account_id = ?",
		"UPDATE tenant_nodes SET create_id = ? WHERE create_id = ?",
		"UPDATE tenant_node_links SET create_id = ? WHERE create_id = ?",
		"UPDATE tenant_chains SET create_id = ? WHERE create_id = ?",
		"UPDATE tenant_route_rule_groups SET create_id = ? WHERE create_id = ?",
		"UPDATE tenant_scopes SET create_id = ? WHERE create_id = ?",
		"UPDATE tenant_access_paths SET create_id = ? WHERE create_id = ?",
		"UPDATE node_onboarding_tasks SET requested_by_account_id = ? WHERE requested_by_account_id = ?",
	}
	if _, err := tx.Exec("DELETE FROM sessions WHERE account_id = ?", accountID); err != nil {
		return err
	}
	for _, statement := range reassignmentStatements {
		if _, err := tx.Exec(statement, replacementID, accountID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec("DELETE FROM accounts WHERE id = ?", accountID); err != nil {
		return err
	}
	return tx.Commit()
}

type accountLookupQuery struct {
	query string
	args  []any
}

func (s *MySQLStore) replacementAccountID(accountID string) (string, error) {
	queries := []accountLookupQuery{
		{
			query: `SELECT a.id
			 FROM accounts a
			 JOIN roles r ON r.id = a.role_id
			 WHERE a.id <> ? AND a.status = ? AND r.name = ?
			 ORDER BY a.id
			 LIMIT 1`,
			args: []any{accountID, domain.AccountStatusActive, domain.AccountRoleSuperAdmin},
		},
		{
			query: `SELECT id
			 FROM accounts
			 WHERE id <> ? AND status = ?
			 ORDER BY id
			 LIMIT 1`,
			args: []any{accountID, domain.AccountStatusActive},
		},
		{
			query: `SELECT id
			 FROM accounts
			 WHERE id <> ?
			 ORDER BY id
			 LIMIT 1`,
			args: []any{accountID},
		},
	}
	for _, query := range queries {
		var replacementID string
		err := s.db.QueryRow(query.query, query.args...).Scan(&replacementID)
		if err == nil && replacementID != "" {
			return replacementID, nil
		}
		if err != nil && err != sql.ErrNoRows {
			return "", err
		}
	}
	return "", sql.ErrNoRows
}

func (s *MySQLStore) defaultOwnerAccountID() (string, error) {
	queries := []accountLookupQuery{
		{
			query: `SELECT id
			 FROM accounts
			 WHERE account = ? AND status = ?
			 ORDER BY id
			 LIMIT 1`,
			args: []any{"admin", domain.AccountStatusActive},
		},
		{
			query: `SELECT a.id
			 FROM accounts a
			 JOIN roles r ON r.id = a.role_id
			 WHERE a.status = ? AND r.name = ?
			 ORDER BY a.id
			 LIMIT 1`,
			args: []any{domain.AccountStatusActive, domain.AccountRoleSuperAdmin},
		},
		{
			query: `SELECT id
			 FROM accounts
			 WHERE status = ?
			 ORDER BY id
			 LIMIT 1`,
			args: []any{domain.AccountStatusActive},
		},
		{
			query: `SELECT id
			 FROM accounts
			 ORDER BY id
			 LIMIT 1`,
		},
	}
	for _, query := range queries {
		var accountID string
		err := s.db.QueryRow(query.query, query.args...).Scan(&accountID)
		if err == nil && accountID != "" {
			return accountID, nil
		}
		if err != nil && err != sql.ErrNoRows {
			return "", err
		}
	}
	return "", sql.ErrNoRows
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
