package store

import (
	"time"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/auth"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func (s *MySQLStore) CreateBootstrapToken(input domain.CreateBootstrapTokenInput) (domain.BootstrapToken, error) {
	token, err := auth.RandomToken()
	if err != nil {
		return domain.BootstrapToken{}, err
	}
	tokenID, err := s.nextID("bootstrap_token")
	if err != nil {
		return domain.BootstrapToken{}, err
	}
	item := domain.BootstrapToken{
		ID:           tokenID,
		Token:        token,
		TargetType:   input.TargetType,
		TargetID:     input.TargetID,
		NodeName:     input.NodeName,
		NodeMode:     input.NodeMode,
		ScopeKey:     input.ScopeKey,
		ParentNodeID: input.ParentNodeID,
		PublicHost:   input.PublicHost,
		PublicPort:   input.PublicPort,
		ExpiresAt:    time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339),
		CreatedAt:    nowRFC3339(),
	}
	_, err = s.db.Exec(
		`INSERT INTO bootstrap_tokens (id, token_hash, target_type, target_id, node_name, node_mode, scope_key, parent_node_id, public_host, public_port, expires_at, consumed_at, created_at)
		 VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?, NULL, ?)`,
		item.ID, auth.TokenHash(token), item.TargetType, item.TargetID, item.NodeName, item.NodeMode, item.ScopeKey, item.ParentNodeID, item.PublicHost, item.PublicPort, item.ExpiresAt, nowRFC3339(),
	)
	return item, err
}

func (s *MySQLStore) CreateBootstrapTokenForTenant(tenantCtx domain.TenantAuthContext, input domain.CreateBootstrapTokenInput) (domain.BootstrapToken, error) {
	token, err := auth.RandomToken()
	if err != nil {
		return domain.BootstrapToken{}, err
	}
	tokenID, err := s.nextID("bootstrap_token")
	if err != nil {
		return domain.BootstrapToken{}, err
	}
	now := nowRFC3339()
	item := domain.BootstrapToken{
		ID:           tokenID,
		Token:        token,
		TargetType:   input.TargetType,
		TargetID:     input.TargetID,
		NodeName:     input.NodeName,
		NodeMode:     input.NodeMode,
		ScopeKey:     input.ScopeKey,
		ParentNodeID: input.ParentNodeID,
		PublicHost:   input.PublicHost,
		PublicPort:   input.PublicPort,
		ExpiresAt:    time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339),
		CreatedAt:    now,
	}
	tx, err := s.db.Begin()
	if err != nil {
		return domain.BootstrapToken{}, err
	}
	defer tx.Rollback()
	if item.TargetID == "" {
		nodeID, err := s.nextNodeID()
		if err != nil {
			return domain.BootstrapToken{}, err
		}
		item.TargetID = nodeID
		if _, err := tx.Exec(
			`INSERT INTO nodes (id, create_id, owner_id, name, mode, public_host, public_port, scope_key, parent_node_id, enabled, status, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), ?, ?, ?, ?)`,
			nodeID, tenantCtx.Account.ID, tenantCtx.Account.ID, item.NodeName, item.NodeMode, item.PublicHost, item.PublicPort, item.ScopeKey, item.ParentNodeID, 1, domain.NodeStatusPending, now, now,
		); err != nil {
			return domain.BootstrapToken{}, err
		}
		if tenantCtx.ActiveTenant.TenantID != "" {
			if err := bindTenantResource(tx, "tenant_nodes", "node_id", tenantCtx.ActiveTenant.TenantID, nodeID, tenantCtx.Account.ID); err != nil {
				return domain.BootstrapToken{}, err
			}
		}
	}
	if _, err := tx.Exec(
		`INSERT INTO bootstrap_tokens (id, token_hash, target_type, target_id, node_name, node_mode, scope_key, parent_node_id, public_host, public_port, expires_at, consumed_at, created_at)
		 VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?, NULL, ?)`,
		item.ID, auth.TokenHash(token), item.TargetType, item.TargetID, item.NodeName, item.NodeMode, item.ScopeKey, item.ParentNodeID, item.PublicHost, item.PublicPort, item.ExpiresAt, item.CreatedAt,
	); err != nil {
		return domain.BootstrapToken{}, err
	}
	return item, tx.Commit()
}

func (s *MySQLStore) ListUnconsumedBootstrapTokens() []domain.BootstrapToken {
	rows, err := s.db.Query(
		`SELECT id, target_type, COALESCE(target_id, ''), COALESCE(node_name, ''), COALESCE(node_mode, ''), COALESCE(scope_key, ''), COALESCE(parent_node_id, ''), COALESCE(public_host, ''), COALESCE(public_port, 0), expires_at, created_at
		 FROM bootstrap_tokens
		 WHERE consumed_at IS NULL AND expires_at > ?
		 ORDER BY created_at DESC`,
		nowRFC3339(),
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.BootstrapToken, 0)
	for rows.Next() {
		var item domain.BootstrapToken
		if err := rows.Scan(&item.ID, &item.TargetType, &item.TargetID, &item.NodeName, &item.NodeMode, &item.ScopeKey, &item.ParentNodeID, &item.PublicHost, &item.PublicPort, &item.ExpiresAt, &item.CreatedAt); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) DeleteBootstrapToken(tokenID string) error {
	_, err := s.db.Exec(
		`DELETE FROM bootstrap_tokens
		 WHERE id = ? AND consumed_at IS NULL`,
		tokenID,
	)
	return err
}
