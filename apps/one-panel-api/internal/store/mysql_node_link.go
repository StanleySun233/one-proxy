package store

import "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"

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
	linkID, err := s.nextID("node_link")
	if err != nil {
		return domain.NodeLink{}, err
	}
	item := domain.NodeLink{
		ID:           linkID,
		SourceNodeID: input.SourceNodeID,
		TargetNodeID: input.TargetNodeID,
		LinkType:     input.LinkType,
		TrustState:   input.TrustState,
	}
	now := nowRFC3339()
	_, err = s.db.Exec(
		`INSERT INTO node_links (id, source_node_id, target_node_id, link_type, trust_state, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.SourceNodeID, item.TargetNodeID, item.LinkType, item.TrustState, now, now,
	)
	return item, err
}

func (s *MySQLStore) UpdateNodeLink(linkID string, input domain.UpdateNodeLinkInput) (domain.NodeLink, error) {
	item := domain.NodeLink{
		ID:           linkID,
		SourceNodeID: input.SourceNodeID,
		TargetNodeID: input.TargetNodeID,
		LinkType:     input.LinkType,
		TrustState:   input.TrustState,
	}
	_, err := s.db.Exec(
		`UPDATE node_links
		 SET source_node_id = ?, target_node_id = ?, link_type = ?, trust_state = ?, updated_at = ?
		 WHERE id = ?`,
		item.SourceNodeID, item.TargetNodeID, item.LinkType, item.TrustState, nowRFC3339(), item.ID,
	)
	return item, err
}

func (s *MySQLStore) DeleteNodeLink(linkID string) error {
	_, err := s.db.Exec(`DELETE FROM node_links WHERE id = ?`, linkID)
	return err
}
