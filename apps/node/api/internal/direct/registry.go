package direct

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
	"github.com/quic-go/quic-go"
)

type PeerState struct {
	LinkID            string
	PeerNodeID        string
	Status            string
	SelectedCandidate domain.DirectCandidate
	PeerIdentity      domain.DirectNodeIdentity
	RTT               time.Duration
	LastProbeAt       time.Time
	FallbackReason    string
}

type ClientSessionValidationRequest struct {
	SessionID  string
	PunchToken string
	TargetHost string
	TargetPort int
}

type ClientSessionValidationResult struct {
	Valid      bool
	TargetHost string
	TargetPort int
}

type ClientSessionValidator interface {
	ValidateClientDirectSession(context.Context, ClientSessionValidationRequest) (ClientSessionValidationResult, error)
}

type Registry struct {
	mu              sync.RWMutex
	peers           map[string]PeerState
	transport       *quic.Transport
	listener        *quic.Listener
	directIdentity  domain.DirectNodeIdentity
	clientValidator ClientSessionValidator
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

func (r *Registry) SetClientSessionValidator(validator ClientSessionValidator) {
	r.mu.Lock()
	r.clientValidator = validator
	r.mu.Unlock()
}

func (r *Registry) DirectIdentity() domain.DirectNodeIdentity {
	r.mu.RLock()
	identity := r.directIdentity
	r.mu.RUnlock()
	return identity
}

func (r *Registry) HasDirectPeer(peerNodeID string) bool {
	state, ok := r.Get(peerNodeID)
	return ok && state.Status == domain.DirectStatusConnected
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
