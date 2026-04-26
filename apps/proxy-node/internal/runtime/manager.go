package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/controlplane"
	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/policystore"
)

type Binding struct {
	ControlPlaneURL string `json:"controlPlaneUrl"`
	NodeID          string `json:"nodeId"`
	NodeAccessToken string `json:"nodeAccessToken"`
	NodeName        string `json:"nodeName"`
	NodeMode        string `json:"nodeMode"`
	NodeScopeKey    string `json:"nodeScopeKey"`
	NodeParentID    string `json:"nodeParentId"`
	NodePublicHost  string `json:"nodePublicHost"`
	NodePublicPort  int    `json:"nodePublicPort"`
}

type Manager struct {
	mu               sync.RWMutex
	path             string
	store            *policystore.Store
	interval         time.Duration
	listenerStatus   map[string]string
	certStatus       map[string]string
	managePublicCert bool
	binding          Binding
}

func New(path string, store *policystore.Store, interval time.Duration, listenerStatus map[string]string, certStatus map[string]string, managePublicCert bool) *Manager {
	manager := &Manager{
		path:             path,
		store:            store,
		interval:         interval,
		listenerStatus:   cloneMap(listenerStatus),
		certStatus:       cloneMap(certStatus),
		managePublicCert: managePublicCert,
	}
	manager.load()
	return manager
}

func (m *Manager) Run() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	m.tick()
	for range ticker.C {
		m.tick()
	}
}

func (m *Manager) Attach(binding Binding) error {
	m.mu.Lock()
	m.binding = binding
	m.mu.Unlock()
	if err := m.persist(); err != nil {
		return err
	}
	m.tick()
	return nil
}

func (m *Manager) Current() Binding {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.binding
}

func (m *Manager) Bound() bool {
	current := m.Current()
	return current.ControlPlaneURL != "" && current.NodeID != "" && current.NodeAccessToken != ""
}

func (m *Manager) NodeID() string {
	return m.Current().NodeID
}

func (m *Manager) tick() {
	current := m.Current()
	if current.ControlPlaneURL == "" || current.NodeID == "" || current.NodeAccessToken == "" {
		return
	}
	client := controlplane.New(current.ControlPlaneURL, current.NodeAccessToken)
	policy, err := client.FetchPolicy()
	if err != nil {
		return
	}
	if err := m.store.Update(policy.PolicyRevisionID, policy.PayloadJSON); err != nil {
		return
	}
	if m.managePublicCert {
		result, renewErr := client.RenewCertificate("public")
		if renewErr != nil {
			m.setCertStatus("public", "degraded")
		} else {
			m.setCertStatus("public", result.Status)
		}
	}
	revision, _ := m.store.Current()
	_, _ = client.SendHeartbeat(revision, cloneMap(m.listenerStatus), cloneMap(m.certStatus))
}

func (m *Manager) setCertStatus(key string, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.certStatus[key] = value
}

func (m *Manager) load() {
	if m.path == "" {
		return
	}
	raw, err := os.ReadFile(m.path)
	if err != nil {
		return
	}
	var binding Binding
	if err := json.Unmarshal(raw, &binding); err != nil {
		return
	}
	m.binding = binding
}

func (m *Manager) persist() error {
	if m.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return err
	}
	raw, err := json.Marshal(m.Current())
	if err != nil {
		return err
	}
	return os.WriteFile(m.path, raw, 0o600)
}

func cloneMap(value map[string]string) map[string]string {
	cloned := make(map[string]string, len(value))
	for key, item := range value {
		cloned[key] = item
	}
	return cloned
}
