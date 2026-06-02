package store

import (
	"database/sql"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (s *MySQLStore) ListNodeLinks() []domain.NodeLink {
	rows, err := s.db.Query(
		`SELECT id, create_id, owner_id, source_node_id, target_node_id, link_type, trust_state FROM node_links ORDER BY source_node_id, target_node_id`,
	)
	return s.scanNodeLinks(rows, err)
}

func (s *MySQLStore) ListNodeLinksForTenant(tenantCtx domain.TenantAuthContext) []domain.NodeLink {
	if tenantCtx.SuperAdmin {
		return s.ListNodeLinks()
	}
	rows, err := s.db.Query(
		`SELECT nl.id, nl.create_id, nl.owner_id, nl.source_node_id, nl.target_node_id, nl.link_type, nl.trust_state
		 FROM node_links nl
		 JOIN tenant_node_links tnl ON tnl.node_link_id = nl.id
		 WHERE tnl.tenant_id = ?
		 ORDER BY nl.source_node_id, nl.target_node_id`,
		tenantCtx.ActiveTenant.TenantID,
	)
	return s.scanNodeLinks(rows, err)
}

func (s *MySQLStore) scanNodeLinks(rows *sql.Rows, err error) []domain.NodeLink {
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.NodeLink, 0)
	for rows.Next() {
		var item domain.NodeLink
		if err := rows.Scan(&item.ID, &item.CreateID, &item.OwnerID, &item.SourceNodeID, &item.TargetNodeID, &item.LinkType, &item.TrustState); err != nil {
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

func (s *MySQLStore) CreateNodeLinkForTenant(tenantCtx domain.TenantAuthContext, input domain.CreateNodeLinkInput) (domain.NodeLink, error) {
	linkID, err := s.nextID("node_link")
	if err != nil {
		return domain.NodeLink{}, err
	}
	item := domain.NodeLink{
		ID:           linkID,
		CreateID:     tenantCtx.Account.ID,
		OwnerID:      tenantCtx.Account.ID,
		SourceNodeID: input.SourceNodeID,
		TargetNodeID: input.TargetNodeID,
		LinkType:     input.LinkType,
		TrustState:   input.TrustState,
	}
	now := nowRFC3339()
	tx, err := s.db.Begin()
	if err != nil {
		return domain.NodeLink{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`INSERT INTO node_links (id, create_id, owner_id, source_node_id, target_node_id, link_type, trust_state, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.CreateID, item.OwnerID, item.SourceNodeID, item.TargetNodeID, item.LinkType, item.TrustState, now, now,
	); err != nil {
		return domain.NodeLink{}, err
	}
	if !tenantCtx.SuperAdmin {
		if err := bindTenantResource(tx, "tenant_node_links", "node_link_id", tenantCtx.ActiveTenant.TenantID, item.ID, tenantCtx.Account.ID); err != nil {
			return domain.NodeLink{}, err
		}
	}
	return item, tx.Commit()
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

func (s *MySQLStore) NodeLinkBindingPermission(tenantCtx domain.TenantAuthContext, linkID string) (domain.BindingPermission, bool) {
	return s.tenantResourcePermission(tenantCtx, "tenant_node_links", "node_link_id", linkID)
}

func (s *MySQLStore) CountNodeLinkBindings(linkID string) int {
	return s.countTenantResourceBindings("tenant_node_links", "node_link_id", linkID)
}
