package store

import (
	"time"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/auth"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func (s *MySQLStore) AuthenticateNodeToken(accessToken string) (string, bool) {
	var (
		nodeID    string
		expiresAt string
		status    string
		enabled   int
	)
	err := s.db.QueryRow(
		`SELECT t.node_id, t.expires_at, n.status, n.enabled
		 FROM node_api_tokens t
		 JOIN nodes n ON n.id = t.node_id
		 WHERE t.token_hash = ?`,
		auth.TokenHash(accessToken),
	).Scan(&nodeID, &expiresAt, &status, &enabled)
	if err != nil {
		return "", false
	}
	expiry, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil || time.Now().UTC().After(expiry) || enabled != 1 || status == domain.NodeStatusPending {
		return "", false
	}
	return nodeID, true
}
