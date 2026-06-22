package store

import (
	"fmt"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

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
	now := nowRFC3339()
	detailsJSON := encodeJSONMap(input.Details)
	if input.TransportType == domain.TransportTypeReverseWSParent && input.ParentNodeID != "" {
		if _, err := s.db.Exec(
			`DELETE FROM node_transports
			 WHERE node_id = ? AND transport_type = ? AND direction = ? AND parent_node_id = ? AND address <> ?`,
			input.NodeID, input.TransportType, input.Direction, input.ParentNodeID, input.Address,
		); err != nil {
			return domain.NodeTransport{}, err
		}
	}
	existingID := ""
	_ = s.db.QueryRow(
		`SELECT id FROM node_transports WHERE node_id = ? AND transport_type = ? AND address = ? LIMIT 1`,
		input.NodeID, input.TransportType, input.Address,
	).Scan(&existingID)
	id := existingID
	if id == "" {
		nextID, err := s.nextID("node_transport")
		if err != nil {
			return domain.NodeTransport{}, err
		}
		id = nextID
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
	if !healthyTransportValues[input.Status] {
		if _, err := s.db.Exec("UPDATE nodes SET status = ?, updated_at = ? WHERE id = ?", domain.NodeStatusDegraded, now, input.NodeID); err != nil {
			return domain.NodeTransport{}, err
		}
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
