package store

import (
	"fmt"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

type SeedStore struct {
	adminPassword string
	sequences     map[string]int64
}

func NewSeedStore() *SeedStore {
	return &SeedStore{adminPassword: "admin", sequences: make(map[string]int64)}
}

func (s *SeedStore) nextID(name string) string {
	s.sequences[name]++
	return fmt.Sprintf("%d", s.sequences[name])
}

func (s *SeedStore) IsInitialized() bool {
	return false
}

func (s *SeedStore) ReinitializeStore(adminPassword string) error {
	s.adminPassword = adminPassword
	return nil
}

func (s *SeedStore) BootstrapAdminPassword() string {
	return s.adminPassword
}

func (s *SeedStore) GetOverview() domain.Overview {
	return domain.Overview{
		Nodes: domain.OverviewNodes{
			Healthy:  0,
			Degraded: 0,
		},
		Policies: domain.OverviewPolicies{},
		Certificates: domain.OverviewCertificates{
			RenewSoon: 0,
		},
	}
}

func (s *SeedStore) ListCertificates() []domain.Certificate {
	return []domain.Certificate{}
}
