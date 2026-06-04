package store

import (
	"encoding/json"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	proxy "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/domain"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/policy"
)

func tenantPolicyContext(tenantCtx domain.TenantAuthContext) domain.TenantAuthContext {
	tenantCtx.SuperAdmin = false
	return tenantCtx
}

func (s *MySQLStore) tenantPolicyInputs(tenantCtx domain.TenantAuthContext) ([]domain.Node, []domain.NodeLink, []proxy.Chain, []proxy.RouteRule) {
	scoped := tenantPolicyContext(tenantCtx)
	return s.policyNodesForTenant(scoped), s.ListNodeLinksForTenant(scoped), s.ListChainsForTenant(scoped), s.ListRouteRulesForTenant(scoped)
}

func (s *MySQLStore) policyNodesForTenant(tenantCtx domain.TenantAuthContext) []domain.Node {
	all := s.ListNodesForTenant(tenantCtx)
	items := make([]domain.Node, 0, len(all))
	for _, node := range all {
		if !node.Enabled || node.Status == domain.NodeStatusPending {
			continue
		}
		items = append(items, node)
	}
	return items
}

func (s *MySQLStore) ListPolicyRevisions() []domain.PolicyRevision {
	rows, err := s.db.Query(
		`SELECT p.id, p.tenant_id, p.version, p.status, p.created_at, COUNT(a.node_id)
		 FROM policy_revisions p
		 LEFT JOIN node_policy_assignments a ON a.policy_revision_id = p.id AND a.tenant_id = p.tenant_id
		 GROUP BY p.id, p.tenant_id, p.version, p.status, p.created_at
		 ORDER BY p.created_at DESC`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.PolicyRevision, 0)
	for rows.Next() {
		var item domain.PolicyRevision
		if err := rows.Scan(&item.ID, &item.TenantID, &item.Version, &item.Status, &item.CreatedAt, &item.AssignedNodes); err != nil {
			continue
		}
		item.AffectedTenantIDs = []string{item.TenantID}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) ListPolicyRevisionsForTenant(tenantCtx domain.TenantAuthContext) []domain.PolicyRevision {
	scoped := tenantPolicyContext(tenantCtx)
	rows, err := s.db.Query(
		`SELECT p.id, p.tenant_id, p.version, p.status, p.created_at, COUNT(a.node_id)
		 FROM policy_revisions p
		 LEFT JOIN node_policy_assignments a ON a.policy_revision_id = p.id AND a.tenant_id = p.tenant_id
		 WHERE p.tenant_id = ?
		 GROUP BY p.id, p.tenant_id, p.version, p.status, p.created_at
		 ORDER BY p.created_at DESC`,
		scoped.ActiveTenant.TenantID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.PolicyRevision, 0)
	for rows.Next() {
		var item domain.PolicyRevision
		if err := rows.Scan(&item.ID, &item.TenantID, &item.Version, &item.Status, &item.CreatedAt, &item.AssignedNodes); err != nil {
			continue
		}
		item.AffectedTenantIDs = []string{item.TenantID}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) PublishPolicy(tenantCtx domain.TenantAuthContext, accountID string) (domain.PolicyRevision, error) {
	nodes, links, chains, rules := s.tenantPolicyInputs(tenantCtx)
	raw, err := policy.CompileForTenant(tenantCtx.ActiveTenant.TenantID, nodes, links, chains, rules, nil)
	if err != nil {
		return domain.PolicyRevision{}, err
	}
	policyID, err := s.nextID("policy_revision")
	if err != nil {
		return domain.PolicyRevision{}, err
	}
	item := domain.PolicyRevision{
		ID:                policyID,
		TenantID:          tenantCtx.ActiveTenant.TenantID,
		Version:           policyID,
		Status:            domain.PolicyStatusPublished,
		CreatedAt:         nowRFC3339(),
		AssignedNodes:     len(nodes),
		AffectedTenantIDs: []string{tenantCtx.ActiveTenant.TenantID},
	}
	tx, err := s.db.Begin()
	if err != nil {
		return domain.PolicyRevision{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`INSERT INTO policy_revisions (id, tenant_id, version, payload_json, status, created_by_account_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.TenantID, item.Version, raw, item.Status, accountID, item.CreatedAt,
	); err != nil {
		return domain.PolicyRevision{}, err
	}
	for _, node := range nodes {
		if _, err := tx.Exec("DELETE FROM node_policy_assignments WHERE tenant_id = ? AND node_id = ?", item.TenantID, node.ID); err != nil {
			return domain.PolicyRevision{}, err
		}
		snapshotJSON, err := policy.CompileForNode(node.ID, nodes, links, chains, rules, nil)
		if err != nil {
			return domain.PolicyRevision{}, err
		}
		wrapped, err := tenantNodePolicyPayload(tenantCtx.ActiveTenant.TenantID, item.Version, snapshotJSON)
		if err != nil {
			return domain.PolicyRevision{}, err
		}
		if _, err := tx.Exec(
			`INSERT INTO node_policy_assignments (tenant_id, node_id, policy_revision_id, snapshot_json, assigned_at) VALUES (?, ?, ?, ?, ?)`,
			item.TenantID, node.ID, item.ID, wrapped, item.CreatedAt,
		); err != nil {
			return domain.PolicyRevision{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.PolicyRevision{}, err
	}
	return item, nil
}

type tenantNodePolicySnapshot struct {
	TenantID         string          `json:"tenantId"`
	PolicyRevisionID string          `json:"policyRevisionId"`
	Payload          json.RawMessage `json:"payload"`
}

type tenantNodePolicyPayloadResult struct {
	Snapshots []tenantNodePolicySnapshot `json:"snapshots"`
}

func tenantNodePolicyPayload(tenantID string, policyRevisionID string, snapshotJSON string) (string, error) {
	payload, err := json.Marshal(tenantNodePolicyPayloadResult{
		Snapshots: []tenantNodePolicySnapshot{
			{
				TenantID:         tenantID,
				PolicyRevisionID: policyRevisionID,
				Payload:          json.RawMessage(snapshotJSON),
			},
		},
	})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func (s *MySQLStore) nodePolicyTenantContexts(nodeID string) []domain.TenantAuthContext {
	rows, err := s.db.Query(
		`SELECT t.id, t.name, tn.permission, t.created_at
		 FROM tenant_nodes tn
		 JOIN tenants t ON t.id = tn.tenant_id
		 WHERE tn.node_id = ? AND tn.permission IN (?, ?)
		 ORDER BY t.id`,
		nodeID, domain.BindingPermissionUse, domain.BindingPermissionManage,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.TenantAuthContext, 0)
	for rows.Next() {
		var tenantID string
		var tenantName string
		var permission domain.BindingPermission
		var joinedAt string
		if err := rows.Scan(&tenantID, &tenantName, &permission, &joinedAt); err != nil {
			continue
		}
		items = append(items, domain.TenantAuthContext{
			ActiveTenant: domain.TenantMembership{
				TenantID:   tenantID,
				TenantName: tenantName,
				Role:       domain.TenantRoleAdmin,
				JoinedAt:   joinedAt,
			},
		})
	}
	return items
}

func (s *MySQLStore) GetNodeAgentPolicy(nodeID string) (domain.NodeAgentPolicy, bool) {
	rows, err := s.db.Query(
		`SELECT a.tenant_id, p.version, a.snapshot_json
		 FROM node_policy_assignments a
		 JOIN policy_revisions p ON p.id = a.policy_revision_id AND p.tenant_id = a.tenant_id
		 JOIN tenant_nodes tn ON tn.tenant_id = a.tenant_id AND tn.node_id = a.node_id
		 WHERE a.node_id = ? AND tn.permission IN (?, ?)
		 ORDER BY a.assigned_at DESC, p.created_at DESC`,
		nodeID, domain.BindingPermissionUse, domain.BindingPermissionManage,
	)
	if err != nil {
		return domain.NodeAgentPolicy{}, false
	}
	defer rows.Close()
	snapshots := make([]tenantNodePolicySnapshot, 0)
	version := ""
	for rows.Next() {
		var tenantID string
		var revisionVersion string
		var snapshotJSON string
		if err := rows.Scan(&tenantID, &revisionVersion, &snapshotJSON); err != nil {
			continue
		}
		var saved tenantNodePolicyPayloadResult
		if err := json.Unmarshal([]byte(snapshotJSON), &saved); err != nil {
			continue
		}
		for _, snapshot := range saved.Snapshots {
			if snapshot.TenantID != tenantID {
				continue
			}
			snapshot.PolicyRevisionID = revisionVersion
			snapshots = append(snapshots, snapshot)
			if version == "" {
				version = revisionVersion
			}
		}
	}
	if len(snapshots) == 0 {
		return domain.NodeAgentPolicy{}, false
	}
	payload, err := json.Marshal(tenantNodePolicyPayloadResult{Snapshots: snapshots})
	if err != nil {
		return domain.NodeAgentPolicy{}, false
	}
	return domain.NodeAgentPolicy{
		NodeID:           nodeID,
		PolicyRevisionID: version,
		PayloadJSON:      string(payload),
	}, true
}
