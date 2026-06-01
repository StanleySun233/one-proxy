package store

import "time"

func (s *SeedStore) CleanupExpiredSessions() (int64, error) {
	return 0, nil
}

func (s *SeedStore) CleanupExpiredBootstrapTokens() (int64, error) {
	return 0, nil
}

func (s *SeedStore) CleanupExpiredNodeTokens() (int64, error) {
	return 0, nil
}

func (s *SeedStore) CleanupNodeHealthHistory(retention time.Duration) (int64, error) {
	return 0, nil
}

func (s *SeedStore) RefreshCertificateStatus(window time.Duration) error {
	_ = window
	return nil
}

func (s *SeedStore) RefreshNodeStatus(staleAfter time.Duration) error {
	_ = staleAfter
	return nil
}
