package store

import (
	"context"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	proxy "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/domain"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/store/deleteplan"
)

func (s *MySQLStore) loadChainHops(chainID string) []string {
	hops, err := s.proxyRepository().listChainHops(context.Background(), chainID)
	if err != nil {
		return nil
	}
	return hops
}

func (s *MySQLStore) ListChains() []proxy.Chain {
	items, err := s.proxyRepository().listChains(context.Background())
	if err != nil {
		return nil
	}
	return items
}

func (s *MySQLStore) ListChainsForTenant(tenantCtx domain.TenantAuthContext) []proxy.Chain {
	if tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.TenantID == "" {
		return s.ListChains()
	}
	items, err := s.proxyRepository().listChainsForTenant(context.Background(), tenantCtx)
	if err != nil {
		return nil
	}
	return items
}

func (s *MySQLStore) GetChainProbeResult(chainID string) (proxy.ChainProbeResult, bool) {
	return s.proxyRepository().getChainProbeResult(context.Background(), chainID)
}

func (s *MySQLStore) SaveChainProbeResult(input proxy.SaveChainProbeResultInput) (proxy.ChainProbeResult, error) {
	return s.proxyRepository().saveChainProbeResult(context.Background(), input)
}

func (s *MySQLStore) CreateChain(input proxy.CreateChainInput) (proxy.Chain, error) {
	ownerID, err := s.defaultOwnerAccountID()
	if err != nil {
		return proxy.Chain{}, err
	}
	chainID, err := s.nextID("chain")
	if err != nil {
		return proxy.Chain{}, err
	}
	item := proxy.Chain{ID: chainID, CreateID: ownerID, OwnerID: ownerID, Name: input.Name, DestinationScope: input.DestinationScope, Enabled: true, Hops: input.Hops}
	if err := s.proxyRepository().createChain(context.Background(), item, ""); err != nil {
		return proxy.Chain{}, err
	}
	return item, nil
}

func (s *MySQLStore) CreateChainForTenant(tenantCtx domain.TenantAuthContext, input proxy.CreateChainInput) (proxy.Chain, error) {
	chainID, err := s.nextID("chain")
	if err != nil {
		return proxy.Chain{}, err
	}
	item := proxy.Chain{ID: chainID, CreateID: tenantCtx.Account.ID, OwnerID: tenantCtx.Account.ID, Name: input.Name, DestinationScope: input.DestinationScope, Enabled: true, Hops: input.Hops}
	if err := s.proxyRepository().createChain(context.Background(), item, tenantCtx.ActiveTenant.TenantID); err != nil {
		return proxy.Chain{}, err
	}
	return item, nil
}

func (s *MySQLStore) UpdateChain(chainID string, input proxy.UpdateChainInput) (proxy.Chain, error) {
	return s.proxyRepository().updateChain(context.Background(), chainID, input)
}

func (s *MySQLStore) DeleteChain(chainID string) error {
	plan, err := s.proxyRepository().buildChainDeletePlan(context.Background(), chainID, false)
	if err != nil {
		return err
	}
	_, err = deleteplan.NewMySQLExecutor(s.db).Execute(context.Background(), plan)
	return err
}

func (s *MySQLStore) ChainBindingPermission(tenantCtx domain.TenantAuthContext, chainID string) (domain.BindingPermission, bool) {
	return s.tenantResourcePermission(tenantCtx, "tenant_chains", "chain_id", chainID)
}

func (s *MySQLStore) CountChainBindings(chainID string) int {
	return s.countTenantResourceBindings("tenant_chains", "chain_id", chainID)
}
