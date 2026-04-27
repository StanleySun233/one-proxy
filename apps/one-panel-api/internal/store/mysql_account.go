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
	now := time.Now().UTC()
	expiresAt := now.Add(12 * time.Hour).Format(time.RFC3339)
	_, _ = s.db.Exec(
		`INSERT INTO sessions (id, account_id, access_token_hash, refresh_token_hash, expires_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		newID("sess"), accountID, accessToken, refreshToken, expiresAt, now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	return domain.LoginResult{
		Account:            domain.Account{ID: accountID, Account: account, Role: role, Status: status, MustRotatePassword: mustRotate},
		AccessToken:        accessToken,
		RefreshToken:       refreshToken,
		ExpiresAt:          expiresAt,
		MustRotatePassword: mustRotate,
	}, true
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
	item := domain.Account{
		ID:                 newID("acct"),
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
