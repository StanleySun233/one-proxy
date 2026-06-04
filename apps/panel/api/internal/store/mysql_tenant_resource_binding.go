package store

import (
	"database/sql"
	"fmt"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

type tenantResourceTable struct {
	table    string
	idColumn string
}

func tenantResourceTableFor(resourceType domain.ResourceType) (tenantResourceTable, bool) {
	switch resourceType {
	case domain.ResourceTypeNode:
		return tenantResourceTable{table: "tenant_nodes", idColumn: "node_id"}, true
	case domain.ResourceTypeNodeLink:
		return tenantResourceTable{table: "tenant_node_links", idColumn: "node_link_id"}, true
	case domain.ResourceTypeScope:
		return tenantResourceTable{table: "tenant_scopes", idColumn: "scope_id"}, true
	case domain.ResourceTypeChain:
		return tenantResourceTable{table: "tenant_chains", idColumn: "chain_id"}, true
	case domain.ResourceTypeRouteRule:
		return tenantResourceTable{table: "tenant_route_rules", idColumn: "route_rule_id"}, true
	case domain.ResourceTypeAccessPath:
		return tenantResourceTable{table: "tenant_access_paths", idColumn: "access_path_id"}, true
	default:
		return tenantResourceTable{}, false
	}
}

func (s *MySQLStore) ListTenantResourceBindings(resourceType domain.ResourceType, resourceID string) ([]domain.TenantResourceBinding, error) {
	spec, ok := tenantResourceTableFor(resourceType)
	if !ok || resourceID == "" {
		return nil, sql.ErrNoRows
	}
	rows, err := s.db.Query(
		fmt.Sprintf(`SELECT tb.tenant_id, t.name, tb.%s, tb.permission, tb.create_id, tb.created_at
		 FROM %s tb
		 JOIN tenants t ON t.id = tb.tenant_id
		 WHERE tb.%s = ? AND tb.permission IN (?, ?)
		 ORDER BY t.name`, spec.idColumn, spec.table, spec.idColumn),
		resourceID, domain.BindingPermissionUse, domain.BindingPermissionManage,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.TenantResourceBinding, 0)
	for rows.Next() {
		var item domain.TenantResourceBinding
		if err := rows.Scan(&item.TenantID, &item.TenantName, &item.ResourceID, &item.Permission, &item.CreateID, &item.CreatedAt); err != nil {
			continue
		}
		item.ResourceType = string(resourceType)
		items = append(items, item)
	}
	return items, nil
}

func (s *MySQLStore) UpsertTenantResourceBinding(resourceType domain.ResourceType, resourceID string, tenantID string, permission domain.BindingPermission, createID string) (domain.TenantResourceBinding, error) {
	spec, ok := tenantResourceTableFor(resourceType)
	if !ok || resourceID == "" || tenantID == "" {
		return domain.TenantResourceBinding{}, sql.ErrNoRows
	}
	now := nowRFC3339()
	_, err := s.db.Exec(
		fmt.Sprintf(`INSERT INTO %s (tenant_id, %s, permission, create_id, created_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE permission = VALUES(permission)`, spec.table, spec.idColumn),
		tenantID, resourceID, permission, createID, now,
	)
	if err != nil {
		return domain.TenantResourceBinding{}, err
	}
	return s.tenantResourceBinding(resourceType, resourceID, tenantID)
}

func (s *MySQLStore) DeleteTenantResourceBinding(resourceType domain.ResourceType, resourceID string, tenantID string) error {
	spec, ok := tenantResourceTableFor(resourceType)
	if !ok || resourceID == "" || tenantID == "" {
		return sql.ErrNoRows
	}
	result, err := s.db.Exec(
		fmt.Sprintf("DELETE FROM %s WHERE tenant_id = ? AND %s = ?", spec.table, spec.idColumn),
		tenantID, resourceID,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *MySQLStore) TenantResourceBindingPermission(tenantCtx domain.TenantAuthContext, resourceType domain.ResourceType, resourceID string) (domain.BindingPermission, bool) {
	spec, ok := tenantResourceTableFor(resourceType)
	if !ok {
		return "", false
	}
	return s.tenantResourcePermission(tenantCtx, spec.table, spec.idColumn, resourceID)
}

func (s *MySQLStore) CountTenantResourceManageBindings(resourceType domain.ResourceType, resourceID string) int {
	spec, ok := tenantResourceTableFor(resourceType)
	if !ok || resourceID == "" {
		return 0
	}
	var count int
	err := s.db.QueryRow(
		fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ? AND permission = ?", spec.table, spec.idColumn),
		resourceID, domain.BindingPermissionManage,
	).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

func (s *MySQLStore) tenantResourceBinding(resourceType domain.ResourceType, resourceID string, tenantID string) (domain.TenantResourceBinding, error) {
	spec, ok := tenantResourceTableFor(resourceType)
	if !ok {
		return domain.TenantResourceBinding{}, sql.ErrNoRows
	}
	var item domain.TenantResourceBinding
	err := s.db.QueryRow(
		fmt.Sprintf(`SELECT tb.tenant_id, t.name, tb.%s, tb.permission, tb.create_id, tb.created_at
		 FROM %s tb
		 JOIN tenants t ON t.id = tb.tenant_id
		 WHERE tb.tenant_id = ? AND tb.%s = ?`, spec.idColumn, spec.table, spec.idColumn),
		tenantID, resourceID,
	).Scan(&item.TenantID, &item.TenantName, &item.ResourceID, &item.Permission, &item.CreateID, &item.CreatedAt)
	if err != nil {
		return domain.TenantResourceBinding{}, err
	}
	item.ResourceType = string(resourceType)
	return item, nil
}
