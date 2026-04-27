package store

import (
	"database/sql"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

var healthyListenerValues = map[string]bool{domain.ListenerStatusUp: true}
var healthyCertValues = map[string]bool{domain.CertStatusHealthy: true, domain.CertStatusRenewed: true}

func heartbeatNodeStatus(listenerStatus map[string]string, certStatus map[string]string) string {
	for _, value := range listenerStatus {
		if !healthyListenerValues[value] {
			return domain.NodeStatusDegraded
		}
	}
	for _, value := range certStatus {
		if !healthyCertValues[value] {
			return domain.NodeStatusDegraded
		}
	}
	return domain.NodeStatusHealthy
}

func (s *MySQLStore) ListNodeHealth() []domain.NodeHealth {
	rows, err := s.db.Query(
		`SELECT node_id, heartbeat_at, COALESCE(policy_revision_id, ''), listener_status_json, cert_status_json
		 FROM node_health_snapshots ORDER BY node_id`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.NodeHealth, 0)
	for rows.Next() {
		var item domain.NodeHealth
		var listenerJSON string
		var certJSON string
		if err := rows.Scan(&item.NodeID, &item.HeartbeatAt, &item.PolicyRevisionID, &listenerJSON, &certJSON); err != nil {
			continue
		}
		item.ListenerStatus = decodeJSONMap(listenerJSON)
		item.CertStatus = decodeJSONMap(certJSON)
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) ListNodeHealthHistory(nodeID string, window time.Duration) ([]domain.NodeHealth, error) {
	since := time.Now().Add(-window).UTC().Format(time.RFC3339)
	rows, err := s.db.Query(
		`SELECT node_id, heartbeat_at, policy_revision_id, listener_status_json, cert_status_json
		 FROM node_health_history
		 WHERE node_id = ? AND heartbeat_at >= ?
		 ORDER BY heartbeat_at ASC`, nodeID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.NodeHealth, 0)
	for rows.Next() {
		var item domain.NodeHealth
		var listenerJSON, certJSON string
		if err := rows.Scan(&item.NodeID, &item.HeartbeatAt, &item.PolicyRevisionID, &listenerJSON, &certJSON); err != nil {
			continue
		}
		item.ListenerStatus = decodeJSONMap(listenerJSON)
		item.CertStatus = decodeJSONMap(certJSON)
		items = append(items, item)
	}
	return items, nil
}

func (s *MySQLStore) UpsertNodeHeartbeat(input domain.NodeHeartbeatInput) (domain.NodeHealth, error) {
	now := nowRFC3339()
	status := heartbeatNodeStatus(input.ListenerStatus, input.CertStatus)
	revisionID := input.PolicyRevisionID
	if revisionID != "" {
		var found int
		if err := s.db.QueryRow("SELECT 1 FROM policy_revisions WHERE id = ?", revisionID).Scan(&found); err != nil {
			revisionID = ""
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return domain.NodeHealth{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`INSERT INTO node_health_snapshots (node_id, heartbeat_at, policy_revision_id, listener_status_json, cert_status_json, updated_at)
		 VALUES (?, ?, NULLIF(?, ''), ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		   heartbeat_at = VALUES(heartbeat_at),
		   policy_revision_id = VALUES(policy_revision_id),
		   listener_status_json = VALUES(listener_status_json),
		   cert_status_json = VALUES(cert_status_json),
		   updated_at = VALUES(updated_at)`,
		input.NodeID, now, revisionID, encodeJSONMap(input.ListenerStatus), encodeJSONMap(input.CertStatus), now,
	); err != nil {
		return domain.NodeHealth{}, err
	}
	if _, err := tx.Exec(
		`INSERT INTO node_health_history (node_id, heartbeat_at, policy_revision_id, listener_status_json, cert_status_json, created_at)
		 VALUES (?, ?, NULLIF(?, ''), ?, ?, ?)`,
		input.NodeID, now, revisionID, encodeJSONMap(input.ListenerStatus), encodeJSONMap(input.CertStatus), now,
	); err != nil {
		return domain.NodeHealth{}, err
	}
	if _, err := tx.Exec("UPDATE nodes SET status = ?, updated_at = ? WHERE id = ?", status, now, input.NodeID); err != nil {
		return domain.NodeHealth{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.NodeHealth{}, err
	}
	return domain.NodeHealth{
		NodeID:           input.NodeID,
		HeartbeatAt:      now,
		PolicyRevisionID: revisionID,
		ListenerStatus:   input.ListenerStatus,
		CertStatus:       input.CertStatus,
	}, nil
}

func (s *MySQLStore) RenewNodeCertificate(input domain.NodeCertRenewInput) (domain.NodeCertRenewResult, error) {
	now := nowRFC3339()
	notAfter := time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339)
	var certID string
	provider := "internal_ca"
	if input.CertType == domain.CertTypePublic {
		provider = "lets_encrypt"
	}
	err := s.db.QueryRow(
		`SELECT id FROM certificates WHERE owner_type = 'node' AND owner_id = ? AND cert_type = ? LIMIT 1`,
		input.NodeID, input.CertType,
	).Scan(&certID)
	if err != nil {
		certID = newID("cert")
		_, err = s.db.Exec(
			`INSERT INTO certificates (id, owner_type, owner_id, cert_type, provider, status, not_before, not_after, created_at, updated_at)
			 VALUES (?, 'node', ?, ?, ?, ?, ?, ?, ?, ?)`,
			certID, input.NodeID, input.CertType, provider, domain.CertStatusRenewed, now, notAfter, now, now,
		)
	} else {
		_, err = s.db.Exec(
			`UPDATE certificates SET provider = ?, status = ?, not_before = ?, not_after = ?, updated_at = ? WHERE id = ?`,
			provider, domain.CertStatusRenewed, now, notAfter, now, certID,
		)
	}
	if err != nil {
		return domain.NodeCertRenewResult{}, err
	}
	return domain.NodeCertRenewResult{
		NodeID:   input.NodeID,
		CertType: input.CertType,
		Status:   domain.CertStatusRenewed,
		NotAfter: notAfter,
	}, nil
}
