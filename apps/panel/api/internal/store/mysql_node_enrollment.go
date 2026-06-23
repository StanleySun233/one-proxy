package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/auth"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func (s *MySQLStore) EnrollNode(input domain.EnrollNodeInput) (domain.EnrollNodeResult, error) {
	var (
		tokenID      string
		targetID     sql.NullString
		nodeName     sql.NullString
		nodeMode     sql.NullString
		scopeKey     sql.NullString
		parentNodeID sql.NullString
		publicHost   sql.NullString
		publicPort   sql.NullInt64
		expiresAt    string
		consumedAt   sql.NullString
	)
	err := s.db.QueryRow(
		`SELECT id, target_id, node_name, node_mode, scope_key, parent_node_id, public_host, public_port, expires_at, consumed_at FROM bootstrap_tokens WHERE token_hash = ?`,
		auth.TokenHash(input.Token),
	).Scan(&tokenID, &targetID, &nodeName, &nodeMode, &scopeKey, &parentNodeID, &publicHost, &publicPort, &expiresAt, &consumedAt)
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
	effectiveMode := input.Mode
	if nodeMode.Valid && nodeMode.String != "" {
		effectiveMode = nodeMode.String
	}
	effectiveScopeKey := input.ScopeKey
	if scopeKey.Valid && scopeKey.String != "" {
		effectiveScopeKey = scopeKey.String
	}
	effectiveParentNodeID := input.ParentNodeID
	if parentNodeID.Valid {
		effectiveParentNodeID = parentNodeID.String
	}
	effectivePublicHost := ""
	if effectiveMode == domain.NodeModeEdge {
		effectivePublicHost = input.PublicHost
	}
	if publicHost.Valid && publicHost.String != "" {
		effectivePublicHost = publicHost.String
	}
	effectivePublicPort := 0
	if effectiveMode == domain.NodeModeEdge {
		effectivePublicPort = input.PublicPort
	}
	if publicPort.Valid && publicPort.Int64 > 0 {
		effectivePublicPort = int(publicPort.Int64)
	}
	if effectiveMode == domain.NodeModeEdge && effectivePublicPort <= 0 {
		effectivePublicPort = input.PublicPort
	}
	now := nowRFC3339()
	enrollmentSecret, err := auth.RandomToken()
	if err != nil {
		return domain.EnrollNodeResult{}, err
	}
	enrollmentTrustID, err := s.nextID("trust_material")
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
			effectiveName, effectiveMode, effectivePublicHost, effectivePublicPort, effectiveScopeKey, effectiveParentNodeID, 1, domain.NodeStatusPending, now, node.ID,
		); err != nil {
			return domain.EnrollNodeResult{}, err
		}
		node.Name = effectiveName
		node.Mode = effectiveMode
		node.ScopeKey = effectiveScopeKey
		node.ParentNodeID = effectiveParentNodeID
		node.PublicHost = effectivePublicHost
		node.PublicPort = effectivePublicPort
		node.Enabled = true
		node.Status = domain.NodeStatusPending
	} else {
		return domain.EnrollNodeResult{}, fmt.Errorf("bootstrap token missing target node")
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
		enrollmentTrustID, node.ID, "enrollment_secret", enrollmentSecret, domain.TrustMaterialStatusPending, now, now,
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
	var enrollmentSecretCount int
	if err := s.db.QueryRow(
		`SELECT COUNT(1) FROM node_trust_materials
		 WHERE node_id = ? AND material_type = 'enrollment_secret' AND status = 'pending'`,
		nodeID,
	).Scan(&enrollmentSecretCount); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if enrollmentSecretCount == 0 {
		return domain.ApproveNodeEnrollmentResult{}, fmt.Errorf("node_not_enrolled")
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
	trustID, err := s.nextID("trust_material")
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	nodeTokenID, err := s.nextID("node_api_token")
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
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
		trustID, nodeID, "shared_secret", trustMaterial, domain.TrustMaterialStatusActive, now, now,
	); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if _, err := tx.Exec(
		`INSERT INTO node_api_tokens (id, node_id, token_hash, expires_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		nodeTokenID, nodeID, auth.TokenHash(accessToken), expiresAt, now, now,
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
		node       domain.Node
		enabled    int
		trustValue string
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
		trustID, err := s.nextID("trust_material")
		if err != nil {
			return domain.ApproveNodeEnrollmentResult{}, err
		}
		_, err = s.db.Exec(
			`INSERT INTO node_trust_materials (id, node_id, material_type, material_value, status, created_at, updated_at)
			 VALUES (?, ?, 'shared_secret', ?, 'active', ?, ?)`,
			trustID, input.NodeID, trustValue, nowRFC3339(), nowRFC3339(),
		)
		if err != nil {
			return domain.ApproveNodeEnrollmentResult{}, err
		}
	}
	accessToken, err := auth.RandomToken()
	if err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	nodeTokenID, err := s.nextID("node_api_token")
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
	if _, err := tx.Exec(
		`UPDATE node_trust_materials
		 SET status = ?, updated_at = ?
		 WHERE node_id = ? AND material_type = 'enrollment_secret' AND material_value = ? AND status = 'pending'`,
		domain.TrustMaterialStatusConsumed, now, input.NodeID, input.EnrollmentSecret,
	); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if _, err := tx.Exec(
		`INSERT INTO node_api_tokens (id, node_id, token_hash, expires_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		nodeTokenID, input.NodeID, auth.TokenHash(accessToken), expiresAt, now, now,
	); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	if err := tx.Commit(); err != nil {
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
		   AND EXISTS (
		     SELECT 1 FROM node_trust_materials ntm
		     WHERE ntm.node_id = nodes.id
		       AND ntm.material_type = 'enrollment_secret'
		       AND ntm.status = 'pending'
		   )
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
	var enrollmentSecretCount int
	if err := s.db.QueryRow(
		`SELECT COUNT(1) FROM node_trust_materials
		 WHERE node_id = ? AND material_type = 'enrollment_secret' AND status = 'pending'`,
		nodeID,
	).Scan(&enrollmentSecretCount); err != nil {
		return err
	}
	if enrollmentSecretCount == 0 {
		return fmt.Errorf("node_not_enrolled")
	}
	now := nowRFC3339()
	_, err = s.db.Exec(
		"UPDATE nodes SET status = ?, reviewed_by = ?, reviewed_at = ?, reject_reason = ?, updated_at = ? WHERE id = ?",
		domain.ApprovalStateRejected, reviewedBy, now, reason, now, nodeID,
	)
	return err
}
