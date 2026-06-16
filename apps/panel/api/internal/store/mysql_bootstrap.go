package store

import (
	"context"
	"database/sql"
	"os"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/auth"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	proxy "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/domain"
)

func (s *MySQLStore) ExtensionBootstrapResourcesForTenant(tenantCtx domain.TenantAuthContext) ([]domain.Node, []proxy.Chain, []proxy.RouteRule) {
	scopedTenantCtx := tenantCtx
	scopedTenantCtx.SuperAdmin = false
	return s.ListNodesForTenant(scopedTenantCtx), s.ListChainsForTenant(scopedTenantCtx), s.ListPolicyRouteRulesForTenant(scopedTenantCtx)
}

func (s *MySQLStore) bootstrapAdmin(ctx context.Context) error {
	now := nowRFC3339()
	if err := s.ensureRole(ctx, "role-super-admin", domain.AccountRoleSuperAdmin, now); err != nil {
		return err
	}
	exists, err := s.exists(ctx, "SELECT 1 FROM accounts WHERE account = ?", "admin")
	if err != nil || exists {
		return err
	}
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		return nil
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	accountID, err := s.nextID("account")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO accounts
		 (id, account, password_hash, role_id, status, must_rotate_password, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		accountID, "admin", hash, "role-super-admin", domain.AccountStatusActive, 0, now, now,
	)
	if err == nil {
		s.bootstrapAdminPassword = password
	}
	return err
}

func (s *MySQLStore) ensureRole(ctx context.Context, id string, name string, now string) error {
	existingID, ok, err := s.roleIDByName(ctx, name)
	if err != nil {
		return err
	}
	if ok {
		if existingID != id {
			return nil
		}
	}
	exists, err := s.exists(ctx, "SELECT 1 FROM roles WHERE id = ?", id)
	if err != nil || exists {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO roles (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)",
		id, name, now, now,
	)
	return err
}

func (s *MySQLStore) roleIDByName(ctx context.Context, name string) (string, bool, error) {
	var id string
	err := s.db.QueryRowContext(ctx, "SELECT id FROM roles WHERE name = ?", name).Scan(&id)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return id, true, nil
}

func (s *MySQLStore) bootstrapConfig(ctx context.Context) error {
	exists, err := s.exists(ctx, "SELECT 1 FROM config WHERE name = ?", "jwt_signing_key")
	if err != nil || exists {
		return err
	}
	key := os.Getenv("JWT_SIGNING_KEY")
	if key == "" || key == "change-me" {
		return nil
	}
	now := nowRFC3339()
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO config (name, value, updated_at) VALUES (?, ?, ?)",
		"jwt_signing_key", key, now,
	)
	return err
}

func (s *MySQLStore) IsInitialized() bool {
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM accounts").Scan(&count); err != nil {
		return false
	}
	return count > 0
}

func (s *MySQLStore) ReinitializeStore(adminPassword string) error {
	s.bootstrapAdminPassword = adminPassword
	if err := os.Setenv("ADMIN_PASSWORD", adminPassword); err != nil {
		return err
	}
	ctx := context.Background()
	if err := s.init(ctx); err != nil {
		return err
	}
	if adminPassword != "" {
		hash, err := auth.HashPassword(adminPassword)
		if err != nil {
			return err
		}
		_, _ = s.db.ExecContext(ctx,
			"UPDATE accounts SET password_hash = ?, must_rotate_password = 0, updated_at = ? WHERE account = ?",
			hash, nowRFC3339(), "admin",
		)
	}
	return nil
}
