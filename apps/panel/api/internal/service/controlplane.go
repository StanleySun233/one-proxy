package service

import (
	"log"
	"sync"
	"time"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/config"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	proxyservice "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/service"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/proxytoken"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/sla"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/store"
)

type ControlPlane struct {
	store              store.Store
	proxyTokens        proxytoken.Store
	slaHeartbeats      sla.HeartbeatStore
	sessionTTL         time.Duration
	proxyTokenCacheTTL time.Duration
	bootstrapTokenTTL  time.Duration
	nodeHeartbeatTTL   time.Duration
	publicRenewWindow  time.Duration
	enumsByField       map[string]map[string]domain.FieldEnum
	proxy              *proxyservice.Service
	proxyStatus        *proxyStatusStore
	clientDirectMu     sync.Mutex
	clientDirect       map[string]clientDirectSessionRecord
	guacdAddr          string
	remoteMu           sync.Mutex
	remoteSessions     map[string]remoteSessionRecord
}

func NewControlPlane(store store.Store, cfg config.Config) *ControlPlane {
	proxyTokens := proxytoken.Store(proxytoken.NewMemoryStore())
	slaHeartbeats := sla.HeartbeatStore(sla.NewMemoryHeartbeatStore(2*time.Hour, slaHeartbeatInterval))
	if cfg.RedisURL != "" {
		if redisStore, err := proxytoken.NewRedisStore(cfg.RedisURL); err == nil {
			proxyTokens = redisStore
		} else {
			log.Printf("warn: redis proxy token store unavailable: %v", err)
		}
		if redisHeartbeatStore, err := sla.NewRedisHeartbeatStore(cfg.RedisURL, 2*time.Hour, slaHeartbeatInterval); err == nil {
			slaHeartbeats = redisHeartbeatStore
		} else {
			log.Printf("warn: redis sla heartbeat store unavailable: %v", err)
		}
	}
	controlPlane := &ControlPlane{
		store:              store,
		proxyTokens:        proxyTokens,
		slaHeartbeats:      slaHeartbeats,
		sessionTTL:         parseDuration(cfg.SessionTTL, 30*24*time.Hour),
		proxyTokenCacheTTL: parseDuration(cfg.ProxyTokenCacheTTL, 24*time.Hour),
		bootstrapTokenTTL:  parseDuration(cfg.BootstrapTokenTTL, 15*time.Minute),
		nodeHeartbeatTTL:   parseDuration(cfg.NodeHeartbeatTTL, 2*time.Minute),
		publicRenewWindow:  parseDuration(cfg.PublicCertRenewWindow, 7*24*time.Hour),
		proxyStatus:        newProxyStatusStore(5000),
		clientDirect:       make(map[string]clientDirectSessionRecord),
		guacdAddr:          cfg.GuacdAddr,
		remoteSessions:     make(map[string]remoteSessionRecord),
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
	if err := c.RecordNodeSLAMinute(); err != nil {
		return err
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
