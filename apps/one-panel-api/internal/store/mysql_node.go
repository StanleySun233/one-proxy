package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/auth"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/policy"
)

func (s *MySQLStore) ListNodes() []domain.Node {
	rows, err := s.db.Query(
		`SELECT id, name, mode, scope_key, COALESCE(parent_node_id, ''), enabled, status, COALESCE(public_host, ''), COALESCE(public_port, 0)
		 FROM nodes ORDER BY name`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.Node, 0)
	for rows.Next() {
		var item domain.Node
		var enabled int
		if err := rows.Scan(&item.ID, &item.Name, &item.Mode, &item.ScopeKey, &item.ParentNodeID, &enabled, &item.Status, &item.PublicHost, &item.PublicPort); err != nil {
			continue
		}
		item.Enabled = enabled == 1
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) ListNodeTransports() []domain.NodeTransport {
	rows, err := s.db.Query(
		`SELECT id, node_id, transport_type, direction, address, status, COALESCE(parent_node_id, ''), COALESCE(connected_at, ''), COALESCE(last_heartbeat_at, ''), latency_ms, details_json
		 FROM node_transports ORDER BY node_id, transport_type, address`,
	)
	if err != nil {
		return s.syntheticPublicTransports(nil)
	}
	defer rows.Close()
	items := make([]domain.NodeTransport, 0)
	for rows.Next() {
		var item domain.NodeTransport
		var detailsJSON string
		if err := rows.Scan(&item.ID, &item.NodeID, &item.TransportType, &item.Direction, &item.Address, &item.Status, &item.ParentNodeID, &item.ConnectedAt, &item.LastHeartbeatAt, &item.LatencyMs, &detailsJSON); err != nil {
			continue
		}
		item.Details = decodeJSONMap(detailsJSON)
		items = append(items, item)
	}
	return s.syntheticPublicTransports(items)
}

func (s *MySQLStore) syntheticPublicTransports(items []domain.NodeTransport) []domain.NodeTransport {
	nodes := s.ListNodes()
	healthByNodeID := make(map[string]domain.NodeHealth)
	for _, health := range s.ListNodeHealth() {
		healthByNodeID[health.NodeID] = health
	}
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		seen[item.NodeID+"|"+item.TransportType+"|"+item.Address] = struct{}{}
	}
	for _, node := range nodes {
		if node.PublicHost == "" || node.PublicPort <= 0 {
			continue
		}
		address := fmt.Sprintf("http://%s:%d", node.PublicHost, node.PublicPort)
		key := node.ID + "|public_http|" + address
		if _, ok := seen[key]; ok {
			continue
		}
		status := domain.TransportStatusAvailable
		lastHeartbeat := ""
		if health, ok := healthByNodeID[node.ID]; ok {
			lastHeartbeat = health.HeartbeatAt
			if node.Status == domain.NodeStatusHealthy {
				status = domain.TransportStatusConnected
			} else {
				status = node.Status
			}
		}
		items = append(items, domain.NodeTransport{
			ID:              "derived-public-" + node.ID,
			NodeID:          node.ID,
			TransportType:   domain.TransportTypePublicHTTP,
			Direction:       "inbound",
			Address:         address,
			Status:          status,
			ConnectedAt:     lastHeartbeat,
			LastHeartbeatAt: lastHeartbeat,
			LatencyMs:       0,
			Details:         map[string]string{"source": "derived_public_endpoint"},
		})
	}
	return items
}

func (s *MySQLStore) UpsertNodeTransport(input domain.UpsertNodeTransportInput) (domain.NodeTransport, error) {
	id := newID("transport")
	now := nowRFC3339()
	detailsJSON := encodeJSONMap(input.Details)
	existingID := ""
	_ = s.db.QueryRow(
		`SELECT id FROM node_transports WHERE node_id = ? AND transport_type = ? AND address = ? LIMIT 1`,
		input.NodeID, input.TransportType, input.Address,
	).Scan(&existingID)
	if existingID != "" {
		id = existingID
	}
	_, err := s.db.Exec(
		`INSERT INTO node_transports (id, node_id, transport_type, direction, address, status, parent_node_id, connected_at, last_heartbeat_at, latency_ms, details_json, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		   direction = VALUES(direction),
		   status = VALUES(status),
		   parent_node_id = VALUES(parent_node_id),
		   connected_at = VALUES(connected_at),
		   last_heartbeat_at = VALUES(last_heartbeat_at),
		   latency_ms = VALUES(latency_ms),
		   details_json = VALUES(details_json),
		   updated_at = VALUES(updated_at)`,
		id, input.NodeID, input.TransportType, input.Direction, input.Address, input.Status, input.ParentNodeID, input.ConnectedAt, input.LastHeartbeatAt, input.LatencyMs, detailsJSON, now, now,
	)
	if err != nil {
		return domain.NodeTransport{}, err
	}
	return domain.NodeTransport{
		ID:              id,
		NodeID:          input.NodeID,
		TransportType:   input.TransportType,
		Direction:       input.Direction,
		Address:         input.Address,
		Status:          input.Status,
		ParentNodeID:    input.ParentNodeID,
		ConnectedAt:     input.ConnectedAt,
		LastHeartbeatAt: input.LastHeartbeatAt,
		LatencyMs:       input.LatencyMs,
		Details:         input.Details,
	}, nil
}

func (s *MySQLStore) CreateNode(input domain.CreateNodeInput) (domain.Node, error) {
	nodeID, err := s.nextNodeID()
	if err != nil {
		return domain.Node{}, err
	}
	item := domain.Node{
		ID:           nodeID,
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
		`INSERT INTO nodes (id, name, mode, public_host, public_port, scope_key, parent_node_id, enabled, status, created_at, updated_at)
		 VALUES (?, ?, ?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), ?, ?, ?, ?)`,
		item.ID, item.Name, item.Mode, item.PublicHost, item.PublicPort, item.ScopeKey, item.ParentNodeID, 1, item.Status, now, now,
	)
	return item, err
}

func (s *MySQLStore) ProvisionNodeAccess(nodeID string) (domain.ApproveNodeEnrollmentResult, error) {
	var (
		node    domain.Node
		enabled int
	)
	err := s.db.QueryRow(
		`SELECT id, name, mode, scope_key, COALESCE(parent_node_id, ''), enabled, status, COALESCE(public_host, ''), COALESCE(public_port, 0)
		 FROM nodes WHERE id = ?`,
		nodeID,
	).Scan(&node.ID, &node.Name, &node.Mode, &node.ScopeKey, &node.ParentNodeID, &enabled, &node.Status, &node.PublicHost, &node.PublicPort)
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	node.Enabled = enabled == 1
	trustMaterial, err := auth.RandomToken()
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	accessToken, err := auth.RandomToken()
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	now := nowRFC3339()
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM node_api_tokens WHERE node_id = ?", nodeID); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if _, err := tx.Exec(
		`UPDATE node_trust_materials SET status = ?, updated_at = ? WHERE node_id = ? AND status = 'active'`,
		domain.TrustMaterialStatusRotated, now, nodeID,
	); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if _, err := tx.Exec("UPDATE nodes SET status = ?, updated_at = ? WHERE id = ?", domain.NodeStatusHealthy, now, nodeID); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if _, err := tx.Exec(
		`INSERT INTO node_trust_materials (id, node_id, material_type, material_value, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		newID("trust"), nodeID, "shared_secret", trustMaterial, domain.TrustMaterialStatusActive, now, now,
	); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if _, err := tx.Exec(
		`INSERT INTO node_api_tokens (id, node_id, token_hash, expires_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		newID("node-token"), nodeID, accessToken, expiresAt, now, now,
	); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if err := s.assignLatestPolicyTx(tx, nodeID, now); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	node.Status = domain.NodeStatusHealthy
	return domain.ApproveNodeEnrollmentResult{
		Node:          node,
		AccessToken:   accessToken,
		TrustMaterial: trustMaterial,
		ExpiresAt:     expiresAt,
	}, nil
}

func (s *MySQLStore) assignLatestPolicyTx(tx *sql.Tx, nodeID string, assignedAt string) error {
	var latestRevisionID string
	err := tx.QueryRow(
		`SELECT id FROM policy_revisions ORDER BY created_at DESC LIMIT 1`,
	).Scan(&latestRevisionID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	snapshotJSON, err := policy.CompileForNode(nodeID, s.policyNodes(), s.ListNodeLinks(), s.ListChains(), s.ListRouteRules(), s.buildGroupEntries())
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM node_policy_assignments WHERE node_id = ?`, nodeID); err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO node_policy_assignments (node_id, policy_revision_id, snapshot_json, assigned_at) VALUES (?, ?, ?, ?)`,
		nodeID, latestRevisionID, snapshotJSON, assignedAt,
	)
	return err
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

func (s *MySQLStore) ListNodeLinks() []domain.NodeLink {
	rows, err := s.db.Query(
		`SELECT id, source_node_id, target_node_id, link_type, trust_state FROM node_links ORDER BY source_node_id, target_node_id`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.NodeLink, 0)
	for rows.Next() {
		var item domain.NodeLink
		if err := rows.Scan(&item.ID, &item.SourceNodeID, &item.TargetNodeID, &item.LinkType, &item.TrustState); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) CreateNodeLink(input domain.CreateNodeLinkInput) (domain.NodeLink, error) {
	item := domain.NodeLink{
		ID:           newID("link"),
		SourceNodeID: input.SourceNodeID,
		TargetNodeID: input.TargetNodeID,
		LinkType:     input.LinkType,
		TrustState:   input.TrustState,
	}
	now := nowRFC3339()
	_, err := s.db.Exec(
		`INSERT INTO node_links (id, source_node_id, target_node_id, link_type, trust_state, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.SourceNodeID, item.TargetNodeID, item.LinkType, item.TrustState, now, now,
	)
	return item, err
}

func (s *MySQLStore) ListNodeAccessPaths() []domain.NodeAccessPath {
	rows, err := s.db.Query(
		`SELECT id, name, mode, COALESCE(target_node_id, ''), COALESCE(entry_node_id, ''), relay_node_ids_json, COALESCE(target_host, ''), COALESCE(target_port, 0), enabled
		 FROM node_access_paths
		 ORDER BY name`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.NodeAccessPath, 0)
	for rows.Next() {
		var item domain.NodeAccessPath
		var relayJSON string
		var enabled int
		if err := rows.Scan(&item.ID, &item.Name, &item.Mode, &item.TargetNodeID, &item.EntryNodeID, &relayJSON, &item.TargetHost, &item.TargetPort, &enabled); err != nil {
			continue
		}
		item.RelayNodeIDs = decodeJSONStringSlice(relayJSON)
		item.Enabled = enabled == 1
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) CreateNodeAccessPath(input domain.CreateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	item := domain.NodeAccessPath{
		ID:           newID("path"),
		Name:         input.Name,
		Mode:         input.Mode,
		TargetNodeID: input.TargetNodeID,
		EntryNodeID:  input.EntryNodeID,
		RelayNodeIDs: normalizeStringSlice(input.RelayNodeIDs),
		TargetHost:   input.TargetHost,
		TargetPort:   input.TargetPort,
		Enabled:      true,
	}
	now := nowRFC3339()
	_, err := s.db.Exec(
		`INSERT INTO node_access_paths (id, name, mode, target_node_id, entry_node_id, relay_node_ids_json, target_host, target_port, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), ?, ?, ?, ?)`,
		item.ID, item.Name, item.Mode, item.TargetNodeID, item.EntryNodeID, encodeJSONStringSlice(item.RelayNodeIDs), item.TargetHost, item.TargetPort, 1, now, now,
	)
	return item, err
}

func (s *MySQLStore) UpdateNodeAccessPath(pathID string, input domain.UpdateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	now := nowRFC3339()
	_, err := s.db.Exec(
		`UPDATE node_access_paths
		 SET name = ?, mode = ?, target_node_id = NULLIF(?, ''), entry_node_id = NULLIF(?, ''), relay_node_ids_json = ?, target_host = NULLIF(?, ''), target_port = ?, enabled = ?, updated_at = ?
		 WHERE id = ?`,
		input.Name, input.Mode, input.TargetNodeID, input.EntryNodeID, encodeJSONStringSlice(input.RelayNodeIDs), input.TargetHost, input.TargetPort, boolToInt(input.Enabled), now, pathID,
	)
	if err != nil {
		return domain.NodeAccessPath{}, err
	}
	for _, item := range s.ListNodeAccessPaths() {
		if item.ID == pathID {
			return item, nil
		}
	}
	return domain.NodeAccessPath{}, sql.ErrNoRows
}

func (s *MySQLStore) DeleteNodeAccessPath(pathID string) error {
	_, err := s.db.Exec("DELETE FROM node_access_paths WHERE id = ?", pathID)
	return err
}

func (s *MySQLStore) ListNodeOnboardingTasks() []domain.NodeOnboardingTask {
	rows, err := s.db.Query(
		`SELECT id, mode, COALESCE(path_id, ''), COALESCE(target_node_id, ''), COALESCE(target_host, ''), COALESCE(target_port, 0), status, status_message, requested_by_account_id, created_at, updated_at
		 FROM node_onboarding_tasks
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.NodeOnboardingTask, 0)
	for rows.Next() {
		var item domain.NodeOnboardingTask
		if err := rows.Scan(&item.ID, &item.Mode, &item.PathID, &item.TargetNodeID, &item.TargetHost, &item.TargetPort, &item.Status, &item.StatusMessage, &item.RequestedByAccountID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) CreateNodeOnboardingTask(accountID string, input domain.CreateNodeOnboardingTaskInput) (domain.NodeOnboardingTask, error) {
	now := nowRFC3339()
	item := domain.NodeOnboardingTask{
		ID:                   newID("task"),
		Mode:                 input.Mode,
		PathID:               input.PathID,
		TargetNodeID:         input.TargetNodeID,
		TargetHost:           input.TargetHost,
		TargetPort:           input.TargetPort,
		Status:               domain.TaskStatusPlanned,
		StatusMessage:        "task created",
		RequestedByAccountID: accountID,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	_, err := s.db.Exec(
		`INSERT INTO node_onboarding_tasks (id, mode, path_id, target_node_id, target_host, target_port, status, status_message, requested_by_account_id, created_at, updated_at)
		 VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Mode, item.PathID, item.TargetNodeID, item.TargetHost, item.TargetPort, item.Status, item.StatusMessage, item.RequestedByAccountID, item.CreatedAt, item.UpdatedAt,
	)
	return item, err
}

func (s *MySQLStore) UpdateNodeOnboardingTaskStatus(taskID string, status string, statusMessage string) (domain.NodeOnboardingTask, error) {
	now := nowRFC3339()
	_, err := s.db.Exec(
		`UPDATE node_onboarding_tasks SET status = ?, status_message = ?, updated_at = ? WHERE id = ?`,
		status, statusMessage, now, taskID,
	)
	if err != nil {
		return domain.NodeOnboardingTask{}, err
	}
	for _, item := range s.ListNodeOnboardingTasks() {
		if item.ID == taskID {
			return item, nil
		}
	}
	return domain.NodeOnboardingTask{}, sql.ErrNoRows
}

func (s *MySQLStore) CreateBootstrapToken(input domain.CreateBootstrapTokenInput) (domain.BootstrapToken, error) {
	token, err := auth.RandomToken()
	if err != nil {
		return domain.BootstrapToken{}, err
	}
	item := domain.BootstrapToken{
		ID:         newID("bootstrap"),
		Token:      token,
		TargetType: input.TargetType,
		TargetID:   input.TargetID,
		NodeName:   input.NodeName,
		ExpiresAt:  time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339),
		CreatedAt:  nowRFC3339(),
	}
	_, err = s.db.Exec(
		`INSERT INTO bootstrap_tokens (id, token_hash, target_type, target_id, node_name, expires_at, consumed_at, created_at)
		 VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, NULL, ?)`,
		item.ID, token, item.TargetType, item.TargetID, item.NodeName, item.ExpiresAt, nowRFC3339(),
	)
	return item, err
}

func (s *MySQLStore) ListUnconsumedBootstrapTokens() []domain.BootstrapToken {
	rows, err := s.db.Query(
		`SELECT id, target_type, COALESCE(target_id, ''), COALESCE(node_name, ''), expires_at, created_at
		 FROM bootstrap_tokens
		 WHERE consumed_at IS NULL AND expires_at > ?
		 ORDER BY created_at DESC`,
		nowRFC3339(),
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.BootstrapToken, 0)
	for rows.Next() {
		var item domain.BootstrapToken
		if err := rows.Scan(&item.ID, &item.TargetType, &item.TargetID, &item.NodeName, &item.ExpiresAt, &item.CreatedAt); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) EnrollNode(input domain.EnrollNodeInput) (domain.EnrollNodeResult, error) {
	var (
		tokenID    string
		targetID   sql.NullString
		nodeName   sql.NullString
		expiresAt  string
		consumedAt sql.NullString
	)
	err := s.db.QueryRow(
		`SELECT id, target_id, node_name, expires_at, consumed_at FROM bootstrap_tokens WHERE token_hash = ?`,
		input.Token,
	).Scan(&tokenID, &targetID, &nodeName, &expiresAt, &consumedAt)
	if err != nil {
		return domain.EnrollNodeResult{}, err
	}
	expiry, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().UTC().After(expiry) || consumedAt.Valid {
		return domain.EnrollNodeResult{}, fmt.Errorf("invalid bootstrap token")
	}
	effectiveName := input.Name
	if nodeName.Valid && nodeName.String != "" {
		effectiveName = nodeName.String
	}
	now := nowRFC3339()
	enrollmentSecret, err := auth.RandomToken()
	if err != nil {
		return domain.EnrollNodeResult{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return domain.EnrollNodeResult{}, err
	}
	defer tx.Rollback()
	var node domain.Node
	if targetID.Valid && targetID.String != "" {
		var enabled int
		err = tx.QueryRow(
			`SELECT id, name, mode, scope_key, COALESCE(parent_node_id, ''), enabled, status, COALESCE(public_host, ''), COALESCE(public_port, 0)
			 FROM nodes WHERE id = ?`,
			targetID.String,
		).Scan(&node.ID, &node.Name, &node.Mode, &node.ScopeKey, &node.ParentNodeID, &enabled, &node.Status, &node.PublicHost, &node.PublicPort)
		if err != nil {
			return domain.EnrollNodeResult{}, err
		}
		if _, err := tx.Exec(
			`UPDATE nodes
			 SET name = ?, mode = ?, public_host = NULLIF(?, ''), public_port = ?, scope_key = ?, parent_node_id = NULLIF(?, ''), enabled = ?, status = ?, updated_at = ?
			 WHERE id = ?`,
			effectiveName, input.Mode, input.PublicHost, input.PublicPort, input.ScopeKey, input.ParentNodeID, 1, domain.NodeStatusHealthy, now, node.ID,
		); err != nil {
			return domain.EnrollNodeResult{}, err
		}
		node.Name = effectiveName
		node.Mode = input.Mode
		node.ScopeKey = input.ScopeKey
		node.ParentNodeID = input.ParentNodeID
		node.PublicHost = input.PublicHost
		node.PublicPort = input.PublicPort
		node.Enabled = true
		node.Status = domain.NodeStatusHealthy
	} else {
		nodeID, err := s.nextNodeID()
		if err != nil {
			return domain.EnrollNodeResult{}, err
		}
		node = domain.Node{
			ID:           nodeID,
			Name:         effectiveName,
			Mode:         input.Mode,
			ScopeKey:     input.ScopeKey,
			ParentNodeID: input.ParentNodeID,
			Enabled:      true,
			Status:       domain.NodeStatusPending,
			PublicHost:   input.PublicHost,
			PublicPort:   input.PublicPort,
		}
		if _, err := tx.Exec(
			`INSERT INTO nodes (id, name, mode, public_host, public_port, scope_key, parent_node_id, enabled, status, created_at, updated_at)
			 VALUES (?, ?, ?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), ?, ?, ?, ?)`,
			node.ID, node.Name, node.Mode, node.PublicHost, node.PublicPort, node.ScopeKey, node.ParentNodeID, 1, node.Status, now, now,
		); err != nil {
			return domain.EnrollNodeResult{}, err
		}
	}
	if _, err := tx.Exec("UPDATE bootstrap_tokens SET consumed_at = ? WHERE id = ?", now, tokenID); err != nil {
		return domain.EnrollNodeResult{}, err
	}
	if _, err := tx.Exec("DELETE FROM node_api_tokens WHERE node_id = ?", node.ID); err != nil {
		return domain.EnrollNodeResult{}, err
	}
	if _, err := tx.Exec(
		`UPDATE node_trust_materials SET status = ?, updated_at = ? WHERE node_id = ?`,
		domain.TrustMaterialStatusRotated, now, node.ID,
	); err != nil {
		return domain.EnrollNodeResult{}, err
	}
	if _, err := tx.Exec(
		`INSERT INTO node_trust_materials (id, node_id, material_type, material_value, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		newID("trust"), node.ID, "enrollment_secret", enrollmentSecret, domain.TrustMaterialStatusPending, now, now,
	); err != nil {
		return domain.EnrollNodeResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.EnrollNodeResult{}, err
	}
	return domain.EnrollNodeResult{
		Node:             domain.Node{ID: node.ID, Name: node.Name, Mode: node.Mode, ScopeKey: node.ScopeKey, ParentNodeID: node.ParentNodeID, Enabled: node.Enabled, Status: domain.NodeStatusPending, PublicHost: node.PublicHost, PublicPort: node.PublicPort},
		EnrollmentSecret: enrollmentSecret,
		ApprovalState:    domain.ApprovalStatePending,
	}, nil
}

func (s *MySQLStore) ApproveNodeEnrollment(nodeID string, reviewedBy string) (domain.ApproveNodeEnrollmentResult, error) {
	var (
		node    domain.Node
		enabled int
	)
	err := s.db.QueryRow(
		`SELECT id, name, mode, scope_key, COALESCE(parent_node_id, ''), enabled, status, COALESCE(public_host, ''), COALESCE(public_port, 0)
		 FROM nodes WHERE id = ?`,
		nodeID,
	).Scan(&node.ID, &node.Name, &node.Mode, &node.ScopeKey, &node.ParentNodeID, &enabled, &node.Status, &node.PublicHost, &node.PublicPort)
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	node.Enabled = enabled == 1
	if node.Status != domain.NodeStatusPending {
		return domain.ApproveNodeEnrollmentResult{}, fmt.Errorf("node_not_pending")
	}
	trustMaterial, err := auth.RandomToken()
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	accessToken, err := auth.RandomToken()
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	now := nowRFC3339()
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM node_api_tokens WHERE node_id = ?", nodeID); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if _, err := tx.Exec("UPDATE nodes SET status = ?, reviewed_by = ?, reviewed_at = ?, updated_at = ? WHERE id = ?", domain.NodeStatusHealthy, reviewedBy, now, now, nodeID); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if _, err := tx.Exec(
		`UPDATE node_trust_materials SET status = ?, updated_at = ? WHERE node_id = ? AND material_type = 'shared_secret' AND status = 'active'`,
		domain.TrustMaterialStatusRotated, now, nodeID,
	); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if _, err := tx.Exec(
		`INSERT INTO node_trust_materials (id, node_id, material_type, material_value, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		newID("trust"), nodeID, "shared_secret", trustMaterial, domain.TrustMaterialStatusActive, now, now,
	); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if _, err := tx.Exec(
		`INSERT INTO node_api_tokens (id, node_id, token_hash, expires_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		newID("node-token"), nodeID, accessToken, expiresAt, now, now,
	); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	node.Status = domain.NodeStatusHealthy
	return domain.ApproveNodeEnrollmentResult{
		Node:          node,
		AccessToken:   accessToken,
		TrustMaterial: trustMaterial,
		ExpiresAt:     expiresAt,
	}, nil
}

func (s *MySQLStore) ExchangeNodeEnrollment(input domain.ExchangeNodeEnrollmentInput) (domain.ApproveNodeEnrollmentResult, error) {
	var (
		node        domain.Node
		enabled     int
		accessToken string
		expiresAt   string
		trustValue  string
	)
	err := s.db.QueryRow(
		`SELECT id, name, mode, scope_key, COALESCE(parent_node_id, ''), enabled, status, COALESCE(public_host, ''), COALESCE(public_port, 0)
		 FROM nodes WHERE id = ?`,
		input.NodeID,
	).Scan(&node.ID, &node.Name, &node.Mode, &node.ScopeKey, &node.ParentNodeID, &enabled, &node.Status, &node.PublicHost, &node.PublicPort)
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	node.Enabled = enabled == 1
	var enrollmentSecretCount int
	if err := s.db.QueryRow(
		`SELECT COUNT(1) FROM node_trust_materials
		 WHERE node_id = ? AND material_type = 'enrollment_secret' AND material_value = ? AND status = 'pending'`,
		input.NodeID, input.EnrollmentSecret,
	).Scan(&enrollmentSecretCount); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if enrollmentSecretCount == 0 {
		return domain.ApproveNodeEnrollmentResult{}, fmt.Errorf("invalid_enrollment_secret")
	}
	if node.Status == domain.NodeStatusPending {
		return domain.ApproveNodeEnrollmentResult{}, fmt.Errorf("node_enrollment_pending")
	}
	err = s.db.QueryRow(
		`SELECT token_hash, expires_at FROM node_api_tokens WHERE node_id = ? ORDER BY created_at DESC LIMIT 1`,
		input.NodeID,
	).Scan(&accessToken, &expiresAt)
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	err = s.db.QueryRow(
		`SELECT material_value FROM node_trust_materials
		 WHERE node_id = ? AND material_type = 'shared_secret' AND status = 'active'
		 ORDER BY created_at DESC LIMIT 1`,
		input.NodeID,
	).Scan(&trustValue)
	if err != nil {
		trustValue, err = auth.RandomToken()
		if err != nil {
			return domain.ApproveNodeEnrollmentResult{}, err
		}
		_, err = s.db.Exec(
			`INSERT INTO node_trust_materials (id, node_id, material_type, material_value, status, created_at, updated_at)
			 VALUES (?, ?, 'shared_secret', ?, 'active', ?, ?)`,
			newID("trust"), input.NodeID, trustValue, nowRFC3339(), nowRFC3339(),
		)
		if err != nil {
			return domain.ApproveNodeEnrollmentResult{}, err
		}
	}
	if _, err := s.db.Exec(
		`UPDATE node_trust_materials
		 SET status = ?, updated_at = ?
		 WHERE node_id = ? AND material_type = 'enrollment_secret' AND material_value = ? AND status = 'pending'`,
		domain.TrustMaterialStatusConsumed, nowRFC3339(), input.NodeID, input.EnrollmentSecret,
	); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	return domain.ApproveNodeEnrollmentResult{
		Node:          node,
		AccessToken:   accessToken,
		TrustMaterial: trustValue,
		ExpiresAt:     expiresAt,
	}, nil
}

func (s *MySQLStore) ListPendingNodes() []domain.Node {
	rows, err := s.db.Query(
		`SELECT id, name, mode, scope_key, COALESCE(parent_node_id, ''), enabled, status,
		        COALESCE(public_host, ''), COALESCE(public_port, 0),
		        COALESCE(reviewed_by, ''), COALESCE(reviewed_at, ''), COALESCE(reject_reason, '')
		 FROM nodes
		 WHERE status = 'pending'
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	nodes := make([]domain.Node, 0)
	for rows.Next() {
		var node domain.Node
		var enabled int
		if err := rows.Scan(
			&node.ID, &node.Name, &node.Mode, &node.ScopeKey, &node.ParentNodeID,
			&enabled, &node.Status, &node.PublicHost, &node.PublicPort,
			&node.ReviewedBy, &node.ReviewedAt, &node.RejectReason,
		); err != nil {
			continue
		}
		node.Enabled = enabled == 1
		nodes = append(nodes, node)
	}
	return nodes
}

func (s *MySQLStore) RejectNodeEnrollment(nodeID string, reviewedBy string, reason string) error {
	var status string
	err := s.db.QueryRow("SELECT status FROM nodes WHERE id = ?", nodeID).Scan(&status)
	if err != nil {
		return err
	}
	if status != domain.NodeStatusPending {
		return fmt.Errorf("node_not_pending")
	}
	now := nowRFC3339()
	_, err = s.db.Exec(
		"UPDATE nodes SET status = ?, reviewed_by = ?, reviewed_at = ?, reject_reason = ?, updated_at = ? WHERE id = ?",
		domain.ApprovalStateRejected, reviewedBy, now, reason, now, nodeID,
	)
	return err
}

func (s *MySQLStore) AuthenticateNodeToken(accessToken string) (string, bool) {
	var (
		nodeID    string
		expiresAt string
		status    string
		enabled   int
	)
	err := s.db.QueryRow(
		`SELECT t.node_id, t.expires_at, n.status, n.enabled
		 FROM node_api_tokens t
		 JOIN nodes n ON n.id = t.node_id
		 WHERE t.token_hash = ?`,
		accessToken,
	).Scan(&nodeID, &expiresAt, &status, &enabled)
	if err != nil {
		return "", false
	}
	expiry, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().UTC().After(expiry) || enabled != 1 || status == domain.NodeStatusPending {
		return "", false
	}
	return nodeID, true
}
