package policystore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/domain"
)

type Snapshot struct {
	Nodes      []domain.Node      `json:"nodes"`
	Links      []domain.NodeLink  `json:"links"`
	Chains     []domain.Chain     `json:"chains"`
	RouteRules []domain.RouteRule `json:"routeRules"`
}

type Store struct {
	mu       sync.RWMutex
	path     string
	revision string
	snapshot Snapshot
}

type persistedState struct {
	Revision string   `json:"revision"`
	Snapshot Snapshot `json:"snapshot"`
}

func New(path string) *Store {
	store := &Store{path: path}
	store.load()
	return store
}

func (s *Store) Update(revision string, payload string) error {
	var snapshot Snapshot
	if err := json.Unmarshal([]byte(payload), &snapshot); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revision = revision
	s.snapshot = snapshot
	return s.persist()
}

func (s *Store) Current() (string, Snapshot) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.revision, s.snapshot
}

func (s *Store) load() {
	if s.path == "" {
		return
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var state persistedState
	if err := json.Unmarshal(raw, &state); err != nil {
		return
	}
	s.revision = state.Revision
	s.snapshot = state.Snapshot
}

func (s *Store) persist() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	raw, err := json.Marshal(persistedState{
		Revision: s.revision,
		Snapshot: s.snapshot,
	})
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o600)
}
