package store

import (
	"context"
	"database/sql"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func (s *MySQLStore) initRemoteSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS remote_credentials (
		id VARCHAR(191) PRIMARY KEY,
		tenant_id VARCHAR(191),
		account_id VARCHAR(191) NOT NULL,
		name VARCHAR(191) NOT NULL,
		protocol VARCHAR(64) NOT NULL,
		scope VARCHAR(64) NOT NULL,
		username VARCHAR(255) NOT NULL,
		secret_type VARCHAR(64) NOT NULL,
		encrypted_payload LONGTEXT NOT NULL,
		created_at VARCHAR(64) NOT NULL,
		updated_at VARCHAR(64) NOT NULL,
		last_used_at VARCHAR(64),
		INDEX idx_remote_credentials_account (account_id, protocol),
		INDEX idx_remote_credentials_tenant (tenant_id, protocol),
		CONSTRAINT fk_remote_credentials_account_id FOREIGN KEY (account_id) REFERENCES accounts(id),
		CONSTRAINT fk_remote_credentials_tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
	)`)
	return err
}

func (s *MySQLStore) ListRemoteCredentials(account domain.Account, tenantCtx domain.TenantAuthContext, protocol string) []domain.RemoteCredential {
	conditions := []string{"protocol = ?"}
	args := []any{protocol}
	if tenantCtx.ActiveTenant.TenantID != "" {
		conditions = append(conditions, "((scope = 'personal' AND account_id = ?) OR (scope = 'tenant' AND tenant_id = ?))")
		args = append(args, account.ID, tenantCtx.ActiveTenant.TenantID)
	} else {
		conditions = append(conditions, "scope = 'personal' AND account_id = ?")
		args = append(args, account.ID)
	}
	rows, err := s.db.Query(remoteCredentialSelect()+" WHERE "+strings.Join(conditions, " AND ")+" ORDER BY scope, name", args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanRemoteCredentials(rows)
}

func (s *MySQLStore) RemoteCredential(account domain.Account, tenantCtx domain.TenantAuthContext, credentialID string) (domain.RemoteCredential, bool) {
	rows, err := s.db.Query(remoteCredentialSelect()+" WHERE id = ?", credentialID)
	if err != nil {
		return domain.RemoteCredential{}, false
	}
	defer rows.Close()
	items := scanRemoteCredentials(rows)
	if len(items) != 1 || !remoteCredentialVisible(account, tenantCtx, items[0]) {
		return domain.RemoteCredential{}, false
	}
	return items[0], true
}

func (s *MySQLStore) CreateRemoteCredential(account domain.Account, tenantCtx domain.TenantAuthContext, input domain.CreateRemoteCredentialInput) (domain.RemoteCredential, error) {
	id, err := s.nextID("remote_credential")
	if err != nil {
		return domain.RemoteCredential{}, err
	}
	now := nowRFC3339()
	tenantID := ""
	if input.Scope == domain.RemoteCredentialScopeTenant {
		tenantID = tenantCtx.ActiveTenant.TenantID
	}
	_, err = s.db.Exec(
		`INSERT INTO remote_credentials (id, tenant_id, account_id, name, protocol, scope, username, secret_type, encrypted_payload, created_at, updated_at)
		 VALUES (?, NULLIF(?, ''), ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, tenantID, account.ID, input.Name, input.Protocol, input.Scope, input.Username, input.SecretType, input.EncryptedPayload, now, now,
	)
	if err != nil {
		return domain.RemoteCredential{}, err
	}
	item, _ := s.RemoteCredential(account, tenantCtx, id)
	return item, nil
}

func (s *MySQLStore) UpdateRemoteCredential(account domain.Account, tenantCtx domain.TenantAuthContext, credentialID string, input domain.UpdateRemoteCredentialInput) (domain.RemoteCredential, error) {
	current, ok := s.RemoteCredential(account, tenantCtx, credentialID)
	if !ok {
		return domain.RemoteCredential{}, sql.ErrNoRows
	}
	if !remoteCredentialManageable(account, tenantCtx, current) {
		return domain.RemoteCredential{}, sql.ErrNoRows
	}
	_, err := s.db.Exec(
		`UPDATE remote_credentials SET name = ?, username = ?, secret_type = ?, encrypted_payload = ?, updated_at = ? WHERE id = ?`,
		input.Name, input.Username, input.SecretType, input.EncryptedPayload, nowRFC3339(), credentialID,
	)
	if err != nil {
		return domain.RemoteCredential{}, err
	}
	item, _ := s.RemoteCredential(account, tenantCtx, credentialID)
	return item, nil
}

func (s *MySQLStore) DeleteRemoteCredential(account domain.Account, tenantCtx domain.TenantAuthContext, credentialID string) error {
	current, ok := s.RemoteCredential(account, tenantCtx, credentialID)
	if !ok || !remoteCredentialManageable(account, tenantCtx, current) {
		return sql.ErrNoRows
	}
	_, err := s.db.Exec("DELETE FROM remote_credentials WHERE id = ?", credentialID)
	return err
}

func (s *MySQLStore) TouchRemoteCredential(credentialID string) error {
	_, err := s.db.Exec("UPDATE remote_credentials SET last_used_at = ? WHERE id = ?", nowRFC3339(), credentialID)
	return err
}

func remoteCredentialSelect() string {
	return `SELECT id, COALESCE(tenant_id, ''), account_id, name, protocol, scope, username, secret_type, encrypted_payload, created_at, updated_at, COALESCE(last_used_at, '') FROM remote_credentials`
}

func scanRemoteCredentials(rows *sql.Rows) []domain.RemoteCredential {
	items := make([]domain.RemoteCredential, 0)
	for rows.Next() {
		var item domain.RemoteCredential
		if err := rows.Scan(&item.ID, &item.TenantID, &item.AccountID, &item.Name, &item.Protocol, &item.Scope, &item.Username, &item.SecretType, &item.EncryptedPayload, &item.CreatedAt, &item.UpdatedAt, &item.LastUsedAt); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

func remoteCredentialVisible(account domain.Account, tenantCtx domain.TenantAuthContext, item domain.RemoteCredential) bool {
	if item.Scope == domain.RemoteCredentialScopePersonal {
		return item.AccountID == account.ID
	}
	return item.Scope == domain.RemoteCredentialScopeTenant && tenantCtx.ActiveTenant.TenantID != "" && item.TenantID == tenantCtx.ActiveTenant.TenantID
}

func remoteCredentialManageable(account domain.Account, tenantCtx domain.TenantAuthContext, item domain.RemoteCredential) bool {
	if item.Scope == domain.RemoteCredentialScopePersonal {
		return item.AccountID == account.ID
	}
	return item.Scope == domain.RemoteCredentialScopeTenant && (tenantCtx.SuperAdmin || tenantCtx.ActiveTenant.Role == domain.TenantRoleAdmin)
}
