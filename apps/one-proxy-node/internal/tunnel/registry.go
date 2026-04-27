package tunnel

import (
	"errors"
	"net"
	"sync"

	"github.com/gorilla/websocket"
)

type Registry struct {
	mu       sync.RWMutex
	sessions map[string]*childSession
}

func NewRegistry() *Registry {
	return &Registry{sessions: make(map[string]*childSession)}
}

func (r *Registry) Add(nodeID string, conn *websocket.Conn) *childSession {
	session := &childSession{
		nodeID:  nodeID,
		conn:    conn,
		pending: make(map[string]chan Message),
		streams: make(map[string]*streamConn),
		done:    make(chan struct{}),
	}
	r.mu.Lock()
	r.sessions[nodeID] = session
	r.mu.Unlock()
	return session
}

func (r *Registry) Remove(nodeID string, session *childSession) {
	r.mu.Lock()
	if current, ok := r.sessions[nodeID]; ok && current == session {
		delete(r.sessions, nodeID)
	}
	r.mu.Unlock()
	session.closeAll()
	close(session.done)
}

func (r *Registry) HasChild(nodeID string) bool {
	r.mu.RLock()
	_, ok := r.sessions[nodeID]
	r.mu.RUnlock()
	return ok
}

func (r *Registry) ForwardProbe(nextNodeID string, requestID string, remaining []string, targetHost string, targetPort int) (Message, error) {
	r.mu.RLock()
	session, ok := r.sessions[nextNodeID]
	r.mu.RUnlock()
	if !ok {
		return Message{}, errors.New("child_tunnel_not_found")
	}
	return session.request(Message{
		Type:                "probe_request",
		RequestID:           requestID,
		RemainingHopNodeIDs: remaining,
		TargetHost:          targetHost,
		TargetPort:          targetPort,
	})
}

func (r *Registry) OpenStream(nextNodeID string, remaining []string, targetHost string, targetPort int) (net.Conn, error) {
	r.mu.RLock()
	session, ok := r.sessions[nextNodeID]
	r.mu.RUnlock()
	if !ok {
		return nil, errors.New("child_tunnel_not_found")
	}
	return session.openStream(remaining, targetHost, targetPort)
}
