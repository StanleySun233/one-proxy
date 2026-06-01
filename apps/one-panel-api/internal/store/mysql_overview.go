package store

import "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"

func (s *MySQLStore) GetOverview() domain.Overview {
	nodes := s.ListNodes()
	health := s.ListNodeHealth()
	healthy := 0
	degraded := 0
	for _, node := range nodes {
		if node.Status == domain.NodeStatusHealthy {
			healthy++
		} else {
			degraded++
		}
	}
	renewSoon := 0
	for _, item := range health {
		for _, state := range item.CertStatus {
			if state == domain.CertStatusRenewSoon || state == "rotate" {
				renewSoon++
				break
			}
		}
	}
	latest := domain.OverviewPolicies{}
	_ = s.db.QueryRow(
		"SELECT version, created_at FROM policy_revisions ORDER BY created_at DESC LIMIT 1",
	).Scan(&latest.ActiveRevision, &latest.PublishedAt)
	return domain.Overview{
		Nodes:        domain.OverviewNodes{Healthy: healthy, Degraded: degraded},
		Policies:     latest,
		Certificates: domain.OverviewCertificates{RenewSoon: renewSoon},
	}
}

func (s *MySQLStore) ListCertificates() []domain.Certificate {
	rows, err := s.db.Query(
		`SELECT id, owner_type, owner_id, cert_type, provider, status, COALESCE(not_before, ''), COALESCE(not_after, '')
		 FROM certificates
		 ORDER BY owner_id, cert_type`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.Certificate, 0)
	for rows.Next() {
		var item domain.Certificate
		if err := rows.Scan(&item.ID, &item.OwnerType, &item.OwnerID, &item.CertType, &item.Provider, &item.Status, &item.NotBefore, &item.NotAfter); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}
