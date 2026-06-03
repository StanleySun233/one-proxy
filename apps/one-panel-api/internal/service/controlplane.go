package service

import (
	"log"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/config"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
	proxyservice "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/features/proxy/service"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/proxytoken"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/store"
)

type ControlPlane struct {
	store              store.Store
	proxyTokens        proxytoken.Store
	sessionTTL         time.Duration
	proxyTokenCacheTTL time.Duration
	bootstrapTokenTTL  time.Duration
	nodeHeartbeatTTL   time.Duration
	publicRenewWindow  time.Duration
	enumsByField       map[string]map[string]domain.FieldEnum
	proxy              *proxyservice.Service
}

func NewControlPlane(store store.Store, cfg config.Config) *ControlPlane {
	proxyTokens := proxytoken.Store(proxytoken.NewMemoryStore())
	if cfg.RedisURL != "" {
		if redisStore, err := proxytoken.NewRedisStore(cfg.RedisURL); err == nil {
			proxyTokens = redisStore
		} else {
			log.Printf("warn: redis proxy token store unavailable: %v", err)
		}
	}
	controlPlane := &ControlPlane{
		store:              store,
		proxyTokens:        proxyTokens,
		sessionTTL:         parseDuration(cfg.SessionTTL, 30*24*time.Hour),
		proxyTokenCacheTTL: parseDuration(cfg.ProxyTokenCacheTTL, 24*time.Hour),
		bootstrapTokenTTL:  parseDuration(cfg.BootstrapTokenTTL, 15*time.Minute),
		nodeHeartbeatTTL:   parseDuration(cfg.NodeHeartbeatTTL, 2*time.Minute),
		publicRenewWindow:  parseDuration(cfg.PublicCertRenewWindow, 7*24*time.Hour),
	}
	controlPlane.proxy = proxyservice.New(store)
	return controlPlane
}

func (c *ControlPlane) IsInitialized() bool {
	return c.store.IsInitialized()
}

func (c *ControlPlane) ReinitializeStore(adminPassword string) error {
	return c.store.ReinitializeStore(adminPassword)
}

func (c *ControlPlane) Proxy() *proxyservice.Service {
	return c.proxy
}

func (c *ControlPlane) ScopeExists(scopeID string) bool {
	return c.proxy.ScopeExists(scopeID)
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
