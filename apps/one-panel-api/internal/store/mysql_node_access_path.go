package store

import (
	"database/sql"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (s *MySQLStore) ListNodeAccessPaths() []domain.NodeAccessPath {
	rows, err := s.db.Query(
		`SELECT id, name, mode, protocol, service_type, COALESCE(target_node_id, ''), COALESCE(entry_node_id, ''), relay_node_ids_json,
		        COALESCE(listen_host, ''), COALESCE(listen_port, 0), target_protocol, COALESCE(target_host, ''), COALESCE(target_port, 0),
		        COALESCE(target_sni, ''), tls_mode, auth_mode, COALESCE(options_json, '{}'), enabled
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
		var optionsJSON string
		var enabled int
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Mode, &item.Protocol, &item.ServiceType, &item.TargetNodeID, &item.EntryNodeID, &relayJSON,
			&item.ListenHost, &item.ListenPort, &item.TargetProtocol, &item.TargetHost, &item.TargetPort,
			&item.TargetSNI, &item.TLSMode, &item.AuthMode, &optionsJSON, &enabled,
		); err != nil {
			continue
		}
		item.RelayNodeIDs = decodeJSONStringSlice(relayJSON)
		item.Options = decodeJSONMap(optionsJSON)
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
		ID:             pathID,
		Name:           input.Name,
		Mode:           input.Mode,
		Protocol:       input.Protocol,
		ServiceType:    input.ServiceType,
		TargetNodeID:   input.TargetNodeID,
		EntryNodeID:    input.EntryNodeID,
		RelayNodeIDs:   normalizeStringSlice(input.RelayNodeIDs),
		ListenHost:     input.ListenHost,
		ListenPort:     input.ListenPort,
		TargetProtocol: input.TargetProtocol,
		TargetHost:     input.TargetHost,
		TargetPort:     input.TargetPort,
		TargetSNI:      input.TargetSNI,
		TLSMode:        input.TLSMode,
		AuthMode:       input.AuthMode,
		Options:        input.Options,
		Enabled:        true,
	}
	now := nowRFC3339()
	_, err = s.db.Exec(
		`INSERT INTO node_access_paths
		 (id, name, mode, protocol, service_type, target_node_id, entry_node_id, relay_node_ids_json, listen_host, listen_port,
		  target_protocol, target_host, target_port, target_sni, tls_mode, auth_mode, options_json, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), ?, NULLIF(?, ''), ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Name, item.Mode, item.Protocol, item.ServiceType, item.TargetNodeID, item.EntryNodeID, encodeJSONStringSlice(item.RelayNodeIDs),
		item.ListenHost, item.ListenPort, item.TargetProtocol, item.TargetHost, item.TargetPort, item.TargetSNI, item.TLSMode, item.AuthMode,
		encodeJSONMap(item.Options), 1, now, now,
	)
	return item, err
}

func (s *MySQLStore) UpdateNodeAccessPath(pathID string, input domain.UpdateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	now := nowRFC3339()
	_, err := s.db.Exec(
		`UPDATE node_access_paths
		 SET name = ?, mode = ?, protocol = ?, service_type = ?, target_node_id = NULLIF(?, ''), entry_node_id = NULLIF(?, ''),
		     relay_node_ids_json = ?, listen_host = NULLIF(?, ''), listen_port = ?, target_protocol = ?, target_host = NULLIF(?, ''),
		     target_port = ?, target_sni = NULLIF(?, ''), tls_mode = ?, auth_mode = ?, options_json = ?, enabled = ?, updated_at = ?
		 WHERE id = ?`,
		input.Name, input.Mode, input.Protocol, input.ServiceType, input.TargetNodeID, input.EntryNodeID, encodeJSONStringSlice(input.RelayNodeIDs),
		input.ListenHost, input.ListenPort, input.TargetProtocol, input.TargetHost, input.TargetPort, input.TargetSNI, input.TLSMode, input.AuthMode,
		encodeJSONMap(input.Options), boolToInt(input.Enabled), now, pathID,
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
