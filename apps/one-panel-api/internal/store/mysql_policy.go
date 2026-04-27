package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/policy"
)

func (s *MySQLStore) policyNodes() []domain.Node {
	all := s.ListNodes()
	items := make([]domain.Node, 0, len(all))
	for _, node := range all {
		if !node.Enabled || node.Status == domain.NodeStatusPending {
			continue
		}
		items = append(items, node)
	}
	return items
}

func (s *MySQLStore) buildGroupEntries() []policy.GroupScopeEntry {
	groupRows, err := s.db.Query("SELECT id, name FROM `groups` WHERE enabled = 1")
	if err != nil {
		return nil
	}
	defer groupRows.Close()
	type rawEntry struct {
		id    string
		entry policy.GroupScopeEntry
	}
	var rawEntries []rawEntry
	for groupRows.Next() {
		var id, name string
		if err := groupRows.Scan(&id, &name); err != nil {
			continue
		}
		rawEntries = append(rawEntries, rawEntry{id: id, entry: policy.GroupScopeEntry{GroupName: name}})
	}
	for i, re := range rawEntries {
		scopeRows, err := s.db.Query("SELECT scope_key FROM group_scopes WHERE group_id = ?", re.id)
		if err != nil {
			continue
		}
		var scopes []string
		for scopeRows.Next() {
			var sk string
			if err := scopeRows.Scan(&sk); err == nil {
				scopes = append(scopes, sk)
			}
		}
		scopeRows.Close()
		rawEntries[i].entry.ScopeKeys = scopes

		acctRows, err := s.db.Query("SELECT account_id FROM account_groups WHERE group_id = ?", re.id)
		if err != nil {
			continue
		}
		var accts []string
		for acctRows.Next() {
			var aid string
			if err := acctRows.Scan(&aid); err == nil {
				accts = append(accts, aid)
			}
		}
		acctRows.Close()
		rawEntries[i].entry.AccountIDs = accts
	}
	entries := make([]policy.GroupScopeEntry, 0, len(rawEntries))
	for _, re := range rawEntries {
		entries = append(entries, re.entry)
	}
	return entries
}

func (s *MySQLStore) compileLatestPolicyForNode(nodeID string) (string, string, bool) {
	exists, err := s.exists(context.Background(), "SELECT 1 FROM nodes WHERE id = ? AND enabled = 1 AND status != 'pending'", nodeID)
	if err != nil || !exists {
		return "", "", false
	}
	var revisionID string
	var version string
	err = s.db.QueryRow(
		`SELECT id, version FROM policy_revisions ORDER BY created_at DESC LIMIT 1`,
	).Scan(&revisionID, &version)
	if err != nil {
		return "", "", false
	}
	snapshotJSON, err := policy.CompileForNode(nodeID, s.policyNodes(), s.ListNodeLinks(), s.ListChains(), s.ListRouteRules(), s.buildGroupEntries())
	if err != nil {
		return "", "", false
	}
	_ = revisionID
	return version, snapshotJSON, true
}

func (s *MySQLStore) ListPolicyRevisions() []domain.PolicyRevision {
	rows, err := s.db.Query(
		`SELECT p.id, p.version, p.status, p.created_at, COUNT(a.node_id)
		 FROM policy_revisions p
		 LEFT JOIN node_policy_assignments a ON a.policy_revision_id = p.id
		 GROUP BY p.id, p.version, p.status, p.created_at
		 ORDER BY p.created_at DESC`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.PolicyRevision, 0)
	for rows.Next() {
		var item domain.PolicyRevision
		if err := rows.Scan(&item.ID, &item.Version, &item.Status, &item.CreatedAt, &item.AssignedNodes); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) PublishPolicy(accountID string) (domain.PolicyRevision, error) {
	nodes := s.policyNodes()
	links := s.ListNodeLinks()
	chains := s.ListChains()
	rules := s.ListRouteRules()
	groupEntries := s.buildGroupEntries()
	raw, err := policy.Compile(nodes, links, chains, rules, groupEntries)
	if err != nil {
		return domain.PolicyRevision{}, err
	}
	item := domain.PolicyRevision{
		ID:            newID("policy"),
		Version:       fmt.Sprintf("rev-%d", time.Now().Unix()),
		Status:        domain.PolicyStatusPublished,
		CreatedAt:     nowRFC3339(),
		AssignedNodes: len(nodes),
	}
	tx, err := s.db.Begin()
	if err != nil {
		return domain.PolicyRevision{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`INSERT INTO policy_revisions (id, version, payload_json, status, created_by_account_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		item.ID, item.Version, raw, item.Status, accountID, item.CreatedAt,
	); err != nil {
		return domain.PolicyRevision{}, err
	}
	if _, err := tx.Exec("DELETE FROM node_policy_assignments"); err != nil {
		return domain.PolicyRevision{}, err
	}
	for _, node := range nodes {
		snapshotJSON, err := policy.CompileForNode(node.ID, nodes, links, chains, rules, groupEntries)
		if err != nil {
			return domain.PolicyRevision{}, err
		}
		if _, err := tx.Exec(
			`INSERT INTO node_policy_assignments (node_id, policy_revision_id, snapshot_json, assigned_at) VALUES (?, ?, ?, ?)`,
			node.ID, item.ID, snapshotJSON, item.CreatedAt,
		); err != nil {
			return domain.PolicyRevision{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.PolicyRevision{}, err
	}
	return item, nil
}

func (s *MySQLStore) GetNodeAgentPolicy(nodeID string) (domain.NodeAgentPolicy, bool) {
	var (
		policyID string
		payload  string
		version  string
	)
	err := s.db.QueryRow(
		`SELECT p.id, p.version, a.snapshot_json
		 FROM node_policy_assignments a
		 JOIN policy_revisions p ON p.id = a.policy_revision_id
		WHERE a.node_id = ?
		 ORDER BY a.assigned_at DESC
		 LIMIT 1`,
		nodeID,
	).Scan(&policyID, &version, &payload)
	if err == nil {
		return domain.NodeAgentPolicy{
			NodeID:           nodeID,
			PolicyRevisionID: version,
			PayloadJSON:      payload,
		}, true
	}
	if err != sql.ErrNoRows {
		return domain.NodeAgentPolicy{}, false
	}
	latestPolicyVersion, snapshotJSON, ok := s.compileLatestPolicyForNode(nodeID)
	if !ok {
		return domain.NodeAgentPolicy{}, false
	}
	_, _ = s.db.Exec(
		`INSERT INTO node_policy_assignments (node_id, policy_revision_id, snapshot_json, assigned_at)
		 SELECT ?, id, ?, ? FROM policy_revisions ORDER BY created_at DESC LIMIT 1`,
		nodeID, snapshotJSON, nowRFC3339(),
	)
	return domain.NodeAgentPolicy{
		NodeID:           nodeID,
		PolicyRevisionID: latestPolicyVersion,
		PayloadJSON:      snapshotJSON,
	}, true
}
