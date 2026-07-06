package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	"github.com/gorilla/websocket"
)

const relayOpenTimeout = 10 * time.Second

var relayUpgrader = websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}

type directRelayHub struct {
	mu       sync.Mutex
	controls map[string]*directRelayControl
	streams  map[string]*directRelayStream
}

type directRelayControl struct {
	nodeID  string
	conn    *websocket.Conn
	writeMu sync.Mutex
}

type directRelayStream struct {
	id           string
	sourceNodeID string
	targetNodeID string
	source       *websocket.Conn
	target       *websocket.Conn
	ready        chan struct{}
	failed       chan string
	done         chan struct{}
	closeOnce    sync.Once
}

type relayOpenRequest struct {
	Type          string   `json:"type"`
	StreamID      string   `json:"streamId"`
	SourceNodeID  string   `json:"sourceNodeId"`
	TargetHost    string   `json:"targetHost"`
	TargetPort    int      `json:"targetPort"`
	RemainingHops []string `json:"remainingHops"`
}

type relayControlMessage struct {
	Type     string `json:"type"`
	StreamID string `json:"streamId"`
	Message  string `json:"message,omitempty"`
}

type relaySourceAck struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func newDirectRelayHub() *directRelayHub {
	return &directRelayHub{
		controls: make(map[string]*directRelayControl),
		streams:  make(map[string]*directRelayStream),
	}
}

func (r *Router) handleDirectRelayControl(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	nodeID, ok := nodeIDFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_node_token")
		return
	}
	conn, err := relayUpgrader.Upgrade(w, req, nil)
	if err != nil {
		return
	}
	control := r.relay.setControl(nodeID, conn)
	defer r.relay.removeControl(nodeID, control)
	defer conn.Close()
	for {
		var message relayControlMessage
		if err := conn.ReadJSON(&message); err != nil {
			return
		}
		if message.Type == "open_failed" {
			r.relay.failStream(message.StreamID, message.Message)
		}
	}
}

func (r *Router) handleDirectRelaySource(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	sourceNodeID, ok := nodeIDFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_node_token")
		return
	}
	query := req.URL.Query()
	targetNodeID := strings.TrimSpace(query.Get("peerNodeId"))
	targetHost := strings.TrimSpace(query.Get("targetHost"))
	targetPort, _ := strconv.Atoi(query.Get("targetPort"))
	if targetNodeID == "" || targetHost == "" || targetPort <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_relay_source_request")
		return
	}
	conn, err := relayUpgrader.Upgrade(w, req, nil)
	if err != nil {
		return
	}
	stream, err := r.relay.createStream(sourceNodeID, targetNodeID, conn)
	if err != nil {
		_ = conn.WriteJSON(relaySourceAck{Status: domain.TransportStatusFailed, Message: err.Error()})
		_ = conn.Close()
		return
	}
	defer r.relay.closeStream(stream.id)
	control := r.relay.control(targetNodeID)
	if control == nil {
		_ = conn.WriteJSON(relaySourceAck{Status: domain.TransportStatusFailed, Message: "relay_target_control_unavailable"})
		return
	}
	request := relayOpenRequest{
		Type:          "open_stream",
		StreamID:      stream.id,
		SourceNodeID:  sourceNodeID,
		TargetHost:    targetHost,
		TargetPort:    targetPort,
		RemainingHops: splitRelayRemaining(query.Get("remaining")),
	}
	if err := control.writeJSON(request); err != nil {
		_ = conn.WriteJSON(relaySourceAck{Status: domain.TransportStatusFailed, Message: "relay_target_control_write_failed"})
		return
	}
	select {
	case <-stream.ready:
		_ = conn.WriteJSON(relaySourceAck{Status: domain.TransportStatusConnected, Message: "stream_ready"})
		bridgeRelay(req.Context(), newRelayWSConn(stream.source), newRelayWSConn(stream.target))
	case message := <-stream.failed:
		if message == "" {
			message = "relay_target_open_failed"
		}
		_ = conn.WriteJSON(relaySourceAck{Status: domain.TransportStatusFailed, Message: message})
	case <-time.After(relayOpenTimeout):
		_ = conn.WriteJSON(relaySourceAck{Status: domain.TransportStatusFailed, Message: "relay_target_open_timeout"})
	case <-req.Context().Done():
	}
}

func (r *Router) handleDirectRelayTarget(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	nodeID, ok := nodeIDFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_node_token")
		return
	}
	streamID := strings.TrimSpace(req.URL.Query().Get("streamId"))
	if streamID == "" {
		writeError(w, http.StatusBadRequest, "missing_stream_id")
		return
	}
	conn, err := relayUpgrader.Upgrade(w, req, nil)
	if err != nil {
		return
	}
	stream, err := r.relay.attachTarget(streamID, nodeID, conn)
	if err != nil {
		_ = conn.Close()
		return
	}
	select {
	case <-stream.done:
	case <-req.Context().Done():
	}
	_ = conn.Close()
}

func (h *directRelayHub) setControl(nodeID string, conn *websocket.Conn) *directRelayControl {
	control := &directRelayControl{nodeID: nodeID, conn: conn}
	h.mu.Lock()
	previous := h.controls[nodeID]
	h.controls[nodeID] = control
	h.mu.Unlock()
	if previous != nil {
		_ = previous.conn.Close()
	}
	return control
}

func (h *directRelayHub) removeControl(nodeID string, control *directRelayControl) {
	h.mu.Lock()
	if h.controls[nodeID] == control {
		delete(h.controls, nodeID)
	}
	h.mu.Unlock()
}

func (h *directRelayHub) control(nodeID string) *directRelayControl {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.controls[nodeID]
}

func (h *directRelayHub) createStream(sourceNodeID string, targetNodeID string, source *websocket.Conn) (*directRelayStream, error) {
	streamID, err := relayStreamID()
	if err != nil {
		return nil, err
	}
	stream := &directRelayStream{
		id:           streamID,
		sourceNodeID: sourceNodeID,
		targetNodeID: targetNodeID,
		source:       source,
		ready:        make(chan struct{}),
		failed:       make(chan string, 1),
		done:         make(chan struct{}),
	}
	h.mu.Lock()
	h.streams[streamID] = stream
	h.mu.Unlock()
	return stream, nil
}

func (h *directRelayHub) attachTarget(streamID string, nodeID string, target *websocket.Conn) (*directRelayStream, error) {
	h.mu.Lock()
	stream := h.streams[streamID]
	if stream == nil || stream.targetNodeID != nodeID || stream.target != nil {
		h.mu.Unlock()
		return nil, errors.New("relay_stream_not_found")
	}
	stream.target = target
	close(stream.ready)
	h.mu.Unlock()
	return stream, nil
}

func (h *directRelayHub) failStream(streamID string, message string) {
	h.mu.Lock()
	stream := h.streams[streamID]
	h.mu.Unlock()
	if stream == nil {
		return
	}
	select {
	case stream.failed <- message:
	default:
	}
}

func (h *directRelayHub) closeStream(streamID string) {
	h.mu.Lock()
	stream := h.streams[streamID]
	delete(h.streams, streamID)
	h.mu.Unlock()
	if stream == nil {
		return
	}
	stream.closeOnce.Do(func() {
		close(stream.done)
		if stream.source != nil {
			_ = stream.source.Close()
		}
		if stream.target != nil {
			_ = stream.target.Close()
		}
	})
}

func (c *directRelayControl) writeJSON(value any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(relayOpenTimeout))
	err := c.conn.WriteJSON(value)
	_ = c.conn.SetWriteDeadline(time.Time{})
	return err
}

func bridgeRelay(ctx context.Context, left net.Conn, right net.Conn) {
	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(left, right)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(right, left)
		done <- struct{}{}
	}()
	select {
	case <-ctx.Done():
	case <-done:
	}
	_ = left.Close()
	_ = right.Close()
}

func splitRelayRemaining(raw string) []string {
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func relayStreamID() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
