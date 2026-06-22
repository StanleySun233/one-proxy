package tunnel

import (
	"errors"
	"log"
	"net"
	"sync"
	"time"

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
	previous := r.sessions[nodeID]
	r.sessions[nodeID] = session
	r.mu.Unlock()
	if previous != nil {
		previous.close()
	}
	return session
}

func (r *Registry) Remove(nodeID string, session *childSession) {
	removed := false
	r.mu.Lock()
	if current, ok := r.sessions[nodeID]; ok && current == session {
		delete(r.sessions, nodeID)
		removed = true
	}
	r.mu.Unlock()
	if removed {
		session.close()
	}
}

func (r *Registry) HasChild(nodeID string) bool {
	r.mu.RLock()
	_, ok := r.sessions[nodeID]
	r.mu.RUnlock()
	return ok
}

func (r *Registry) ForwardProbe(fromNodeID string, nextNodeID string, requestID string, remaining []string, protocol string, targetHost string, targetPort int) (Message, error) {
	r.mu.RLock()
	session, ok := r.sessions[nextNodeID]
	r.mu.RUnlock()
	if !ok {
		return Message{}, errors.New("child_tunnel_not_found")
	}
	started := time.Now()
	response, err := session.request(Message{
		Type:                "probe_request",
		RequestID:           requestID,
		RemainingHopNodeIDs: remaining,
		Protocol:            protocol,
		TargetHost:          targetHost,
		TargetPort:          targetPort,
	})
	if err != nil {
		return Message{}, err
	}
	ended := time.Now().UTC()
	response.PathTimings = append([]PathTiming{{
		FromNodeID:  fromNodeID,
		ToNodeID:    nextNodeID,
		RoundTripMs: ended.Sub(started).Milliseconds(),
		SampleTSMs:  ended.UnixMilli(),
		Count:       1,
	}}, response.PathTimings...)
	return response, nil
}

func (r *Registry) OpenStream(nextNodeID string, remaining []string, targetHost string, targetPort int) (net.Conn, error) {
	r.mu.RLock()
	session, ok := r.sessions[nextNodeID]
	r.mu.RUnlock()
	if !ok {
		log.Printf("node tunnel stream_open_failed nextNodeID=%s target=%s:%d err=child_tunnel_not_found", nextNodeID, targetHost, targetPort)
		return nil, errors.New("child_tunnel_not_found")
	}
	started := time.Now()
	conn, err := session.openStream(remaining, targetHost, targetPort)
	if err != nil {
		log.Printf("node tunnel stream_open_failed nextNodeID=%s remaining=%v target=%s:%d duration=%s err=%v", nextNodeID, remaining, targetHost, targetPort, time.Since(started), err)
		if errors.Is(err, errStreamOpenTimeout) || errors.Is(err, errChildTunnelClosed) {
			r.Remove(nextNodeID, session)
		}
	}
	return conn, err
}
