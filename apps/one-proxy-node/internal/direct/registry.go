package direct

import (
	"errors"
	"sync"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
)

type PeerState struct {
	LinkID            string
	PeerNodeID        string
	Status            string
	SelectedCandidate domain.DirectCandidate
	RTT               time.Duration
	LastProbeAt       time.Time
	FallbackReason    string
}

type Registry struct {
	mu    sync.RWMutex
	peers map[string]PeerState
}

func NewRegistry() *Registry {
	return &Registry{peers: make(map[string]PeerState)}
}

func (r *Registry) Upsert(state PeerState) {
	r.mu.Lock()
	r.peers[state.PeerNodeID] = state
	r.mu.Unlock()
}

func (r *Registry) Get(peerNodeID string) (PeerState, bool) {
	r.mu.RLock()
	state, ok := r.peers[peerNodeID]
	r.mu.RUnlock()
	return state, ok
}

func (r *Registry) Remove(peerNodeID string) {
	r.mu.Lock()
	delete(r.peers, peerNodeID)
	r.mu.Unlock()
}

func (r *Registry) OpenStream(peerNodeID string) error {
	state, ok := r.Get(peerNodeID)
	if !ok {
		return errors.New("direct_peer_not_found")
	}
	if state.Status != domain.DirectStatusConnected {
		return errors.New("direct_peer_not_connected")
	}
	return errors.New("direct_quic_not_ready")
}
