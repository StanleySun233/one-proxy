package tunnel

import (
	"encoding/base64"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/controlplane"
	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/proxy-node/internal/runtime"
	"github.com/gorilla/websocket"
)

type Controller struct {
	manager           *runtime.Manager
	registry          *Registry
	parentTunnelURL   string
	tunnelPath        string
	heartbeatInterval time.Duration
	writeMu           sync.Mutex
	streamsMu         sync.RWMutex
	streams           map[string]net.Conn
}

func NewController(manager *runtime.Manager, registry *Registry, parentTunnelURL string, tunnelPath string, heartbeatInterval time.Duration) *Controller {
	return &Controller{
		manager:           manager,
		registry:          registry,
		parentTunnelURL:   strings.TrimRight(parentTunnelURL, "/"),
		tunnelPath:        tunnelPath,
		heartbeatInterval: heartbeatInterval,
		streams:           make(map[string]net.Conn),
	}
}

func (c *Controller) Run() {
	if c.parentTunnelURL == "" {
		return
	}
	if c.heartbeatInterval <= 0 {
		c.heartbeatInterval = 15 * time.Second
	}
	for {
		if !c.manager.Bound() {
			time.Sleep(2 * time.Second)
			continue
		}
		current := c.manager.Current()
		if current.NodeParentID == "" {
			time.Sleep(5 * time.Second)
			continue
		}
		if err := c.connect(current); err != nil {
			log.Printf("node tunnel disconnected nodeID=%s parentNodeID=%s err=%v", current.NodeID, current.NodeParentID, err)
			c.closeStreams()
			c.report(current, "disconnected", "")
			time.Sleep(3 * time.Second)
			continue
		}
	}
}

func (c *Controller) connect(current runtime.Binding) error {
	wsURL, err := c.websocketURL(current.NodeParentID)
	if err != nil {
		return err
	}
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+current.NodeAccessToken)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		return err
	}
	defer conn.Close()
	now := time.Now().UTC().Format(time.RFC3339)
	c.report(current, "connected", now)
	_ = conn.SetReadDeadline(time.Now().Add(c.heartbeatInterval * 3))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(c.heartbeatInterval * 3))
	})
	c.writeMu.Lock()
	err = conn.WriteJSON(Message{
		Type:      "register",
		NodeID:    current.NodeID,
		ParentID:  current.NodeParentID,
		Timestamp: now,
		Status:    "connected",
	})
	c.writeMu.Unlock()
	if err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() {
		for {
			var message Message
			if err := conn.ReadJSON(&message); err != nil {
				done <- err
				return
			}
			switch message.Type {
			case "heartbeat_ack", "register_ack":
				c.report(current, "connected", time.Now().UTC().Format(time.RFC3339))
			case "probe_request":
				response := c.handleProbeRequest(message)
				response.RequestID = message.RequestID
				response.Type = "probe_response"
				c.writeMu.Lock()
				err := conn.WriteJSON(response)
				c.writeMu.Unlock()
				if err != nil {
					done <- err
					return
				}
			case "open_stream":
				if err := c.handleOpenStream(conn, message); err != nil {
					done <- err
					return
				}
			case "stream_data":
				if err := c.handleStreamData(message); err != nil {
					done <- err
					return
				}
			case "close_stream":
				c.handleStreamClose(message.StreamID)
			}
		}
	}()
	ticker := time.NewTicker(c.heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			return err
		case tick := <-ticker.C:
			c.writeMu.Lock()
			err := conn.WriteJSON(Message{
				Type:      "heartbeat",
				NodeID:    current.NodeID,
				ParentID:  current.NodeParentID,
				Timestamp: tick.UTC().Format(time.RFC3339),
				Status:    "connected",
			})
			c.writeMu.Unlock()
			if err != nil {
				return err
			}
			c.report(current, "connected", tick.UTC().Format(time.RFC3339))
		}
	}
}

func (c *Controller) handleProbeRequest(message Message) Message {
	if len(message.RemainingHopNodeIDs) == 0 {
		return Message{Status: "connected", Message: "chain_reachable"}
	}
	nextNodeID := message.RemainingHopNodeIDs[0]
	response, err := c.registry.ForwardProbe(nextNodeID, message.RequestID, message.RemainingHopNodeIDs[1:], message.TargetHost, message.TargetPort)
	if err != nil {
		return Message{Status: "failed", Message: "next_hop_unreachable"}
	}
	return response
}

func (c *Controller) handleOpenStream(wsConn *websocket.Conn, message Message) error {
	targetConn, err := c.resolveStreamTarget(message)
	if err != nil {
		return c.writeMessage(wsConn, Message{
			Type:     "open_ack",
			StreamID: message.StreamID,
			Status:   "failed",
			Message:  err.Error(),
		})
	}
	c.streamsMu.Lock()
	c.streams[message.StreamID] = targetConn
	c.streamsMu.Unlock()
	if err := c.writeMessage(wsConn, Message{
		Type:     "open_ack",
		StreamID: message.StreamID,
		Status:   "connected",
		Message:  "stream_ready",
	}); err != nil {
		targetConn.Close()
		c.handleStreamClose(message.StreamID)
		return err
	}
	go c.pipeStreamBack(wsConn, message.StreamID, targetConn)
	return nil
}

func (c *Controller) resolveStreamTarget(message Message) (net.Conn, error) {
	if len(message.RemainingHopNodeIDs) > 0 {
		nextNodeID := message.RemainingHopNodeIDs[0]
		return c.registry.OpenStream(nextNodeID, message.RemainingHopNodeIDs[1:], message.TargetHost, message.TargetPort)
	}
	return net.Dial("tcp", net.JoinHostPort(message.TargetHost, strconvPort(message.TargetPort)))
}

func (c *Controller) handleStreamData(message Message) error {
	c.streamsMu.RLock()
	targetConn, ok := c.streams[message.StreamID]
	c.streamsMu.RUnlock()
	if !ok {
		return nil
	}
	payload, err := base64.StdEncoding.DecodeString(message.Data)
	if err != nil {
		return err
	}
	_, err = targetConn.Write(payload)
	return err
}

func (c *Controller) handleStreamClose(streamID string) {
	c.streamsMu.Lock()
	targetConn, ok := c.streams[streamID]
	if ok {
		delete(c.streams, streamID)
	}
	c.streamsMu.Unlock()
	if ok {
		_ = targetConn.Close()
	}
}

func (c *Controller) pipeStreamBack(wsConn *websocket.Conn, streamID string, targetConn net.Conn) {
	buffer := make([]byte, streamChunkSize)
	for {
		n, err := targetConn.Read(buffer)
		if n > 0 {
			if writeErr := c.writeMessage(wsConn, Message{
				Type:     "stream_data",
				StreamID: streamID,
				Data:     base64.StdEncoding.EncodeToString(buffer[:n]),
			}); writeErr != nil {
				break
			}
		}
		if err != nil {
			if err != io.EOF {
				_ = c.writeMessage(wsConn, Message{
					Type:     "close_stream",
					StreamID: streamID,
					Message:  err.Error(),
				})
			} else {
				_ = c.writeMessage(wsConn, Message{
					Type:     "close_stream",
					StreamID: streamID,
					Message:  "eof",
				})
			}
			break
		}
	}
	c.handleStreamClose(streamID)
}

func (c *Controller) writeMessage(conn *websocket.Conn, message Message) error {
	c.writeMu.Lock()
	err := conn.WriteJSON(message)
	c.writeMu.Unlock()
	return err
}

func (c *Controller) closeStreams() {
	c.streamsMu.Lock()
	streams := c.streams
	c.streams = make(map[string]net.Conn)
	c.streamsMu.Unlock()
	for _, item := range streams {
		_ = item.Close()
	}
}

func (c *Controller) websocketURL(parentNodeID string) (string, error) {
	base, err := url.Parse(c.parentTunnelURL)
	if err != nil {
		return "", err
	}
	switch base.Scheme {
	case "https":
		base.Scheme = "wss"
	default:
		base.Scheme = "ws"
	}
	base.Path = c.tunnelPath
	query := base.Query()
	query.Set("parentNodeId", parentNodeID)
	base.RawQuery = query.Encode()
	return base.String(), nil
}

func (c *Controller) report(current runtime.Binding, status string, lastHeartbeatAt string) {
	client := controlplane.New(current.ControlPlaneURL, current.NodeAccessToken)
	address, err := c.websocketURL(current.NodeParentID)
	if err != nil {
		return
	}
	_, _ = client.UpsertTransport(domain.UpsertNodeTransportInput{
		TransportType:   "reverse_ws_parent",
		Direction:       "outbound",
		Address:         address,
		Status:          status,
		ParentNodeID:    current.NodeParentID,
		ConnectedAt:     lastHeartbeatAt,
		LastHeartbeatAt: lastHeartbeatAt,
		LatencyMs:       0,
		Details:         map[string]string{"source": "parent_tunnel"},
	})
}

func strconvPort(port int) string {
	if port <= 0 {
		return "0"
	}
	return strconv.Itoa(port)
}
