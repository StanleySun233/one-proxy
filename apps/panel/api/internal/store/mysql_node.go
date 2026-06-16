package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func (s *MySQLStore) ListNodes() []domain.Node {
	rows, err := s.db.Query(
		`SELECT id, create_id, owner_id, name, mode, scope_key, COALESCE(parent_node_id, ''), enabled, status, COALESCE(public_host, ''), COALESCE(public_port, 0)
		 FROM nodes ORDER BY name`,
	)
	return s.scanNodes(rows, err)
}

func (s *MySQLStore) ListNodesForTenant(tenantCtx domain.TenantAuthContext) []domain.Node {
	if tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.TenantID == "" {
		return s.ListNodes()
	}
	rows, err := s.db.Query(
		`SELECT n.id, n.create_id, n.owner_id, n.name, n.mode, n.scope_key, COALESCE(n.parent_node_id, ''), n.enabled, n.status, COALESCE(n.public_host, ''), COALESCE(n.public_port, 0),
		        CASE WHEN MAX(visible.permission_rank) = 2 THEN ? ELSE ? END AS permission
		 FROM nodes n
		 JOIN (
		   SELECT tn.node_id, CASE WHEN tn.permission = ? THEN 2 ELSE 1 END AS permission_rank
		     FROM tenant_nodes tn
		    WHERE tn.tenant_id = ? AND tn.permission IN (?, ?)
		   UNION ALL
		   SELECT ch.node_id, 1
		     FROM tenant_chains tc
		     JOIN chain_hops ch ON ch.chain_id = tc.chain_id
		    WHERE tc.tenant_id = ? AND tc.permission IN (?, ?)
		   UNION ALL
		   SELECT nl.source_node_id, 1
		     FROM tenant_node_links tnl
		     JOIN node_links nl ON nl.id = tnl.node_link_id
		    WHERE tnl.tenant_id = ? AND tnl.permission IN (?, ?)
		   UNION ALL
		   SELECT nl.target_node_id, 1
		     FROM tenant_node_links tnl
		     JOIN node_links nl ON nl.id = tnl.node_link_id
		    WHERE tnl.tenant_id = ? AND tnl.permission IN (?, ?)
		   UNION ALL
		   SELECT nap.target_node_id, 1
		     FROM tenant_access_paths tap
		     JOIN node_access_paths nap ON nap.id = tap.access_path_id
		    WHERE tap.tenant_id = ? AND tap.permission IN (?, ?) AND nap.target_node_id IS NOT NULL
		   UNION ALL
		   SELECT nap.entry_node_id, 1
		     FROM tenant_access_paths tap
		     JOIN node_access_paths nap ON nap.id = tap.access_path_id
		    WHERE tap.tenant_id = ? AND tap.permission IN (?, ?) AND nap.entry_node_id IS NOT NULL
		   UNION ALL
		   SELECT relay.id, 1
		     FROM tenant_access_paths tap
		     JOIN node_access_paths nap ON nap.id = tap.access_path_id
		     JOIN nodes relay ON JSON_CONTAINS(nap.relay_node_ids_json, JSON_QUOTE(relay.id))
		    WHERE tap.tenant_id = ? AND tap.permission IN (?, ?)
		   UNION ALL
		   SELECT n_scope.id, 1
		     FROM tenant_scopes ts
		     JOIN scopes sc ON sc.id = ts.scope_id
		     JOIN nodes n_scope ON n_scope.scope_key = sc.id
		    WHERE ts.tenant_id = ? AND ts.permission IN (?, ?)
		 ) visible ON visible.node_id = n.id
		 GROUP BY n.id, n.create_id, n.owner_id, n.name, n.mode, n.scope_key, n.parent_node_id, n.enabled, n.status, n.public_host, n.public_port
		 ORDER BY n.name`,
		domain.BindingPermissionManage, domain.BindingPermissionUse,
		domain.BindingPermissionManage,
		tenantCtx.ActiveTenant.TenantID, domain.BindingPermissionUse, domain.BindingPermissionManage,
		tenantCtx.ActiveTenant.TenantID, domain.BindingPermissionUse, domain.BindingPermissionManage,
		tenantCtx.ActiveTenant.TenantID, domain.BindingPermissionUse, domain.BindingPermissionManage,
		tenantCtx.ActiveTenant.TenantID, domain.BindingPermissionUse, domain.BindingPermissionManage,
		tenantCtx.ActiveTenant.TenantID, domain.BindingPermissionUse, domain.BindingPermissionManage,
		tenantCtx.ActiveTenant.TenantID, domain.BindingPermissionUse, domain.BindingPermissionManage,
		tenantCtx.ActiveTenant.TenantID, domain.BindingPermissionUse, domain.BindingPermissionManage,
		tenantCtx.ActiveTenant.TenantID, domain.BindingPermissionUse, domain.BindingPermissionManage,
	)
	return s.scanNodes(rows, err)
}

func (s *MySQLStore) scanNodes(rows *sql.Rows, err error) []domain.Node {
	if err != nil {
		return nil
	}
	defer rows.Close()
	columns, _ := rows.Columns()
	hasPermission := len(columns) == 12
	items := make([]domain.Node, 0)
	for rows.Next() {
		var item domain.Node
		var enabled int
		if hasPermission {
			if err := rows.Scan(&item.ID, &item.CreateID, &item.OwnerID, &item.Name, &item.Mode, &item.ScopeKey, &item.ParentNodeID, &enabled, &item.Status, &item.PublicHost, &item.PublicPort, &item.Permission); err != nil {
				continue
			}
		} else {
			if err := rows.Scan(&item.ID, &item.CreateID, &item.OwnerID, &item.Name, &item.Mode, &item.ScopeKey, &item.ParentNodeID, &enabled, &item.Status, &item.PublicHost, &item.PublicPort); err != nil {
				continue
			}
		}
		item.Enabled = enabled == 1
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) CreateNode(input domain.CreateNodeInput) (domain.Node, error) {
	ownerID, err := s.defaultOwnerAccountID()
	if err != nil {
		return domain.Node{}, err
	}
	nodeID, err := s.nextNodeID()
	if err != nil {
		return domain.Node{}, err
	}
	item := domain.Node{
		ID:           nodeID,
		CreateID:     ownerID,
		OwnerID:      ownerID,
		Name:         input.Name,
		Mode:         input.Mode,
		ScopeKey:     input.ScopeKey,
		ParentNodeID: input.ParentNodeID,
		Enabled:      true,
		Status:       domain.NodeStatusHealthy,
		PublicHost:   input.PublicHost,
		PublicPort:   input.PublicPort,
	}
	now := nowRFC3339()
	_, err = s.db.Exec(
		`INSERT INTO nodes (id, create_id, owner_id, name, mode, public_host, public_port, scope_key, parent_node_id, enabled, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), ?, ?, ?, ?)`,
		item.ID, item.CreateID, item.OwnerID, item.Name, item.Mode, item.PublicHost, item.PublicPort, item.ScopeKey, item.ParentNodeID, 1, item.Status, now, now,
	)
	return item, err
}

func (s *MySQLStore) CreateNodeForTenant(tenantCtx domain.TenantAuthContext, input domain.CreateNodeInput) (domain.Node, error) {
	nodeID, err := s.nextNodeID()
	if err != nil {
		return domain.Node{}, err
	}
	item := domain.Node{
		ID:           nodeID,
		CreateID:     tenantCtx.Account.ID,
		OwnerID:      tenantCtx.Account.ID,
		Name:         input.Name,
		Mode:         input.Mode,
		ScopeKey:     input.ScopeKey,
		ParentNodeID: input.ParentNodeID,
		Enabled:      true,
		Status:       domain.NodeStatusHealthy,
		PublicHost:   input.PublicHost,
		PublicPort:   input.PublicPort,
	}
	now := nowRFC3339()
	tx, err := s.db.Begin()
	if err != nil {
		return domain.Node{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`INSERT INTO nodes (id, create_id, owner_id, name, mode, public_host, public_port, scope_key, parent_node_id, enabled, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), ?, ?, ?, ?)`,
		item.ID, item.CreateID, item.OwnerID, item.Name, item.Mode, item.PublicHost, item.PublicPort, item.ScopeKey, item.ParentNodeID, 1, item.Status, now, now,
	); err != nil {
		return domain.Node{}, err
	}
	if tenantCtx.ActiveTenant.TenantID != "" {
		if err := bindTenantResource(tx, "tenant_nodes", "node_id", tenantCtx.ActiveTenant.TenantID, item.ID, tenantCtx.Account.ID); err != nil {
			return domain.Node{}, err
		}
	}
	return item, tx.Commit()
}

func (s *MySQLStore) UpdateNode(nodeID string, input domain.UpdateNodeInput) (domain.Node, error) {
	now := nowRFC3339()
	_, err := s.db.Exec(
		`UPDATE nodes
		 SET name = ?, mode = ?, public_host = NULLIF(?, ''), public_port = ?, scope_key = ?, parent_node_id = NULLIF(?, ''), enabled = ?, status = ?, updated_at = ?
		 WHERE id = ?`,
		input.Name, input.Mode, input.PublicHost, input.PublicPort, input.ScopeKey, input.ParentNodeID, boolToInt(input.Enabled), input.Status, now, nodeID,
	)
	if err != nil {
		return domain.Node{}, err
	}
	for _, item := range s.ListNodes() {
		if item.ID == nodeID {
			return item, nil
		}
	}
	return domain.Node{}, sql.ErrNoRows
}

func (s *MySQLStore) GetNodeDeleteImpact(nodeID string) (domain.NodeDeleteImpact, error) {
	impact := domain.NodeDeleteImpact{NodeID: nodeID}
	chainIDs, err := nodeChainIDs(s.db, nodeID)
	if err != nil {
		return impact, err
	}
	pathCondition, pathArgs := nodeDeleteAccessPathCondition(nodeID, chainIDs)

	if impact.Delete.Node, err = s.countNodeDeleteRows("SELECT COUNT(*) FROM nodes WHERE id = ?", nodeID); err != nil {
		return impact, err
	}
	impact.Delete.Chains = len(chainIDs)
	if impact.Delete.ChainHops, err = s.countNodeDeleteRowsIn("SELECT COUNT(*) FROM chain_hops WHERE chain_id IN (%s)", chainIDs); err != nil {
		return impact, err
	}
	if impact.Delete.RouteRules, err = s.countNodeDeleteRowsIn("SELECT COUNT(*) FROM route_rules WHERE chain_id IN (%s)", chainIDs); err != nil {
		return impact, err
	}
	if impact.Delete.AccessPaths, err = s.countNodeDeleteRows(fmt.Sprintf("SELECT COUNT(DISTINCT id) FROM node_access_paths WHERE %s", pathCondition), pathArgs...); err != nil {
		return impact, err
	}
	onboardingArgs := append([]any{nodeID}, pathArgs...)
	if impact.Delete.OnboardingTasks, err = s.countNodeDeleteRows(fmt.Sprintf("SELECT COUNT(DISTINCT id) FROM node_onboarding_tasks WHERE target_node_id = ? OR path_id IN (SELECT id FROM node_access_paths WHERE %s)", pathCondition), onboardingArgs...); err != nil {
		return impact, err
	}
	if impact.Delete.ChainProbeResults, err = s.countNodeDeleteChainProbeResults(nodeID, chainIDs); err != nil {
		return impact, err
	}
	if impact.Delete.RuntimeTransports, err = s.countNodeDeleteRows("SELECT COUNT(*) FROM node_transports WHERE node_id = ? OR parent_node_id = ?", nodeID, nodeID); err != nil {
		return impact, err
	}
	if impact.Delete.NodeLinks, err = s.countNodeDeleteRows("SELECT COUNT(*) FROM node_links WHERE source_node_id = ? OR target_node_id = ?", nodeID, nodeID); err != nil {
		return impact, err
	}
	if impact.Delete.PolicyAssignments, err = s.countNodeDeleteRows("SELECT COUNT(*) FROM node_policy_assignments WHERE node_id = ?", nodeID); err != nil {
		return impact, err
	}
	if impact.Delete.HealthSnapshots, err = s.countNodeDeleteRows("SELECT COUNT(*) FROM node_health_snapshots WHERE node_id = ?", nodeID); err != nil {
		return impact, err
	}
	if impact.Delete.SLAMinutes, err = s.countNodeDeleteRows("SELECT COUNT(*) FROM node_sla_minutes WHERE node_id = ?", nodeID); err != nil {
		return impact, err
	}
	if impact.Delete.APITokens, err = s.countNodeDeleteRows("SELECT COUNT(*) FROM node_api_tokens WHERE node_id = ?", nodeID); err != nil {
		return impact, err
	}
	if impact.Delete.TrustMaterials, err = s.countNodeDeleteRows("SELECT COUNT(*) FROM node_trust_materials WHERE node_id = ?", nodeID); err != nil {
		return impact, err
	}
	if impact.Delete.BootstrapTokens, err = s.countNodeDeleteRows("SELECT COUNT(*) FROM bootstrap_tokens WHERE target_id = ?", nodeID); err != nil {
		return impact, err
	}
	if impact.Delete.TenantBindings, err = s.countNodeDeleteTenantBindings(nodeID, chainIDs, pathCondition, pathArgs); err != nil {
		return impact, err
	}
	if impact.Update.ChildNodesDetached, err = s.countNodeDeleteRows("SELECT COUNT(*) FROM nodes WHERE parent_node_id = ?", nodeID); err != nil {
		return impact, err
	}
	return impact, nil
}

func (s *MySQLStore) DeleteNode(nodeID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	chainIDs, err := nodeChainIDs(tx, nodeID)
	if err != nil {
		return err
	}
	if err := deleteChainsForNodeDelete(tx, chainIDs); err != nil {
		return err
	}

	statements := []string{
		"DELETE FROM node_onboarding_tasks WHERE path_id IN (SELECT id FROM node_access_paths WHERE target_node_id = ? OR entry_node_id = ? OR JSON_CONTAINS(relay_node_ids_json, JSON_QUOTE(?)))",
		"DELETE FROM node_access_paths WHERE target_node_id = ? OR entry_node_id = ? OR JSON_CONTAINS(relay_node_ids_json, JSON_QUOTE(?))",
		"DELETE FROM chain_probe_results WHERE blocking_node_id = ?",
		"DELETE FROM node_transports WHERE node_id = ? OR parent_node_id = ?",
		"DELETE FROM node_links WHERE source_node_id = ? OR target_node_id = ?",
		"DELETE FROM node_onboarding_tasks WHERE target_node_id = ?",
		"DELETE FROM node_policy_assignments WHERE node_id = ?",
		"DELETE FROM node_health_snapshots WHERE node_id = ?",
		"DELETE FROM node_sla_minutes WHERE node_id = ?",
		"DELETE FROM node_api_tokens WHERE node_id = ?",
		"DELETE FROM node_trust_materials WHERE node_id = ?",
		"DELETE FROM bootstrap_tokens WHERE target_id = ?",
		"DELETE FROM tenant_nodes WHERE node_id = ?",
		"UPDATE nodes SET parent_node_id = NULL WHERE parent_node_id = ?",
	}
	for _, statement := range statements {
		switch strings.Count(statement, "?") {
		case 3:
			if _, err := tx.Exec(statement, nodeID, nodeID, nodeID); err != nil {
				return err
			}
		case 2:
			if _, err := tx.Exec(statement, nodeID, nodeID); err != nil {
				return err
			}
		default:
			if _, err := tx.Exec(statement, nodeID); err != nil {
				return err
			}
		}
	}
	if _, err := tx.Exec("DELETE FROM nodes WHERE id = ?", nodeID); err != nil {
		return err
	}
	return tx.Commit()
}

type nodeChainQuerier interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

func nodeChainIDs(db nodeChainQuerier, nodeID string) ([]string, error) {
	rows, err := db.Query("SELECT DISTINCT chain_id FROM chain_hops WHERE node_id = ? ORDER BY chain_id", nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	chainIDs := make([]string, 0)
	for rows.Next() {
		var chainID string
		if err := rows.Scan(&chainID); err != nil {
			return nil, err
		}
		chainIDs = append(chainIDs, chainID)
	}
	return chainIDs, rows.Err()
}

func deleteChainsForNodeDelete(tx *sql.Tx, chainIDs []string) error {
	if len(chainIDs) == 0 {
		return nil
	}
	placeholders := questionPlaceholders(len(chainIDs))
	statements := []string{
		fmt.Sprintf("DELETE FROM route_rules WHERE chain_id IN (%s)", placeholders),
		fmt.Sprintf("DELETE FROM node_onboarding_tasks WHERE path_id IN (SELECT id FROM node_access_paths WHERE chain_id IN (%s))", placeholders),
		fmt.Sprintf("DELETE FROM node_access_paths WHERE chain_id IN (%s)", placeholders),
		fmt.Sprintf("DELETE FROM chain_probe_results WHERE chain_id IN (%s)", placeholders),
		fmt.Sprintf("DELETE FROM tenant_chains WHERE chain_id IN (%s)", placeholders),
		fmt.Sprintf("DELETE FROM chain_hops WHERE chain_id IN (%s)", placeholders),
		fmt.Sprintf("DELETE FROM chains WHERE id IN (%s)", placeholders),
	}
	args := stringArgs(chainIDs)
	for _, statement := range statements {
		if _, err := tx.Exec(statement, args...); err != nil {
			return err
		}
	}
	return nil
}

func (s *MySQLStore) countNodeDeleteRows(query string, args ...any) (int, error) {
	var count int
	err := s.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

func (s *MySQLStore) countNodeDeleteRowsIn(queryFormat string, values []string) (int, error) {
	if len(values) == 0 {
		return 0, nil
	}
	return s.countNodeDeleteRows(fmt.Sprintf(queryFormat, questionPlaceholders(len(values))), stringArgs(values)...)
}

func (s *MySQLStore) countNodeDeleteChainProbeResults(nodeID string, chainIDs []string) (int, error) {
	if len(chainIDs) == 0 {
		return s.countNodeDeleteRows("SELECT COUNT(DISTINCT chain_id) FROM chain_probe_results WHERE blocking_node_id = ?", nodeID)
	}
	args := append(stringArgs(chainIDs), nodeID)
	return s.countNodeDeleteRows(fmt.Sprintf("SELECT COUNT(DISTINCT chain_id) FROM chain_probe_results WHERE chain_id IN (%s) OR blocking_node_id = ?", questionPlaceholders(len(chainIDs))), args...)
}

func (s *MySQLStore) countNodeDeleteTenantBindings(nodeID string, chainIDs []string, pathCondition string, pathArgs []any) (int, error) {
	total, err := s.countNodeDeleteRows("SELECT COUNT(*) FROM tenant_nodes WHERE node_id = ?", nodeID)
	if err != nil {
		return 0, err
	}
	count, err := s.countNodeDeleteRows("SELECT COUNT(*) FROM tenant_node_links WHERE node_link_id IN (SELECT id FROM node_links WHERE source_node_id = ? OR target_node_id = ?)", nodeID, nodeID)
	if err != nil {
		return 0, err
	}
	total += count
	count, err = s.countNodeDeleteRows(fmt.Sprintf("SELECT COUNT(*) FROM tenant_access_paths WHERE access_path_id IN (SELECT id FROM node_access_paths WHERE %s)", pathCondition), pathArgs...)
	if err != nil {
		return 0, err
	}
	total += count
	count, err = s.countNodeDeleteRowsIn("SELECT COUNT(*) FROM tenant_chains WHERE chain_id IN (%s)", chainIDs)
	if err != nil {
		return 0, err
	}
	total += count
	if len(chainIDs) == 0 {
		return total, nil
	}
	return total, nil
}

func nodeDeleteAccessPathCondition(nodeID string, chainIDs []string) (string, []any) {
	conditions := make([]string, 0, 4)
	args := make([]any, 0, len(chainIDs)+3)
	if len(chainIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf("chain_id IN (%s)", questionPlaceholders(len(chainIDs))))
		args = append(args, stringArgs(chainIDs)...)
	}
	conditions = append(conditions, "target_node_id = ?", "entry_node_id = ?", "JSON_CONTAINS(relay_node_ids_json, JSON_QUOTE(?))")
	args = append(args, nodeID, nodeID, nodeID)
	return strings.Join(conditions, " OR "), args
}

func questionPlaceholders(count int) string {
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}

func stringArgs(values []string) []any {
	args := make([]any, len(values))
	for index, value := range values {
		args[index] = value
	}
	return args
}

func (s *MySQLStore) NodeBindingPermission(tenantCtx domain.TenantAuthContext, nodeID string) (domain.BindingPermission, bool) {
	return s.tenantResourcePermission(tenantCtx, "tenant_nodes", "node_id", nodeID)
}

func (s *MySQLStore) CountNodeBindings(nodeID string) int {
	return s.countTenantResourceBindings("tenant_nodes", "node_id", nodeID)
}

func bindTenantResource(tx *sql.Tx, table string, idColumn string, tenantID string, resourceID string, createID string) error {
	now := nowRFC3339()
	_, err := tx.Exec(
		fmt.Sprintf(`INSERT INTO %s (tenant_id, %s, permission, create_id, created_at) VALUES (?, ?, ?, ?, ?)`, table, idColumn),
		tenantID, resourceID, domain.BindingPermissionManage, createID, now,
	)
	return err
}

func (s *MySQLStore) tenantResourcePermission(tenantCtx domain.TenantAuthContext, table string, idColumn string, resourceID string) (domain.BindingPermission, bool) {
	if tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.TenantID == "" {
		return domain.BindingPermissionManage, true
	}
	var permission domain.BindingPermission
	err := s.db.QueryRow(
		fmt.Sprintf(`SELECT permission FROM %s WHERE tenant_id = ? AND %s = ?`, table, idColumn),
		tenantCtx.ActiveTenant.TenantID, resourceID,
	).Scan(&permission)
	if err != nil || (permission != domain.BindingPermissionUse && permission != domain.BindingPermissionManage) {
		return "", false
	}
	return permission, true
}

func (s *MySQLStore) countTenantResourceBindings(table string, idColumn string, resourceID string) int {
	var count int
	err := s.db.QueryRow(
		fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s = ?`, table, idColumn),
		resourceID,
	).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}
