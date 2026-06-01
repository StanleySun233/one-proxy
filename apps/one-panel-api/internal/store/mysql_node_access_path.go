package store

import (
	"database/sql"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

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
	pathID, err := s.nextID("node_access_path")
	if err != nil {
		return domain.NodeAccessPath{}, err
	}
	item := domain.NodeAccessPath{
		ID:           pathID,
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
	_, err = s.db.Exec(
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
