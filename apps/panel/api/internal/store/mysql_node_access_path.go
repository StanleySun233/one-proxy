package store

import (
	"database/sql"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func (s *MySQLStore) ListNodeAccessPaths() []domain.NodeAccessPath {
	rows, err := s.db.Query(
		`SELECT id, create_id, owner_id, COALESCE(chain_id, ''), name, mode, protocol, service_type, COALESCE(target_node_id, ''), COALESCE(entry_node_id, ''), relay_node_ids_json,
		        COALESCE(listen_host, ''), COALESCE(listen_port, 0), target_protocol, COALESCE(target_host, ''), COALESCE(target_port, 0),
		        COALESCE(target_sni, ''), tls_mode, auth_mode, COALESCE(options_json, '{}'), enabled
		 FROM node_access_paths
		 ORDER BY name`,
	)
	return s.scanNodeAccessPaths(rows, err)
}

func (s *MySQLStore) ListNodeAccessPathsForTenant(tenantCtx domain.TenantAuthContext) []domain.NodeAccessPath {
	if tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.TenantID == "" {
		return s.ListNodeAccessPaths()
	}
	rows, err := s.db.Query(
		`SELECT nap.id, nap.create_id, nap.owner_id, COALESCE(nap.chain_id, ''), nap.name, nap.mode, nap.protocol, nap.service_type, COALESCE(nap.target_node_id, ''), COALESCE(nap.entry_node_id, ''), nap.relay_node_ids_json,
		        COALESCE(nap.listen_host, ''), COALESCE(nap.listen_port, 0), nap.target_protocol, COALESCE(nap.target_host, ''), COALESCE(nap.target_port, 0),
		        COALESCE(nap.target_sni, ''), nap.tls_mode, nap.auth_mode, COALESCE(nap.options_json, '{}'), nap.enabled, tap.permission
		 FROM node_access_paths nap
		 JOIN tenant_access_paths tap ON tap.access_path_id = nap.id
		 WHERE tap.tenant_id = ? AND tap.permission IN (?, ?)
		 ORDER BY nap.name`,
		tenantCtx.ActiveTenant.TenantID, domain.BindingPermissionUse, domain.BindingPermissionManage,
	)
	return s.scanNodeAccessPaths(rows, err)
}

func (s *MySQLStore) scanNodeAccessPaths(rows *sql.Rows, err error) []domain.NodeAccessPath {
	if err != nil {
		return nil
	}
	defer rows.Close()
	columns, _ := rows.Columns()
	hasPermission := len(columns) == 22
	items := make([]domain.NodeAccessPath, 0)
	for rows.Next() {
		var item domain.NodeAccessPath
		var relayJSON string
		var optionsJSON string
		var enabled int
		if hasPermission {
			if err := rows.Scan(
				&item.ID, &item.CreateID, &item.OwnerID, &item.ChainID, &item.Name, &item.Mode, &item.Protocol, &item.ServiceType, &item.TargetNodeID, &item.EntryNodeID, &relayJSON,
				&item.ListenHost, &item.ListenPort, &item.TargetProtocol, &item.TargetHost, &item.TargetPort,
				&item.TargetSNI, &item.TLSMode, &item.AuthMode, &optionsJSON, &enabled, &item.Permission,
			); err != nil {
				continue
			}
		} else {
			if err := rows.Scan(
				&item.ID, &item.CreateID, &item.OwnerID, &item.ChainID, &item.Name, &item.Mode, &item.Protocol, &item.ServiceType, &item.TargetNodeID, &item.EntryNodeID, &relayJSON,
				&item.ListenHost, &item.ListenPort, &item.TargetProtocol, &item.TargetHost, &item.TargetPort,
				&item.TargetSNI, &item.TLSMode, &item.AuthMode, &optionsJSON, &enabled,
			); err != nil {
				continue
			}
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
		ChainID:        input.ChainID,
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
		 (id, chain_id, name, mode, protocol, service_type, target_node_id, entry_node_id, relay_node_ids_json, listen_host, listen_port,
		  target_protocol, target_host, target_port, target_sni, tls_mode, auth_mode, options_json, enabled, created_at, updated_at)
		 VALUES (?, NULLIF(?, ''), ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), ?, NULLIF(?, ''), ?, ?, ?, ?, ?, ?)`,
		item.ID, item.ChainID, item.Name, item.Mode, item.Protocol, item.ServiceType, item.TargetNodeID, item.EntryNodeID, encodeJSONStringSlice(item.RelayNodeIDs),
		item.ListenHost, item.ListenPort, item.TargetProtocol, item.TargetHost, item.TargetPort, item.TargetSNI, item.TLSMode, item.AuthMode,
		encodeJSONMap(item.Options), 1, now, now,
	)
	return item, err
}

func (s *MySQLStore) CreateNodeAccessPathForTenant(tenantCtx domain.TenantAuthContext, input domain.CreateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	pathID, err := s.nextID("node_access_path")
	if err != nil {
		return domain.NodeAccessPath{}, err
	}
	item := domain.NodeAccessPath{
		ID:             pathID,
		CreateID:       tenantCtx.Account.ID,
		OwnerID:        tenantCtx.Account.ID,
		ChainID:        input.ChainID,
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
	tx, err := s.db.Begin()
	if err != nil {
		return domain.NodeAccessPath{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`INSERT INTO node_access_paths
		 (id, create_id, owner_id, chain_id, name, mode, protocol, service_type, target_node_id, entry_node_id, relay_node_ids_json, listen_host, listen_port,
		  target_protocol, target_host, target_port, target_sni, tls_mode, auth_mode, options_json, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, NULLIF(?, ''), ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), ?, NULLIF(?, ''), ?, ?, ?, ?, ?, ?)`,
		item.ID, item.CreateID, item.OwnerID, item.ChainID, item.Name, item.Mode, item.Protocol, item.ServiceType, item.TargetNodeID, item.EntryNodeID, encodeJSONStringSlice(item.RelayNodeIDs),
		item.ListenHost, item.ListenPort, item.TargetProtocol, item.TargetHost, item.TargetPort, item.TargetSNI, item.TLSMode, item.AuthMode,
		encodeJSONMap(item.Options), 1, now, now,
	); err != nil {
		return domain.NodeAccessPath{}, err
	}
	if tenantCtx.ActiveTenant.TenantID != "" {
		if err := bindTenantResource(tx, "tenant_access_paths", "access_path_id", tenantCtx.ActiveTenant.TenantID, item.ID, tenantCtx.Account.ID); err != nil {
			return domain.NodeAccessPath{}, err
		}
	}
	return item, tx.Commit()
}

func (s *MySQLStore) UpdateNodeAccessPath(pathID string, input domain.UpdateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	now := nowRFC3339()
	_, err := s.db.Exec(
		`UPDATE node_access_paths
		 SET chain_id = NULLIF(?, ''), name = ?, mode = ?, protocol = ?, service_type = ?, target_node_id = NULLIF(?, ''), entry_node_id = NULLIF(?, ''),
		     relay_node_ids_json = ?, listen_host = NULLIF(?, ''), listen_port = ?, target_protocol = ?, target_host = NULLIF(?, ''),
		     target_port = ?, target_sni = NULLIF(?, ''), tls_mode = ?, auth_mode = ?, options_json = ?, enabled = ?, updated_at = ?
		 WHERE id = ?`,
		input.ChainID, input.Name, input.Mode, input.Protocol, input.ServiceType, input.TargetNodeID, input.EntryNodeID, encodeJSONStringSlice(input.RelayNodeIDs),
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

func (s *MySQLStore) NodeAccessPathBindingPermission(tenantCtx domain.TenantAuthContext, pathID string) (domain.BindingPermission, bool) {
	return s.tenantResourcePermission(tenantCtx, "tenant_access_paths", "access_path_id", pathID)
}

func (s *MySQLStore) CountNodeAccessPathBindings(pathID string) int {
	return s.countTenantResourceBindings("tenant_access_paths", "access_path_id", pathID)
}
