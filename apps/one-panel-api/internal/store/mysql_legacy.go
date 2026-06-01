package store

import "context"

func (s *MySQLStore) cleanupLegacyDemoTopology(ctx context.Context) error {
	var nodeCount int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM nodes").Scan(&nodeCount); err != nil {
		return err
	}
	if nodeCount == 0 {
		return nil
	}
	var pathCount int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM node_access_paths").Scan(&pathCount); err != nil {
		return err
	}
	if pathCount > 0 {
		return nil
	}
	var taskCount int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM node_onboarding_tasks").Scan(&taskCount); err != nil {
		return err
	}
	if taskCount > 0 {
		return nil
	}
	rows, err := s.db.QueryContext(ctx, "SELECT id FROM nodes ORDER BY id")
	if err != nil {
		return err
	}
	defer rows.Close()
	nodeIDs := make([]string, 0, 4)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		nodeIDs = append(nodeIDs, id)
	}
	if len(nodeIDs) != 4 ||
		nodeIDs[0] != "edge-a" ||
		nodeIDs[1] != "relay-b" ||
		nodeIDs[2] != "relay-c" ||
		nodeIDs[3] != "relay-d" {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		return err
	}
	statements := []string{
		"DELETE FROM node_onboarding_tasks",
		"DELETE FROM node_access_paths",
		"DELETE FROM node_policy_assignments",
		"DELETE FROM node_health_snapshots",
		"DELETE FROM node_api_tokens",
		"DELETE FROM node_trust_materials",
		"DELETE FROM bootstrap_tokens",
		"DELETE FROM certificates",
		"DELETE FROM policy_revisions",
		"DELETE FROM route_rules",
		"DELETE FROM chain_hops",
		"DELETE FROM chains",
		"DELETE FROM node_links",
		"DELETE FROM nodes",
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 1"); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *MySQLStore) repairLegacyUnreportedNodeStatus(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE nodes
		 SET status = 'healthy', updated_at = ?
		 WHERE status = 'degraded'
		   AND id NOT IN (SELECT node_id FROM node_health_snapshots)`,
		nowRFC3339(),
	)
	return err
}
