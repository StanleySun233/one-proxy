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
	nodeID, err := s.nextNodeID()
	if err != nil {
		return domain.Node{}, err
	}
	item := domain.Node{
		ID:           nodeID,
		CreateID:     "1",
		OwnerID:      "1",
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

func (s *MySQLStore) DeleteNode(nodeID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	statements := []string{
		"DELETE FROM chain_hops WHERE node_id = ?",
		"DELETE FROM node_links WHERE source_node_id = ? OR target_node_id = ?",
		"DELETE FROM node_onboarding_tasks WHERE target_node_id = ?",
		"DELETE FROM node_access_paths WHERE target_node_id = ? OR entry_node_id = ?",
		"DELETE FROM node_policy_assignments WHERE node_id = ?",
		"DELETE FROM node_health_snapshots WHERE node_id = ?",
		"DELETE FROM node_api_tokens WHERE node_id = ?",
		"DELETE FROM node_trust_materials WHERE node_id = ?",
		"UPDATE nodes SET parent_node_id = NULL WHERE parent_node_id = ?",
	}
	for _, statement := range statements {
		if strings.Count(statement, "?") == 2 {
			if _, err := tx.Exec(statement, nodeID, nodeID); err != nil {
				return err
			}
			continue
		}
		if _, err := tx.Exec(statement, nodeID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec("DELETE FROM nodes WHERE id = ?", nodeID); err != nil {
		return err
	}
	return tx.Commit()
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
