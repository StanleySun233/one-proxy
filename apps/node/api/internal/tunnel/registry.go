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
	waiters  map[string][]chan struct{}
}

func NewRegistry() *Registry {
	return &Registry{
		sessions: make(map[string]*childSession),
		waiters:  make(map[string][]chan struct{}),
	}
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
	waiters := r.waiters[nodeID]
	delete(r.waiters, nodeID)
	r.mu.Unlock()
	if previous != nil {
		previous.close()
	}
	for _, waiter := range waiters {
		close(waiter)
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
		RoundTripMs: roundTripMs(ended.Sub(started)),
		SampleTSMs:  ended.UnixMilli(),
		Count:       1,
	}}, response.PathTimings...)
	return response, nil
}

func roundTripMs(elapsed time.Duration) int64 {
	if elapsed <= 0 {
		return 0
	}
	ms := elapsed.Milliseconds()
	if ms == 0 {
		return 1
	}
	return ms
}

func (r *Registry) OpenStream(nextNodeID string, remaining []string, targetHost string, targetPort int) (net.Conn, error) {
	for attempt := 0; attempt < 2; attempt++ {
		session, ok := r.session(nextNodeID)
		if !ok {
			waited, waitErr := r.waitForSession(nextNodeID, streamReconnectWaitTimeout)
			if waitErr != nil {
				log.Printf("node tunnel stream_open_failed nextNodeID=%s target=%s:%d err=child_tunnel_not_found", nextNodeID, targetHost, targetPort)
				return nil, errors.New("child_tunnel_not_found")
			}
			session = waited
		}
		started := time.Now()
		conn, err := session.openStream(remaining, targetHost, targetPort)
		if err == nil {
			return conn, nil
		}
		log.Printf("node tunnel stream_open_failed nextNodeID=%s remaining=%v target=%s:%d duration=%s err=%v", nextNodeID, remaining, targetHost, targetPort, time.Since(started), err)
		if errors.Is(err, errStreamOpenTimeout) || errors.Is(err, errChildTunnelClosed) {
			r.Remove(nextNodeID, session)
			if attempt == 0 {
				if _, waitErr := r.waitForSession(nextNodeID, streamReconnectWaitTimeout); waitErr == nil {
					continue
				}
			}
		}
		return nil, err
	}
	return nil, errors.New("child_tunnel_not_found")
}

func (r *Registry) session(nodeID string) (*childSession, bool) {
	r.mu.RLock()
	session, ok := r.sessions[nodeID]
	r.mu.RUnlock()
	return session, ok
}

func (r *Registry) waitForSession(nodeID string, timeout time.Duration) (*childSession, error) {
	if timeout <= 0 {
		return nil, errors.New("child_tunnel_not_found")
	}
	waiter := make(chan struct{})
	r.mu.Lock()
	if session, ok := r.sessions[nodeID]; ok {
		r.mu.Unlock()
		return session, nil
	}
	r.waiters[nodeID] = append(r.waiters[nodeID], waiter)
	r.mu.Unlock()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-waiter:
		session, ok := r.session(nodeID)
		if ok {
			return session, nil
		}
		return nil, errors.New("child_tunnel_not_found")
	case <-timer.C:
		r.removeWaiter(nodeID, waiter)
		return nil, errors.New("child_tunnel_not_found")
	}
}

func (r *Registry) removeWaiter(nodeID string, waiter chan struct{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	waiters := r.waiters[nodeID]
	for index, item := range waiters {
		if item == waiter {
			waiters = append(waiters[:index], waiters[index+1:]...)
			break
		}
	}
	if len(waiters) == 0 {
		delete(r.waiters, nodeID)
		return
	}
	r.waiters[nodeID] = waiters
}
