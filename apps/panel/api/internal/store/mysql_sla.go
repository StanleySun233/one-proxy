package store

import "github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"

func (s *MySQLStore) UpsertNodeSLAMinute(input domain.NodeSLAMinuteInput) error {
	now := nowRFC3339()
	_, err := s.db.Exec(
		`INSERT INTO node_sla_minutes (scenario_id, node_id, window_start, expected_heartbeats, received_heartbeats, success, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		   expected_heartbeats = VALUES(expected_heartbeats),
		   received_heartbeats = VALUES(received_heartbeats),
		   success = VALUES(success),
		   updated_at = VALUES(updated_at)`,
		"node:"+input.NodeID,
		input.NodeID,
		input.WindowStart,
		input.ExpectedHeartbeats,
		input.ReceivedHeartbeats,
		input.Success,
		now,
		now,
	)
	return err
}

func (s *MySQLStore) ListNodeSLAMinutes(since string) ([]domain.NodeSLAMinute, error) {
	rows, err := s.db.Query(
		`SELECT m.scenario_id, m.node_id, n.name, m.window_start, m.expected_heartbeats, m.received_heartbeats, m.success, m.created_at, m.updated_at
		 FROM node_sla_minutes m
		 JOIN nodes n ON n.id = m.node_id
		 WHERE m.window_start >= ?
		 ORDER BY m.window_start DESC, n.name ASC`,
		since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.NodeSLAMinute, 0)
	for rows.Next() {
		var item domain.NodeSLAMinute
		if err := rows.Scan(&item.ScenarioID, &item.NodeID, &item.NodeName, &item.WindowStart, &item.ExpectedHeartbeats, &item.ReceivedHeartbeats, &item.Success, &item.CreatedAt, &item.UpdatedAt); err != nil {
			continue
		}
		item.ScenarioName = item.NodeName + " heartbeat"
		items = append(items, item)
	}
	return items, rows.Err()
}
