package store

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	proxy "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/domain"
	"github.com/uptrace/bun"
)

type proxyRepository struct {
	db  *bun.DB
	raw *sql.DB
}

func (s *MySQLStore) proxyRepository() proxyRepository {
	return proxyRepository{db: s.bunDB, raw: s.db}
}

func (r proxyRepository) listChainHops(ctx context.Context, chainID string) ([]string, error) {
	var hops []ChainHopModel
	if err := r.db.NewSelect().Model(&hops).Where("chain_id = ?", chainID).OrderExpr("hop_index").Scan(ctx); err != nil {
		return nil, err
	}
	nodeIDs := make([]string, 0, len(hops))
	for _, hop := range hops {
		nodeIDs = append(nodeIDs, hop.NodeID)
	}
	return nodeIDs, nil
}

func (r proxyRepository) listChains(ctx context.Context) ([]proxy.Chain, error) {
	var models []ChainModel
	if err := r.db.NewSelect().Model(&models).OrderExpr("name").Scan(ctx); err != nil {
		return nil, err
	}
	return r.chainModels(ctx, models)
}

func (r proxyRepository) listChainsForTenant(ctx context.Context, tenantCtx domain.TenantAuthContext) ([]proxy.Chain, error) {
	var rows []struct {
		ChainModel
		Permission string `bun:"permission"`
	}
	if err := r.db.NewSelect().
		TableExpr("chains AS c").
		ColumnExpr("c.id, c.create_id, c.owner_id, c.name, c.destination_scope, c.enabled").
		ColumnExpr("tc.permission").
		Join("JOIN tenant_chains AS tc ON tc.chain_id = c.id").
		Where("tc.tenant_id = ?", tenantCtx.ActiveTenant.TenantID).
		Where("tc.permission IN (?, ?)", domain.BindingPermissionUse, domain.BindingPermissionManage).
		OrderExpr("c.name").
		Scan(ctx, &rows); err != nil {
		return nil, err
	}
	items := make([]proxy.Chain, 0, len(rows))
	for _, row := range rows {
		item, err := r.chainModel(ctx, row.ChainModel)
		if err != nil {
			return nil, err
		}
		item.Permission = row.Permission
		items = append(items, item)
	}
	return items, nil
}

func (r proxyRepository) chainModels(ctx context.Context, models []ChainModel) ([]proxy.Chain, error) {
	items := make([]proxy.Chain, 0, len(models))
	for _, model := range models {
		item, err := r.chainModel(ctx, model)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r proxyRepository) chainModel(ctx context.Context, model ChainModel) (proxy.Chain, error) {
	hops, err := r.listChainHops(ctx, model.ID)
	if err != nil {
		return proxy.Chain{}, err
	}
	return proxy.Chain{
		ID:               model.ID,
		CreateID:         model.CreateID,
		OwnerID:          model.OwnerID,
		Name:             model.Name,
		DestinationScope: model.DestinationScope,
		Enabled:          model.Enabled,
		Hops:             hops,
	}, nil
}

func (r proxyRepository) createChain(ctx context.Context, item proxy.Chain, tenantID string) error {
	now := nowRFC3339()
	model := ChainModel{ID: item.ID, CreateID: item.CreateID, OwnerID: item.OwnerID, Name: item.Name, DestinationScope: item.DestinationScope, Enabled: item.Enabled, CreatedAt: now, UpdatedAt: now}
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(&model).Exec(ctx); err != nil {
			return err
		}
		if err := insertChainHops(ctx, tx, item.ID, item.Hops); err != nil {
			return err
		}
		if tenantID != "" {
			binding := TenantChainModel{TenantID: tenantID, ChainID: item.ID, Permission: string(domain.BindingPermissionManage), CreateID: item.CreateID, CreatedAt: now}
			if _, err := tx.NewInsert().Model(&binding).Exec(ctx); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r proxyRepository) updateChain(ctx context.Context, chainID string, input proxy.UpdateChainInput) (proxy.Chain, error) {
	now := nowRFC3339()
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewUpdate().Model((*ChainModel)(nil)).
			Set("name = ?", input.Name).
			Set("destination_scope = ?", input.DestinationScope).
			Set("enabled = ?", input.Enabled).
			Set("updated_at = ?", now).
			Where("id = ?", chainID).
			Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.NewDelete().Model((*ChainHopModel)(nil)).Where("chain_id = ?", chainID).Exec(ctx); err != nil {
			return err
		}
		return insertChainHops(ctx, tx, chainID, input.Hops)
	})
	if err != nil {
		return proxy.Chain{}, err
	}
	return r.getChain(ctx, chainID)
}

func (r proxyRepository) getChain(ctx context.Context, chainID string) (proxy.Chain, error) {
	var model ChainModel
	if err := r.db.NewSelect().Model(&model).Where("id = ?", chainID).Scan(ctx); err != nil {
		return proxy.Chain{}, err
	}
	return r.chainModel(ctx, model)
}

func insertChainHops(ctx context.Context, tx bun.Tx, chainID string, hops []string) error {
	if len(hops) == 0 {
		return nil
	}
	models := make([]ChainHopModel, 0, len(hops))
	for index, hop := range hops {
		models = append(models, ChainHopModel{ChainID: chainID, HopIndex: index, NodeID: hop})
	}
	_, err := tx.NewInsert().Model(&models).Exec(ctx)
	return err
}

func (r proxyRepository) getChainProbeResult(ctx context.Context, chainID string) (proxy.ChainProbeResult, bool) {
	var model ChainProbeResultModel
	if err := r.db.NewSelect().Model(&model).Where("chain_id = ?", chainID).Scan(ctx); err != nil {
		return proxy.ChainProbeResult{}, false
	}
	item := proxy.ChainProbeResult{
		ChainID:        model.ChainID,
		Status:         model.Status,
		Message:        model.Message,
		BlockingNodeID: model.BlockingNodeID,
		BlockingReason: model.BlockingReason,
		TargetHost:     model.TargetHost,
		TargetPort:     model.TargetPort,
		ProbedAt:       model.ProbedAt,
	}
	_ = json.Unmarshal([]byte(model.ResolvedHopsJSON), &item.ResolvedHops)
	return item, true
}

func (r proxyRepository) saveChainProbeResult(ctx context.Context, input proxy.SaveChainProbeResultInput) (proxy.ChainProbeResult, error) {
	hopsJSON, err := json.Marshal(input.ResolvedHops)
	if err != nil {
		return proxy.ChainProbeResult{}, err
	}
	model := ChainProbeResultModel{
		ChainID:          input.ChainID,
		Status:           input.Status,
		Message:          input.Message,
		ResolvedHopsJSON: string(hopsJSON),
		BlockingNodeID:   input.BlockingNodeID,
		BlockingReason:   input.BlockingReason,
		TargetHost:       input.TargetHost,
		TargetPort:       input.TargetPort,
		ProbedAt:         input.ProbedAt,
	}
	_, err = r.db.NewInsert().Model(&model).
		On("DUPLICATE KEY UPDATE").
		Set("status = VALUES(status)").
		Set("message = VALUES(message)").
		Set("resolved_hops_json = VALUES(resolved_hops_json)").
		Set("blocking_node_id = VALUES(blocking_node_id)").
		Set("blocking_reason = VALUES(blocking_reason)").
		Set("target_host = VALUES(target_host)").
		Set("target_port = VALUES(target_port)").
		Set("probed_at = VALUES(probed_at)").
		Exec(ctx)
	if err != nil {
		return proxy.ChainProbeResult{}, err
	}
	return chainProbeResultFromInput(input), nil
}

func (r proxyRepository) listRouteRules(ctx context.Context) ([]proxy.RouteRule, error) {
	var models []RouteRuleModel
	if err := r.db.NewSelect().Model(&models).OrderExpr("priority ASC").Scan(ctx); err != nil {
		return nil, err
	}
	items := make([]proxy.RouteRule, 0, len(models))
	for _, model := range models {
		items = append(items, routeRuleModel(model, ""))
	}
	return items, nil
}

func (r proxyRepository) listRouteRulesForTenant(ctx context.Context, tenantCtx domain.TenantAuthContext) ([]proxy.RouteRule, error) {
	var rows []struct {
		RouteRuleModel
		Permission string `bun:"permission"`
	}
	if err := r.db.NewSelect().
		TableExpr("route_rules AS rr").
		ColumnExpr("rr.id, rr.create_id, rr.owner_id, rr.priority, rr.match_type, rr.match_value, rr.action_type, rr.chain_id, rr.destination_scope, rr.enabled").
		ColumnExpr("trr.permission").
		Join("JOIN tenant_route_rules AS trr ON trr.route_rule_id = rr.id").
		Where("trr.tenant_id = ?", tenantCtx.ActiveTenant.TenantID).
		Where("trr.permission IN (?, ?)", domain.BindingPermissionUse, domain.BindingPermissionManage).
		OrderExpr("rr.priority ASC").
		Scan(ctx, &rows); err != nil {
		return nil, err
	}
	items := make([]proxy.RouteRule, 0, len(rows))
	for _, row := range rows {
		items = append(items, routeRuleModel(row.RouteRuleModel, row.Permission))
	}
	return items, nil
}

func (r proxyRepository) createRouteRule(ctx context.Context, item proxy.RouteRule, tenantID string) error {
	now := nowRFC3339()
	model := RouteRuleModel{ID: item.ID, Priority: item.Priority, MatchType: item.MatchType, MatchValue: item.MatchValue, ActionType: item.ActionType, ChainID: item.ChainID, DestinationScope: item.DestinationScope, Enabled: item.Enabled, CreateID: item.CreateID, OwnerID: item.OwnerID, CreatedAt: now, UpdatedAt: now}
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(&model).Exec(ctx); err != nil {
			return err
		}
		if tenantID != "" {
			binding := TenantRouteRuleModel{TenantID: tenantID, RouteRuleID: item.ID, Permission: string(domain.BindingPermissionManage), CreateID: item.CreateID, CreatedAt: now}
			if _, err := tx.NewInsert().Model(&binding).Exec(ctx); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r proxyRepository) updateRouteRule(ctx context.Context, ruleID string, input proxy.UpdateRouteRuleInput) (proxy.RouteRule, error) {
	now := nowRFC3339()
	if _, err := r.db.NewUpdate().Model((*RouteRuleModel)(nil)).
		Set("priority = ?", input.Priority).
		Set("match_type = ?", input.MatchType).
		Set("match_value = ?", input.MatchValue).
		Set("action_type = ?", input.ActionType).
		Set("chain_id = NULLIF(?, '')", input.ChainID).
		Set("destination_scope = NULLIF(?, '')", input.DestinationScope).
		Set("enabled = ?", input.Enabled).
		Set("updated_at = ?", now).
		Where("id = ?", ruleID).
		Exec(ctx); err != nil {
		return proxy.RouteRule{}, err
	}
	return r.getRouteRule(ctx, ruleID)
}

func (r proxyRepository) getRouteRule(ctx context.Context, ruleID string) (proxy.RouteRule, error) {
	var model RouteRuleModel
	if err := r.db.NewSelect().Model(&model).Where("id = ?", ruleID).Scan(ctx); err != nil {
		return proxy.RouteRule{}, err
	}
	return routeRuleModel(model, ""), nil
}

func routeRuleModel(model RouteRuleModel, permission string) proxy.RouteRule {
	return proxy.RouteRule{ID: model.ID, CreateID: model.CreateID, OwnerID: model.OwnerID, Priority: model.Priority, MatchType: model.MatchType, MatchValue: model.MatchValue, ActionType: model.ActionType, ChainID: model.ChainID, DestinationScope: model.DestinationScope, Enabled: model.Enabled, Permission: permission}
}

func (r proxyRepository) deleteRouteRule(ctx context.Context, ruleID string) error {
	tx, err := r.raw.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM tenant_route_rules WHERE route_rule_id = ?", ruleID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM route_rules WHERE id = ?", ruleID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r proxyRepository) listNodeAccessPaths(ctx context.Context) ([]domain.NodeAccessPath, error) {
	var models []NodeAccessPathModel
	if err := r.db.NewSelect().Model(&models).OrderExpr("name").Scan(ctx); err != nil {
		return nil, err
	}
	return nodeAccessPathModels(models, nil), nil
}

func (r proxyRepository) listNodeAccessPathsForTenant(ctx context.Context, tenantCtx domain.TenantAuthContext) ([]domain.NodeAccessPath, error) {
	var rows []struct {
		NodeAccessPathModel
		Permission string `bun:"permission"`
	}
	if err := r.db.NewSelect().
		TableExpr("node_access_paths AS nap").
		ColumnExpr("nap.*").
		ColumnExpr("tap.permission").
		Join("JOIN tenant_access_paths AS tap ON tap.access_path_id = nap.id").
		Where("tap.tenant_id = ?", tenantCtx.ActiveTenant.TenantID).
		Where("tap.permission IN (?, ?)", domain.BindingPermissionUse, domain.BindingPermissionManage).
		OrderExpr("nap.name").
		Scan(ctx, &rows); err != nil {
		return nil, err
	}
	items := make([]domain.NodeAccessPath, 0, len(rows))
	for _, row := range rows {
		item := nodeAccessPathModel(row.NodeAccessPathModel)
		item.Permission = row.Permission
		items = append(items, item)
	}
	return items, nil
}

func (r proxyRepository) createNodeAccessPath(ctx context.Context, item domain.NodeAccessPath, tenantID string) error {
	now := nowRFC3339()
	model := nodeAccessPathStoreModel(item, now, now)
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(&model).Exec(ctx); err != nil {
			return err
		}
		if tenantID != "" {
			binding := TenantAccessPathModel{TenantID: tenantID, AccessPathID: item.ID, Permission: string(domain.BindingPermissionManage), CreateID: item.CreateID, CreatedAt: now}
			if _, err := tx.NewInsert().Model(&binding).Exec(ctx); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r proxyRepository) updateNodeAccessPath(ctx context.Context, pathID string, input domain.UpdateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	now := nowRFC3339()
	if _, err := r.db.NewUpdate().Model((*NodeAccessPathModel)(nil)).
		Set("chain_id = NULLIF(?, '')", input.ChainID).
		Set("name = ?", input.Name).
		Set("mode = ?", input.Mode).
		Set("protocol = ?", input.Protocol).
		Set("service_type = ?", input.ServiceType).
		Set("target_node_id = NULLIF(?, '')", input.TargetNodeID).
		Set("entry_node_id = NULLIF(?, '')", input.EntryNodeID).
		Set("relay_node_ids_json = ?", encodeJSONStringSlice(input.RelayNodeIDs)).
		Set("listen_host = NULLIF(?, '')", input.ListenHost).
		Set("listen_port = ?", input.ListenPort).
		Set("target_protocol = ?", input.TargetProtocol).
		Set("target_host = NULLIF(?, '')", input.TargetHost).
		Set("target_port = ?", input.TargetPort).
		Set("target_sni = NULLIF(?, '')", input.TargetSNI).
		Set("tls_mode = ?", input.TLSMode).
		Set("auth_mode = ?", input.AuthMode).
		Set("options_json = ?", encodeJSONMap(input.Options)).
		Set("enabled = ?", input.Enabled).
		Set("updated_at = ?", now).
		Where("id = ?", pathID).
		Exec(ctx); err != nil {
		return domain.NodeAccessPath{}, err
	}
	return r.getNodeAccessPath(ctx, pathID)
}

func (r proxyRepository) getNodeAccessPath(ctx context.Context, pathID string) (domain.NodeAccessPath, error) {
	var model NodeAccessPathModel
	if err := r.db.NewSelect().Model(&model).Where("id = ?", pathID).Scan(ctx); err != nil {
		return domain.NodeAccessPath{}, err
	}
	return nodeAccessPathModel(model), nil
}

func nodeAccessPathModels(models []NodeAccessPathModel, permissions []string) []domain.NodeAccessPath {
	items := make([]domain.NodeAccessPath, 0, len(models))
	for index, model := range models {
		item := nodeAccessPathModel(model)
		if index < len(permissions) {
			item.Permission = permissions[index]
		}
		items = append(items, item)
	}
	return items
}

func nodeAccessPathModel(model NodeAccessPathModel) domain.NodeAccessPath {
	return domain.NodeAccessPath{ID: model.ID, CreateID: model.CreateID, OwnerID: model.OwnerID, ChainID: model.ChainID, Name: model.Name, Mode: model.Mode, Protocol: model.Protocol, ServiceType: model.ServiceType, TargetNodeID: model.TargetNodeID, EntryNodeID: model.EntryNodeID, RelayNodeIDs: decodeJSONStringSlice(model.RelayNodeIDsJSON), ListenHost: model.ListenHost, ListenPort: model.ListenPort, TargetProtocol: model.TargetProtocol, TargetHost: model.TargetHost, TargetPort: model.TargetPort, TargetSNI: model.TargetSNI, TLSMode: model.TLSMode, AuthMode: model.AuthMode, Options: decodeJSONMap(model.OptionsJSON), Enabled: model.Enabled}
}

func nodeAccessPathStoreModel(item domain.NodeAccessPath, createdAt string, updatedAt string) NodeAccessPathModel {
	return NodeAccessPathModel{ID: item.ID, ChainID: item.ChainID, Name: item.Name, Mode: item.Mode, Protocol: item.Protocol, ServiceType: item.ServiceType, TargetNodeID: item.TargetNodeID, EntryNodeID: item.EntryNodeID, RelayNodeIDsJSON: encodeJSONStringSlice(item.RelayNodeIDs), ListenHost: item.ListenHost, ListenPort: item.ListenPort, TargetProtocol: item.TargetProtocol, TargetHost: item.TargetHost, TargetPort: item.TargetPort, TargetSNI: item.TargetSNI, TLSMode: item.TLSMode, AuthMode: item.AuthMode, OptionsJSON: encodeJSONMap(item.Options), Enabled: item.Enabled, CreateID: item.CreateID, OwnerID: item.OwnerID, CreatedAt: createdAt, UpdatedAt: updatedAt}
}

func chainProbeResultFromInput(input proxy.SaveChainProbeResultInput) proxy.ChainProbeResult {
	return proxy.ChainProbeResult{ChainID: input.ChainID, Status: input.Status, Message: input.Message, ResolvedHops: input.ResolvedHops, BlockingNodeID: input.BlockingNodeID, BlockingReason: input.BlockingReason, TargetHost: input.TargetHost, TargetPort: input.TargetPort, ProbedAt: input.ProbedAt}
}
