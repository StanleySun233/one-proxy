package store

import "time"

func (s *MySQLStore) CleanupExpiredSessions() (int64, error) {
	result, err := s.db.Exec("DELETE FROM sessions WHERE expires_at <= ?", nowRFC3339())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *MySQLStore) CleanupExpiredBootstrapTokens() (int64, error) {
	result, err := s.db.Exec("DELETE FROM bootstrap_tokens WHERE expires_at <= ? OR consumed_at IS NOT NULL", nowRFC3339())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *MySQLStore) CleanupExpiredNodeTokens() (int64, error) {
	result, err := s.db.Exec("DELETE FROM node_api_tokens WHERE expires_at <= ?", nowRFC3339())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *MySQLStore) RefreshCertificateStatus(window time.Duration) error {
	now := time.Now().UTC()
	renewBefore := now.Add(window).Format(time.RFC3339)
	if _, err := s.db.Exec(
		"UPDATE certificates SET status = 'healthy', updated_at = ? WHERE not_after IS NOT NULL AND not_after > ?",
		now.Format(time.RFC3339), renewBefore,
	); err != nil {
		return err
	}
	if _, err := s.db.Exec(
		"UPDATE certificates SET status = 'expired', updated_at = ? WHERE not_after IS NOT NULL AND not_after <= ?",
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	); err != nil {
		return err
	}
	_, err := s.db.Exec(
		"UPDATE certificates SET status = 'renew-soon', updated_at = ? WHERE not_after IS NOT NULL AND not_after > ? AND not_after <= ?",
		now.Format(time.RFC3339), now.Format(time.RFC3339), renewBefore,
	)
	return err
}

func (s *MySQLStore) RefreshNodeStatus(staleAfter time.Duration) error {
	staleAt := time.Now().UTC().Add(-staleAfter).Format(time.RFC3339)
	if _, err := s.db.Exec(
		"UPDATE nodes SET status = 'degraded', updated_at = ? WHERE id IN (SELECT node_id FROM node_health_snapshots WHERE heartbeat_at <= ?)",
		nowRFC3339(), staleAt,
	); err != nil {
		return err
	}
	_, err := s.db.Exec(
		"UPDATE nodes SET status = 'healthy', updated_at = ? WHERE id IN (SELECT node_id FROM node_health_snapshots WHERE heartbeat_at > ?)",
		nowRFC3339(), staleAt,
	)
	return err
}

func (s *MySQLStore) CleanupNodeHealthHistory(retention time.Duration) (int64, error) {
	cutoff := time.Now().Add(-retention).UTC().Format(time.RFC3339)
	result, err := s.db.Exec("DELETE FROM node_health_history WHERE heartbeat_at < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
