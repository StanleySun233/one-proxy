package store

import "context"

func (s *MySQLStore) ensureAccountRoleModel(ctx context.Context) error {
	now := nowRFC3339()
	if err := s.ensureRole(ctx, "role-user", "user", now); err != nil {
		return err
	}
	userRoleID, ok, err := s.roleIDByName(ctx, "user")
	if err != nil {
		return err
	}
	if !ok {
		userRoleID = "role-user"
	}
	if _, err := s.db.ExecContext(ctx, `
		UPDATE accounts a
		JOIN roles r ON r.id = a.role_id
		SET a.role_id = ?, a.updated_at = ?
		WHERE r.name = 'operator'
	`, userRoleID, now); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, "DELETE FROM field_enum WHERE field = 'account_role' AND value NOT IN ('super_admin', 'user')"); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, "DELETE FROM roles WHERE name = 'operator'"); err != nil {
		return err
	}
	return nil
}
