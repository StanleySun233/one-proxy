package service

import (
	"log"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/config"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/store"
)

type ControlPlane struct {
	store             store.Store
	sessionTTL        time.Duration
	bootstrapTokenTTL time.Duration
	nodeHeartbeatTTL  time.Duration
	publicRenewWindow time.Duration
	enumsByField      map[string]map[string]domain.FieldEnum
}

func NewControlPlane(store store.Store, cfg config.Config) *ControlPlane {
	return &ControlPlane{
		store:             store,
		sessionTTL:        parseDuration(cfg.SessionTTL, 12*time.Hour),
		bootstrapTokenTTL: parseDuration(cfg.BootstrapTokenTTL, 15*time.Minute),
		nodeHeartbeatTTL:  parseDuration(cfg.NodeHeartbeatTTL, 2*time.Minute),
		publicRenewWindow: parseDuration(cfg.PublicCertRenewWindow, 7*24*time.Hour),
	}
}

func (c *ControlPlane) IsInitialized() bool {
	return c.store.IsInitialized()
}

func (c *ControlPlane) ReinitializeStore(adminPassword string) error {
	return c.store.ReinitializeStore(adminPassword)
}

func (c *ControlPlane) RunMaintenance() error {
	if _, err := c.store.CleanupExpiredSessions(); err != nil {
		return err
	}
	if _, err := c.store.CleanupExpiredBootstrapTokens(); err != nil {
		return err
	}
	if _, err := c.store.CleanupExpiredNodeTokens(); err != nil {
		return err
	}
	if err := c.store.RefreshCertificateStatus(c.publicRenewWindow); err != nil {
		return err
	}
	for _, cert := range c.store.ListCertificates() {
		if cert.OwnerType != "node" || cert.CertType != domain.CertTypePublic {
			continue
		}
		if cert.Status != domain.CertStatusRenewSoon && cert.Status != domain.CertStatusExpired {
			continue
		}
		if _, err := c.store.RenewNodeCertificate(domain.NodeCertRenewInput{
			NodeID:   cert.OwnerID,
			CertType: cert.CertType,
		}); err != nil {
			return err
		}
	}
	if err := c.store.RefreshNodeStatus(c.nodeHeartbeatTTL); err != nil {
		return err
	}
	if removed, err := c.store.CleanupNodeHealthHistory(7 * 24 * time.Hour); err != nil {
		log.Printf("maintenance: failed to cleanup node health history: %v", err)
	} else if removed > 0 {
		log.Printf("maintenance: cleaned up %d stale health history rows", removed)
	}
	return nil
}

func parseDuration(raw string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func uniqueStrings(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}
