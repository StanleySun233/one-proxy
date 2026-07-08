package runtime

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/controlplane"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/policystore"
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

type persistedState struct {
	Binding
	JoinPassword       string `json:"joinPassword"`
	MustRotatePassword bool   `json:"mustRotatePassword"`
}

type Manager struct {
	mu                 sync.RWMutex
	path               string
	store              *policystore.Store
	interval           time.Duration
	listenerStatus     map[string]string
	certStatus         map[string]string
	managePublicCert   bool
	binding            Binding
	joinPassword       string
	mustRotatePassword bool
}

func New(path string, store *policystore.Store, interval time.Duration, listenerStatus map[string]string, certStatus map[string]string, managePublicCert bool, joinPassword string, mustRotatePassword bool) *Manager {
	manager := &Manager{
		path:               path,
		store:              store,
		interval:           interval,
		listenerStatus:     cloneMap(listenerStatus),
		certStatus:         cloneMap(certStatus),
		managePublicCert:   managePublicCert,
		joinPassword:       joinPassword,
		mustRotatePassword: mustRotatePassword,
	}
	manager.load()
	return manager
}

func (m *Manager) Run() {
	for {
		next := nextTick(time.Now(), m.interval)
		time.Sleep(time.Until(next))
		m.tick(next.UTC())
	}
}

func (m *Manager) Attach(binding Binding) error {
	m.mu.Lock()
	m.binding = binding
	m.mu.Unlock()
	if err := m.persist(); err != nil {
		return err
	}
	m.tick(time.Now().UTC())
	return nil
}

func (m *Manager) Current() Binding {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.binding
}

func (m *Manager) JoinPassword() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.joinPassword
}

func (m *Manager) MustRotatePassword() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mustRotatePassword
}

func (m *Manager) Bound() bool {
	current := m.Current()
	return current.ControlPlaneURL != "" && current.NodeID != "" && current.NodeAccessToken != ""
}

func (m *Manager) NodeID() string {
	return m.Current().NodeID
}

func (m *Manager) RotateJoinPassword(currentPassword string, newPassword string) error {
	m.mu.Lock()
	if currentPassword == "" || currentPassword != m.joinPassword {
		m.mu.Unlock()
		return os.ErrPermission
	}
	if newPassword == "" || newPassword == currentPassword {
		m.mu.Unlock()
		return os.ErrInvalid
	}
	m.joinPassword = newPassword
	m.mustRotatePassword = false
	m.mu.Unlock()
	return m.persist()
}

func (m *Manager) tick(heartbeatAt time.Time) {
	current := m.Current()
	if current.ControlPlaneURL == "" || current.NodeID == "" || current.NodeAccessToken == "" {
		return
	}
	client := controlplane.New(current.ControlPlaneURL, current.NodeAccessToken)
	policy, err := client.FetchPolicy()
	if err == nil {
		if err := m.store.Update(policy.PolicyRevisionID, policy.PayloadJSON); err != nil {
			log.Printf("runtime policy update failed nodeID=%s revision=%s err=%v", current.NodeID, policy.PolicyRevisionID, err)
			return
		}
	} else {
		log.Printf("runtime policy fetch failed nodeID=%s err=%v", current.NodeID, err)
	}
	if m.managePublicCert {
		result, renewErr := client.RenewCertificate("public")
		if renewErr != nil {
			m.setCertStatus("public", domain.CertStatusDegraded)
			log.Printf("runtime cert renew failed nodeID=%s cert=public err=%v", current.NodeID, renewErr)
		} else {
			m.setCertStatus("public", result.Status)
		}
	}
	revision, _ := m.store.Current()
	listenerStatus, certStatus := m.healthStatus()
	if _, err := client.SendHeartbeat(heartbeatAt, revision, listenerStatus, certStatus); err != nil {
		log.Printf("runtime heartbeat failed nodeID=%s revision=%s err=%v", current.NodeID, revision, err)
	}
}

func (m *Manager) SetListenerStatus(key string, value string) {
	if key == "" || value == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listenerStatus[key] = value
}

func (m *Manager) setCertStatus(key string, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.certStatus[key] = value
}

func (m *Manager) healthStatus() (map[string]string, map[string]string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return cloneMap(m.listenerStatus), cloneMap(m.certStatus)
}

func (m *Manager) load() {
	if m.path == "" {
		return
	}
	raw, err := os.ReadFile(m.path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("runtime state load failed path=%s err=%v", m.path, err)
		}
		return
	}
	var state persistedState
	if err := json.Unmarshal(raw, &state); err != nil {
		log.Printf("runtime state parse failed path=%s err=%v", m.path, err)
		return
	}
	m.binding = state.Binding
	if state.JoinPassword != "" {
		m.joinPassword = state.JoinPassword
		m.mustRotatePassword = state.MustRotatePassword
	}
}

func (m *Manager) persist() error {
	if m.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return err
	}
	m.mu.RLock()
	state := persistedState{
		Binding:            m.binding,
		JoinPassword:       m.joinPassword,
		MustRotatePassword: m.mustRotatePassword,
	}
	m.mu.RUnlock()
	raw, err := json.Marshal(state)
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

func nextTick(now time.Time, interval time.Duration) time.Time {
	return now.Truncate(interval).Add(interval)
}
