package store

import (
	"context"
	"database/sql"
	"strings"

	proxy "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/domain"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/store/deleteplan"
)

const deleteImpactDetailSeparator = "\x00"

type proxyDeleteStepSpec struct {
	step          deleteplan.DeletePlanStep
	impactQueries []proxyDeleteImpactQuery
}

type proxyDeleteImpactQuery struct {
	resourceType string
	query        string
	args         []any
}

func (s *MySQLStore) GetChainDeleteImpact(chainID string) (proxy.ChainDeleteImpact, error) {
	plan, err := s.proxyRepository().buildChainDeletePlan(context.Background(), chainID, true)
	if err != nil {
		return proxy.ChainDeleteImpact{ChainID: chainID}, err
	}
	impact := chainDeleteImpactFromPlan(plan)
	if len(impact.Delete.Chain) == 0 {
		return impact, sql.ErrNoRows
	}
	return impact, nil
}

func (s *MySQLStore) GetNodeAccessPathDeleteImpact(pathID string) (proxy.NodeAccessPathDeleteImpact, error) {
	plan, err := s.proxyRepository().buildNodeAccessPathDeletePlan(context.Background(), pathID, true)
	if err != nil {
		return proxy.NodeAccessPathDeleteImpact{PathID: pathID}, err
	}
	impact := nodeAccessPathDeleteImpactFromPlan(plan)
	if len(impact.Delete.AccessPath) == 0 {
		return impact, sql.ErrNoRows
	}
	return impact, nil
}

func (r proxyRepository) buildChainDeletePlan(ctx context.Context, chainID string, includeImpact bool) (deleteplan.DeletePlan, error) {
	return r.buildDeletePlan(ctx, "chain", chainID, chainDeleteStepSpecs(chainID), includeImpact)
}

func (r proxyRepository) buildNodeAccessPathDeletePlan(ctx context.Context, pathID string, includeImpact bool) (deleteplan.DeletePlan, error) {
	return r.buildDeletePlan(ctx, "node_access_path", pathID, nodeAccessPathDeleteStepSpecs(pathID), includeImpact)
}

func (r proxyRepository) buildRouteRuleGroupDeletePlan(ctx context.Context, groupID string, includeImpact bool) (deleteplan.DeletePlan, error) {
	return r.buildDeletePlan(ctx, "route_rule_group", groupID, routeRuleGroupDeleteStepSpecs(groupID), includeImpact)
}

func chainDeleteStepSpecs(chainID string) []proxyDeleteStepSpec {
	args := []any{chainID}
	return []proxyDeleteStepSpec{
		deleteStep("route-rules", "route_rules", "chain_id = ?", args,
			impact("route_rule", `SELECT id, CONCAT(match_type, ' ', match_value), CONCAT('priority ', priority, ' - ', action_type)
			 FROM route_rules
			 WHERE chain_id = ?
			 ORDER BY priority, id`, args),
		),
		deleteStep("node-onboarding-tasks", "node_onboarding_tasks", "path_id IN (SELECT id FROM node_access_paths WHERE chain_id = ?)", args,
			impact("node_onboarding_task", `SELECT task.id, COALESCE(NULLIF(task.target_host, ''), nap.name), CONCAT(task.mode, ' - ', task.status, ' - ', nap.name)
			 FROM node_onboarding_tasks task
			 JOIN node_access_paths nap ON nap.id = task.path_id
			 WHERE nap.chain_id = ?
			 ORDER BY task.created_at DESC`, args),
		),
		deleteStep("tenant-access-paths", "tenant_access_paths", "access_path_id IN (SELECT id FROM node_access_paths WHERE chain_id = ?)", args,
			impact("access_path_tenant_binding", `SELECT CONCAT('path:', tap.tenant_id, ':', tap.access_path_id), t.name, CONCAT('path ', nap.name, ' - ', tap.permission)
			 FROM tenant_access_paths tap
			 JOIN tenants t ON t.id = tap.tenant_id
			 JOIN node_access_paths nap ON nap.id = tap.access_path_id
			 WHERE nap.chain_id = ?`, args),
		),
		deleteStep("node-access-paths", "node_access_paths", "chain_id = ?", args,
			impact("access_path", `SELECT id, name, CONCAT(protocol, ' ', COALESCE(NULLIF(listen_host, ''), '*'), ':', listen_port, ' -> ', COALESCE(target_host, ''), ':', target_port)
			 FROM node_access_paths
			 WHERE chain_id = ?
			 ORDER BY name`, args),
		),
		deleteStep("chain-probe-results", "chain_probe_results", "chain_id = ?", args,
			impact("chain_probe_result", `SELECT chain_id, status, message
			 FROM chain_probe_results
			 WHERE chain_id = ?`, args),
		),
		deleteStep("tenant-chains", "tenant_chains", "chain_id = ?", args,
			impact("chain_tenant_binding", `SELECT CONCAT('chain:', tc.tenant_id, ':', tc.chain_id), t.name, CONCAT('chain - ', tc.permission)
			 FROM tenant_chains tc
			 JOIN tenants t ON t.id = tc.tenant_id
			 WHERE tc.chain_id = ?`, args),
		),
		deleteStep("chain-hops", "chain_hops", "chain_id = ?", args,
			impact("chain_hop", `SELECT n.id, n.name, CONCAT('hop ', ch.hop_index + 1, ' - ', n.mode)
			 FROM chain_hops ch
			 JOIN nodes n ON n.id = ch.node_id
			 WHERE ch.chain_id = ?
			 ORDER BY ch.hop_index`, args),
		),
		deleteStep("chain", "chains", "id = ?", args,
			impact("chain", "SELECT id, name, '' FROM chains WHERE id = ?", args),
		),
	}
}

func routeRuleGroupDeleteStepSpecs(groupID string) []proxyDeleteStepSpec {
	args := []any{groupID}
	return []proxyDeleteStepSpec{
		deleteStep("route-rules", "route_rules", "group_id = ?", args,
			impact("route_rule", `SELECT id, CONCAT(match_type, ' ', match_value), CONCAT('priority ', priority, ' - ', action_type)
			 FROM route_rules
			 WHERE group_id = ?
			 ORDER BY priority, id`, args),
		),
		deleteStep("tenant-route-rule-groups", "tenant_route_rule_groups", "route_rule_group_id = ?", args,
			impact("route_rule_group_tenant_binding", `SELECT CONCAT('route-group:', trg.tenant_id, ':', trg.route_rule_group_id), t.name, CONCAT('route group ', rrg.name, ' - ', trg.permission)
			 FROM tenant_route_rule_groups trg
			 JOIN tenants t ON t.id = trg.tenant_id
			 JOIN route_rule_groups rrg ON rrg.id = trg.route_rule_group_id
			 WHERE trg.route_rule_group_id = ?`, args),
		),
		deleteStep("route-rule-group", "route_rule_groups", "id = ?", args,
			impact("route_rule_group", `SELECT id, name, description
			 FROM route_rule_groups
			 WHERE id = ?`, args),
		),
	}
}

func nodeAccessPathDeleteStepSpecs(pathID string) []proxyDeleteStepSpec {
	args := []any{pathID}
	return []proxyDeleteStepSpec{
		deleteStep("node-onboarding-tasks", "node_onboarding_tasks", "path_id = ?", args,
			impact("node_onboarding_task", `SELECT id, COALESCE(NULLIF(target_host, ''), mode), CONCAT(mode, ' - ', status)
			 FROM node_onboarding_tasks
			 WHERE path_id = ?
			 ORDER BY created_at DESC`, args),
		),
		deleteStep("tenant-access-paths", "tenant_access_paths", "access_path_id = ?", args,
			impact("access_path_tenant_binding", `SELECT CONCAT(tap.tenant_id, ':', tap.access_path_id), t.name, tap.permission
			 FROM tenant_access_paths tap
			 JOIN tenants t ON t.id = tap.tenant_id
			 WHERE tap.access_path_id = ?`, args),
		),
		deleteStep("node-access-path", "node_access_paths", "id = ?", args,
			impact("access_path", `SELECT id, name, CONCAT(protocol, ' ', COALESCE(NULLIF(listen_host, ''), '*'), ':', listen_port, ' -> ', COALESCE(target_host, ''), ':', target_port)
			 FROM node_access_paths
			 WHERE id = ?`, args),
		),
	}
}

func deleteStep(name string, table string, whereSQL string, args []any, impactQueries ...proxyDeleteImpactQuery) proxyDeleteStepSpec {
	return proxyDeleteStepSpec{
		step: deleteplan.DeletePlanStep{
			Name:      name,
			Table:     table,
			Operation: deleteplan.OperationDelete,
			WhereSQL:  whereSQL,
			Args:      args,
		},
		impactQueries: impactQueries,
	}
}

func impact(resourceType string, query string, args []any) proxyDeleteImpactQuery {
	return proxyDeleteImpactQuery{resourceType: resourceType, query: query, args: args}
}

func (r proxyRepository) buildDeletePlan(ctx context.Context, resourceType string, resourceID string, specs []proxyDeleteStepSpec, includeImpact bool) (deleteplan.DeletePlan, error) {
	plan := deleteplan.DeletePlan{ResourceType: resourceType, ResourceID: resourceID, Steps: make([]deleteplan.DeletePlanStep, 0, len(specs))}
	for _, spec := range specs {
		step := spec.step
		if includeImpact {
			for _, query := range spec.impactQueries {
				items, err := r.listPlanImpactItems(ctx, query)
				if err != nil {
					return plan, err
				}
				step.ExpectedImpact = append(step.ExpectedImpact, items...)
				plan.Summary = append(plan.Summary, items...)
			}
		}
		plan.Steps = append(plan.Steps, step)
	}
	return plan, nil
}

func (r proxyRepository) listPlanImpactItems(ctx context.Context, query proxyDeleteImpactQuery) ([]deleteplan.DeleteImpactItem, error) {
	rows, err := r.raw.QueryContext(ctx, query.query, query.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]deleteplan.DeleteImpactItem, 0)
	for rows.Next() {
		var id, name, detail string
		if err := rows.Scan(&id, &name, &detail); err != nil {
			return nil, err
		}
		items = append(items, planImpactItem(query.resourceType, id, name, detail))
	}
	return items, rows.Err()
}

func planImpactItem(resourceType string, id string, name string, detail string) deleteplan.DeleteImpactItem {
	if detail != "" {
		name += deleteImpactDetailSeparator + detail
	}
	return deleteplan.DeleteImpactItem{ResourceType: resourceType, ResourceID: id, DisplayName: name, Count: 1}
}

func proxyImpactItem(item deleteplan.DeleteImpactItem) proxy.DeleteImpactItem {
	name, detail, _ := strings.Cut(item.DisplayName, deleteImpactDetailSeparator)
	return proxy.DeleteImpactItem{ID: item.ResourceID, Name: name, Detail: detail}
}

func chainDeleteImpactFromPlan(plan deleteplan.DeletePlan) proxy.ChainDeleteImpact {
	impact := proxy.ChainDeleteImpact{ChainID: plan.ResourceID}
	var chainBindings []proxy.DeleteImpactItem
	var pathBindings []proxy.DeleteImpactItem
	for _, item := range planImpactItems(plan) {
		switch item.ResourceType {
		case "chain":
			impact.Delete.Chain = append(impact.Delete.Chain, proxyImpactItem(item))
		case "chain_hop":
			impact.Delete.ChainHops = append(impact.Delete.ChainHops, proxyImpactItem(item))
		case "route_rule":
			impact.Delete.RouteRules = append(impact.Delete.RouteRules, proxyImpactItem(item))
		case "access_path":
			impact.Delete.AccessPaths = append(impact.Delete.AccessPaths, proxyImpactItem(item))
		case "node_onboarding_task":
			impact.Delete.OnboardingTasks = append(impact.Delete.OnboardingTasks, proxyImpactItem(item))
		case "chain_probe_result":
			impact.Delete.ChainProbeResults = append(impact.Delete.ChainProbeResults, proxyImpactItem(item))
		case "chain_tenant_binding":
			chainBindings = append(chainBindings, proxyImpactItem(item))
		case "access_path_tenant_binding":
			pathBindings = append(pathBindings, proxyImpactItem(item))
		}
	}
	impact.Delete.TenantBindings = append(impact.Delete.TenantBindings, chainBindings...)
	impact.Delete.TenantBindings = append(impact.Delete.TenantBindings, pathBindings...)
	return impact
}

func nodeAccessPathDeleteImpactFromPlan(plan deleteplan.DeletePlan) proxy.NodeAccessPathDeleteImpact {
	impact := proxy.NodeAccessPathDeleteImpact{PathID: plan.ResourceID}
	for _, item := range planImpactItems(plan) {
		switch item.ResourceType {
		case "access_path":
			impact.Delete.AccessPath = append(impact.Delete.AccessPath, proxyImpactItem(item))
		case "node_onboarding_task":
			impact.Delete.OnboardingTasks = append(impact.Delete.OnboardingTasks, proxyImpactItem(item))
		case "access_path_tenant_binding":
			impact.Delete.TenantBindings = append(impact.Delete.TenantBindings, proxyImpactItem(item))
		}
	}
	return impact
}

func routeRuleGroupDeleteImpactFromPlan(plan deleteplan.DeletePlan) proxy.RouteRuleGroupDeleteImpact {
	impact := proxy.RouteRuleGroupDeleteImpact{GroupID: plan.ResourceID}
	for _, item := range planImpactItems(plan) {
		switch item.ResourceType {
		case "route_rule_group":
			impact.Delete.Group = append(impact.Delete.Group, proxyImpactItem(item))
		case "route_rule":
			impact.Delete.RouteRules = append(impact.Delete.RouteRules, proxyImpactItem(item))
		case "route_rule_group_tenant_binding":
			impact.Delete.TenantBindings = append(impact.Delete.TenantBindings, proxyImpactItem(item))
		}
	}
	return impact
}

func planImpactItems(plan deleteplan.DeletePlan) []deleteplan.DeleteImpactItem {
	items := make([]deleteplan.DeleteImpactItem, 0)
	for _, step := range plan.Steps {
		items = append(items, step.ExpectedImpact...)
	}
	return items
}
