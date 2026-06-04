package policystore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
)

type Snapshot struct {
	Nodes      []domain.Node      `json:"nodes"`
	Links      []domain.NodeLink  `json:"links"`
	Chains     []domain.Chain     `json:"chains"`
	RouteRules []domain.RouteRule `json:"routeRules"`
}

type tenantPayload struct {
	Snapshots []struct {
		TenantID string   `json:"tenantId"`
		Payload  Snapshot `json:"payload"`
	} `json:"snapshots"`
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
	snapshot, err := decodeSnapshot(payload)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revision = revision
	s.snapshot = snapshot
	return s.persist()
}

func decodeSnapshot(payload string) (Snapshot, error) {
	var wrapped tenantPayload
	if err := json.Unmarshal([]byte(payload), &wrapped); err != nil {
		return Snapshot{}, err
	}
	if len(wrapped.Snapshots) > 0 {
		return mergeSnapshots(wrapped.Snapshots), nil
	}
	var snapshot Snapshot
	if err := json.Unmarshal([]byte(payload), &snapshot); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func mergeSnapshots(items []struct {
	TenantID string   `json:"tenantId"`
	Payload  Snapshot `json:"payload"`
}) Snapshot {
	merged := Snapshot{}
	for _, item := range items {
		for index := range item.Payload.Chains {
			if item.Payload.Chains[index].TenantID == "" {
				item.Payload.Chains[index].TenantID = item.TenantID
			}
		}
		for index := range item.Payload.RouteRules {
			if item.Payload.RouteRules[index].TenantID == "" {
				item.Payload.RouteRules[index].TenantID = item.TenantID
			}
		}
		merged.Nodes = append(merged.Nodes, item.Payload.Nodes...)
		merged.Links = append(merged.Links, item.Payload.Links...)
		merged.Chains = append(merged.Chains, item.Payload.Chains...)
		merged.RouteRules = append(merged.RouteRules, item.Payload.RouteRules...)
	}
	return merged
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
