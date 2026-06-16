package store

import (
	"database/sql"

	proxy "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/domain"
)

func (s *MySQLStore) GetChainDeleteImpact(chainID string) (proxy.ChainDeleteImpact, error) {
	impact := proxy.ChainDeleteImpact{ChainID: chainID}
	var err error
	if impact.Delete.Chain, err = s.listDeleteImpactItems("SELECT id, name, '' FROM chains WHERE id = ?", chainID); err != nil {
		return impact, err
	}
	if len(impact.Delete.Chain) == 0 {
		return impact, sql.ErrNoRows
	}
	if impact.Delete.ChainHops, err = s.listDeleteImpactItems(
		`SELECT n.id, n.name, CONCAT('hop ', ch.hop_index + 1, ' - ', n.mode)
		 FROM chain_hops ch
		 JOIN nodes n ON n.id = ch.node_id
		 WHERE ch.chain_id = ?
		 ORDER BY ch.hop_index`,
		chainID,
	); err != nil {
		return impact, err
	}
	if impact.Delete.RouteRules, err = s.listDeleteImpactItems(
		`SELECT id, CONCAT(match_type, ' ', match_value), CONCAT('priority ', priority, ' - ', action_type)
		 FROM route_rules
		 WHERE chain_id = ?
		 ORDER BY priority, id`,
		chainID,
	); err != nil {
		return impact, err
	}
	if impact.Delete.AccessPaths, err = s.listDeleteImpactItems(
		`SELECT id, name, CONCAT(protocol, ' ', COALESCE(NULLIF(listen_host, ''), '*'), ':', listen_port, ' -> ', COALESCE(target_host, ''), ':', target_port)
		 FROM node_access_paths
		 WHERE chain_id = ?
		 ORDER BY name`,
		chainID,
	); err != nil {
		return impact, err
	}
	if impact.Delete.OnboardingTasks, err = s.listDeleteImpactItems(
		`SELECT task.id, COALESCE(NULLIF(task.target_host, ''), nap.name), CONCAT(task.mode, ' - ', task.status, ' - ', nap.name)
		 FROM node_onboarding_tasks task
		 JOIN node_access_paths nap ON nap.id = task.path_id
		 WHERE nap.chain_id = ?
		 ORDER BY task.created_at DESC`,
		chainID,
	); err != nil {
		return impact, err
	}
	if impact.Delete.ChainProbeResults, err = s.listDeleteImpactItems(
		`SELECT chain_id, status, message
		 FROM chain_probe_results
		 WHERE chain_id = ?`,
		chainID,
	); err != nil {
		return impact, err
	}
	if impact.Delete.TenantBindings, err = s.listDeleteImpactItems(
		`SELECT CONCAT('chain:', tc.tenant_id, ':', tc.chain_id), t.name, CONCAT('chain - ', tc.permission)
		 FROM tenant_chains tc
		 JOIN tenants t ON t.id = tc.tenant_id
		 WHERE tc.chain_id = ?
		 UNION ALL
		 SELECT CONCAT('route:', trr.tenant_id, ':', trr.route_rule_id), t.name, CONCAT('route ', rr.match_value, ' - ', trr.permission)
		 FROM tenant_route_rules trr
		 JOIN tenants t ON t.id = trr.tenant_id
		 JOIN route_rules rr ON rr.id = trr.route_rule_id
		 WHERE rr.chain_id = ?
		 UNION ALL
		 SELECT CONCAT('path:', tap.tenant_id, ':', tap.access_path_id), t.name, CONCAT('path ', nap.name, ' - ', tap.permission)
		 FROM tenant_access_paths tap
		 JOIN tenants t ON t.id = tap.tenant_id
		 JOIN node_access_paths nap ON nap.id = tap.access_path_id
		 WHERE nap.chain_id = ?`,
		chainID, chainID, chainID,
	); err != nil {
		return impact, err
	}
	return impact, nil
}

func (s *MySQLStore) GetNodeAccessPathDeleteImpact(pathID string) (proxy.NodeAccessPathDeleteImpact, error) {
	impact := proxy.NodeAccessPathDeleteImpact{PathID: pathID}
	var err error
	if impact.Delete.AccessPath, err = s.listDeleteImpactItems(
		`SELECT id, name, CONCAT(protocol, ' ', COALESCE(NULLIF(listen_host, ''), '*'), ':', listen_port, ' -> ', COALESCE(target_host, ''), ':', target_port)
		 FROM node_access_paths
		 WHERE id = ?`,
		pathID,
	); err != nil {
		return impact, err
	}
	if len(impact.Delete.AccessPath) == 0 {
		return impact, sql.ErrNoRows
	}
	if impact.Delete.OnboardingTasks, err = s.listDeleteImpactItems(
		`SELECT id, COALESCE(NULLIF(target_host, ''), mode), CONCAT(mode, ' - ', status)
		 FROM node_onboarding_tasks
		 WHERE path_id = ?
		 ORDER BY created_at DESC`,
		pathID,
	); err != nil {
		return impact, err
	}
	if impact.Delete.TenantBindings, err = s.listDeleteImpactItems(
		`SELECT CONCAT(tap.tenant_id, ':', tap.access_path_id), t.name, tap.permission
		 FROM tenant_access_paths tap
		 JOIN tenants t ON t.id = tap.tenant_id
		 WHERE tap.access_path_id = ?`,
		pathID,
	); err != nil {
		return impact, err
	}
	return impact, nil
}

func (s *MySQLStore) listDeleteImpactItems(query string, args ...any) ([]proxy.DeleteImpactItem, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]proxy.DeleteImpactItem, 0)
	for rows.Next() {
		var item proxy.DeleteImpactItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Detail); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
