package store

import (
	"context"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/store/deleteplan"
)

func (s *MySQLStore) ListNodeAccessPaths() []domain.NodeAccessPath {
	items, err := s.proxyRepository().listNodeAccessPaths(context.Background())
	if err != nil {
		return nil
	}
	return items
}

func (s *MySQLStore) ListNodeAccessPathsForTenant(tenantCtx domain.TenantAuthContext) []domain.NodeAccessPath {
	if tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.TenantID == "" {
		return s.ListNodeAccessPaths()
	}
	items, err := s.proxyRepository().listNodeAccessPathsForTenant(context.Background(), tenantCtx)
	if err != nil {
		return nil
	}
	return items
}

func (s *MySQLStore) CreateNodeAccessPath(input domain.CreateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	ownerID, err := s.defaultOwnerAccountID()
	if err != nil {
		return domain.NodeAccessPath{}, err
	}
	pathID, err := s.nextID("node_access_path")
	if err != nil {
		return domain.NodeAccessPath{}, err
	}
	item := domain.NodeAccessPath{
		ID:             pathID,
		CreateID:       ownerID,
		OwnerID:        ownerID,
		ChainID:        input.ChainID,
		Name:           input.Name,
		Mode:           input.Mode,
		Protocol:       input.Protocol,
		ServiceType:    input.ServiceType,
		TargetNodeID:   input.TargetNodeID,
		EntryNodeID:    input.EntryNodeID,
		RelayNodeIDs:   normalizeStringSlice(input.RelayNodeIDs),
		ListenHost:     input.ListenHost,
		ListenPort:     input.ListenPort,
		TargetProtocol: input.TargetProtocol,
		TargetHost:     input.TargetHost,
		TargetPort:     input.TargetPort,
		TargetSNI:      input.TargetSNI,
		TLSMode:        input.TLSMode,
		AuthMode:       input.AuthMode,
		Options:        input.Options,
		Enabled:        true,
	}
	return item, s.proxyRepository().createNodeAccessPath(context.Background(), item, "")
}

func (s *MySQLStore) CreateNodeAccessPathForTenant(tenantCtx domain.TenantAuthContext, input domain.CreateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	pathID, err := s.nextID("node_access_path")
	if err != nil {
		return domain.NodeAccessPath{}, err
	}
	item := domain.NodeAccessPath{
		ID:             pathID,
		CreateID:       tenantCtx.Account.ID,
		OwnerID:        tenantCtx.Account.ID,
		ChainID:        input.ChainID,
		Name:           input.Name,
		Mode:           input.Mode,
		Protocol:       input.Protocol,
		ServiceType:    input.ServiceType,
		TargetNodeID:   input.TargetNodeID,
		EntryNodeID:    input.EntryNodeID,
		RelayNodeIDs:   normalizeStringSlice(input.RelayNodeIDs),
		ListenHost:     input.ListenHost,
		ListenPort:     input.ListenPort,
		TargetProtocol: input.TargetProtocol,
		TargetHost:     input.TargetHost,
		TargetPort:     input.TargetPort,
		TargetSNI:      input.TargetSNI,
		TLSMode:        input.TLSMode,
		AuthMode:       input.AuthMode,
		Options:        input.Options,
		Enabled:        true,
	}
	return item, s.proxyRepository().createNodeAccessPath(context.Background(), item, tenantCtx.ActiveTenant.TenantID)
}

func (s *MySQLStore) UpdateNodeAccessPath(pathID string, input domain.UpdateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	return s.proxyRepository().updateNodeAccessPath(context.Background(), pathID, input)
}

func (s *MySQLStore) DeleteNodeAccessPath(pathID string) error {
	plan, err := s.proxyRepository().buildNodeAccessPathDeletePlan(context.Background(), pathID, false)
	if err != nil {
		return err
	}
	_, err = deleteplan.NewMySQLExecutor(s.db).Execute(context.Background(), plan)
	return err
}

func (s *MySQLStore) NodeAccessPathBindingPermission(tenantCtx domain.TenantAuthContext, pathID string) (domain.BindingPermission, bool) {
	return s.tenantResourcePermission(tenantCtx, "tenant_access_paths", "access_path_id", pathID)
}

func (s *MySQLStore) CountNodeAccessPathBindings(pathID string) int {
	return s.countTenantResourceBindings("tenant_access_paths", "access_path_id", pathID)
}
