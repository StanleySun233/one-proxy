package store

import (
	"database/sql"
	"encoding/json"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (s *MySQLStore) loadChainHops(chainID string) []string {
	rows, err := s.db.Query("SELECT node_id FROM chain_hops WHERE chain_id = ? ORDER BY hop_index", chainID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	hops := make([]string, 0)
	for rows.Next() {
		var nodeID string
		if err := rows.Scan(&nodeID); err != nil {
			continue
		}
		hops = append(hops, nodeID)
	}
	return hops
}

func (s *MySQLStore) ListChains() []domain.Chain {
	rows, err := s.db.Query("SELECT id, name, destination_scope, enabled FROM chains ORDER BY name")
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.Chain, 0)
	for rows.Next() {
		var item domain.Chain
		var enabled int
		if err := rows.Scan(&item.ID, &item.Name, &item.DestinationScope, &enabled); err != nil {
			continue
		}
		item.Enabled = enabled == 1
		item.Hops = s.loadChainHops(item.ID)
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) GetChainProbeResult(chainID string) (domain.ChainProbeResult, bool) {
	var item domain.ChainProbeResult
	var hopsJSON string
	err := s.db.QueryRow(
		`SELECT chain_id, status, message, resolved_hops_json, COALESCE(blocking_node_id, ''), COALESCE(blocking_reason, ''), COALESCE(target_host, ''), target_port, probed_at
		 FROM chain_probe_results WHERE chain_id = ?`,
		chainID,
	).Scan(&item.ChainID, &item.Status, &item.Message, &hopsJSON, &item.BlockingNodeID, &item.BlockingReason, &item.TargetHost, &item.TargetPort, &item.ProbedAt)
	if err != nil {
		return domain.ChainProbeResult{}, false
	}
	_ = json.Unmarshal([]byte(hopsJSON), &item.ResolvedHops)
	return item, true
}

func (s *MySQLStore) SaveChainProbeResult(input domain.SaveChainProbeResultInput) (domain.ChainProbeResult, error) {
	hopsJSON, err := json.Marshal(input.ResolvedHops)
	if err != nil {
		return domain.ChainProbeResult{}, err
	}
	_, err = s.db.Exec(
		`INSERT INTO chain_probe_results (chain_id, status, message, resolved_hops_json, blocking_node_id, blocking_reason, target_host, target_port, probed_at)
		 VALUES (?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?)
		 ON DUPLICATE KEY UPDATE
		   status = VALUES(status),
		   message = VALUES(message),
		   resolved_hops_json = VALUES(resolved_hops_json),
		   blocking_node_id = VALUES(blocking_node_id),
		   blocking_reason = VALUES(blocking_reason),
		   target_host = VALUES(target_host),
		   target_port = VALUES(target_port),
		   probed_at = VALUES(probed_at)`,
		input.ChainID, input.Status, input.Message, string(hopsJSON), input.BlockingNodeID, input.BlockingReason, input.TargetHost, input.TargetPort, input.ProbedAt,
	)
	if err != nil {
		return domain.ChainProbeResult{}, err
	}
	return domain.ChainProbeResult{
		ChainID:        input.ChainID,
		Status:         input.Status,
		Message:        input.Message,
		ResolvedHops:   input.ResolvedHops,
		BlockingNodeID: input.BlockingNodeID,
		BlockingReason: input.BlockingReason,
		TargetHost:     input.TargetHost,
		TargetPort:     input.TargetPort,
		ProbedAt:       input.ProbedAt,
	}, nil
}

func (s *MySQLStore) CreateChain(input domain.CreateChainInput) (domain.Chain, error) {
	item := domain.Chain{ID: newID("chain"), Name: input.Name, DestinationScope: input.DestinationScope, Enabled: true, Hops: input.Hops}
	now := nowRFC3339()
	tx, err := s.db.Begin()
	if err != nil {
		return domain.Chain{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`INSERT INTO chains (id, name, destination_scope, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		item.ID, item.Name, item.DestinationScope, 1, now, now,
	); err != nil {
		return domain.Chain{}, err
	}
	for index, hop := range item.Hops {
		if _, err := tx.Exec("INSERT INTO chain_hops (chain_id, hop_index, node_id) VALUES (?, ?, ?)", item.ID, index, hop); err != nil {
			return domain.Chain{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.Chain{}, err
	}
	return item, nil
}

func (s *MySQLStore) UpdateChain(chainID string, input domain.UpdateChainInput) (domain.Chain, error) {
	now := nowRFC3339()
	tx, err := s.db.Begin()
	if err != nil {
		return domain.Chain{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`UPDATE chains SET name = ?, destination_scope = ?, enabled = ?, updated_at = ? WHERE id = ?`,
		input.Name, input.DestinationScope, boolToInt(input.Enabled), now, chainID,
	); err != nil {
		return domain.Chain{}, err
	}
	if _, err := tx.Exec("DELETE FROM chain_hops WHERE chain_id = ?", chainID); err != nil {
		return domain.Chain{}, err
	}
	for index, hop := range input.Hops {
		if _, err := tx.Exec("INSERT INTO chain_hops (chain_id, hop_index, node_id) VALUES (?, ?, ?)", chainID, index, hop); err != nil {
			return domain.Chain{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.Chain{}, err
	}
	for _, item := range s.ListChains() {
		if item.ID == chainID {
			return item, nil
		}
	}
	return domain.Chain{}, sql.ErrNoRows
}

func (s *MySQLStore) DeleteChain(chainID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM chain_hops WHERE chain_id = ?", chainID); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM chains WHERE id = ?", chainID); err != nil {
		return err
	}
	return tx.Commit()
}
