package service

import (
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (c *ControlPlane) NodeHealth() []domain.NodeHealth {
	return c.store.ListNodeHealth()
}

func (c *ControlPlane) NodeHealthHistory(nodeID string, window time.Duration) ([]domain.NodeHealth, error) {
	if nodeID == "" {
		return nil, invalidInput("missing_node_id")
	}
	if window <= 0 || window > 7*24*time.Hour {
		window = 24 * time.Hour
	}
	return c.store.ListNodeHealthHistory(nodeID, window)
}

func (c *ControlPlane) UpsertNodeHeartbeat(input domain.NodeHeartbeatInput) (domain.NodeHealth, error) {
	if input.NodeID == "" {
		return domain.NodeHealth{}, invalidInput("missing_node_id")
	}
	item, err := c.store.UpsertNodeHeartbeat(input)
	item.ProxyTokenCacheTTLSeconds = int(c.proxyTokenCacheTTL.Seconds())
	return item, err
}

func (c *ControlPlane) RenewNodeCertificate(input domain.NodeCertRenewInput) (domain.NodeCertRenewResult, error) {
	if input.NodeID == "" || input.CertType == "" {
		return domain.NodeCertRenewResult{}, invalidInput("invalid_cert_renew_payload")
	}
	return c.store.RenewNodeCertificate(input)
}
