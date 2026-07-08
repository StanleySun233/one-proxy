package directrelay

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/controlplane"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/runtime"
	"github.com/gorilla/websocket"
)

const openTimeout = 10 * time.Second

type Client struct {
	manager *runtime.Manager
	writeMu sync.Mutex
}

type openRequest struct {
	Type          string   `json:"type"`
	StreamID      string   `json:"streamId"`
	SourceNodeID  string   `json:"sourceNodeId"`
	TargetHost    string   `json:"targetHost"`
	TargetPort    int      `json:"targetPort"`
	RemainingHops []string `json:"remainingHops"`
}

type controlMessage struct {
	Type     string `json:"type"`
	StreamID string `json:"streamId"`
	Message  string `json:"message,omitempty"`
}

type sourceAck struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func New(manager *runtime.Manager) *Client {
	return &Client{manager: manager}
}

func (c *Client) Run(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		if c.manager == nil || !c.manager.Bound() {
			sleep(ctx, 2*time.Second)
			continue
		}
		current := c.manager.Current()
		if err := c.runControl(ctx, current); err != nil {
			log.Printf("direct relay control disconnected nodeID=%s err=%v", current.NodeID, err)
			sleep(ctx, 3*time.Second)
		}
	}
}

func (c *Client) runControl(ctx context.Context, current runtime.Binding) error {
	wsURL, err := relayURL(current, "/api/node/agent/direct/relay/control", nil)
	if err != nil {
		return err
	}
	conn, resp, err := dialRelay(ctx, wsURL, current.NodeAccessToken)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("direct_relay_control_dial_failed status=%d err=%w", resp.StatusCode, err)
		}
		return err
	}
	defer conn.Close()
	for {
		var request openRequest
		if err := conn.ReadJSON(&request); err != nil {
			return err
		}
		if request.Type == "open_stream" {
			go c.handleOpen(ctx, current, conn, request)
		}
	}
}

func (c *Client) OpenRelayStream(ctx context.Context, peerNodeID string, remaining []string, targetHost string, targetPort int) (net.Conn, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, openTimeout)
	defer cancel()
	if c.manager == nil || !c.manager.Bound() {
		return nil, errors.New("relay_control_unbound")
	}
	current := c.manager.Current()
	values := url.Values{}
	values.Set("peerNodeId", peerNodeID)
	values.Set("targetHost", targetHost)
	values.Set("targetPort", strconv.Itoa(targetPort))
	if len(remaining) > 0 {
		values.Set("remaining", strings.Join(remaining, ","))
	}
	wsURL, err := relayURL(current, "/api/node/agent/direct/relay/source", values)
	if err != nil {
		return nil, err
	}
	conn, _, err := dialRelay(ctx, wsURL, current.NodeAccessToken)
	if err != nil {
		c.reportRelayStatus(current, peerNodeID, domain.TransportStatusFailed, err.Error())
		return nil, err
	}
	var ack sourceAck
	if err := conn.ReadJSON(&ack); err != nil {
		_ = conn.Close()
		c.reportRelayStatus(current, peerNodeID, domain.TransportStatusFailed, err.Error())
		return nil, err
	}
	if ack.Status != domain.TransportStatusConnected {
		_ = conn.Close()
		if ack.Message == "" {
			ack.Message = "relay_stream_open_failed"
		}
		c.reportRelayStatus(current, peerNodeID, domain.TransportStatusFailed, ack.Message)
		return nil, errors.New(ack.Message)
	}
	c.reportRelayStatus(current, peerNodeID, domain.TransportStatusConnected, "")
	return newWSConn(conn), nil
}

func (c *Client) handleOpen(ctx context.Context, current runtime.Binding, controlConn *websocket.Conn, request openRequest) {
	if len(request.RemainingHops) > 0 {
		log.Printf("direct relay target open failed streamID=%s sourceNodeID=%s target=%s:%d err=relay_remaining_hops_not_supported", request.StreamID, request.SourceNodeID, request.TargetHost, request.TargetPort)
		c.writeControl(controlConn, controlMessage{Type: "open_failed", StreamID: request.StreamID, Message: "relay_remaining_hops_not_supported"})
		return
	}
	dialCtx, cancel := context.WithTimeout(ctx, openTimeout)
	targetConn, err := (&net.Dialer{Timeout: openTimeout}).DialContext(dialCtx, "tcp", net.JoinHostPort(request.TargetHost, strconv.Itoa(request.TargetPort)))
	cancel()
	if err != nil {
		log.Printf("direct relay target dial failed streamID=%s sourceNodeID=%s target=%s:%d err=%v", request.StreamID, request.SourceNodeID, request.TargetHost, request.TargetPort, err)
		c.writeControl(controlConn, controlMessage{Type: "open_failed", StreamID: request.StreamID, Message: err.Error()})
		return
	}
	values := url.Values{}
	values.Set("streamId", request.StreamID)
	wsURL, err := relayURL(current, "/api/node/agent/direct/relay/target", values)
	if err != nil {
		_ = targetConn.Close()
		c.writeControl(controlConn, controlMessage{Type: "open_failed", StreamID: request.StreamID, Message: err.Error()})
		return
	}
	wsConn, _, err := dialRelay(ctx, wsURL, current.NodeAccessToken)
	if err != nil {
		_ = targetConn.Close()
		log.Printf("direct relay target attach failed streamID=%s sourceNodeID=%s target=%s:%d err=%v", request.StreamID, request.SourceNodeID, request.TargetHost, request.TargetPort, err)
		c.writeControl(controlConn, controlMessage{Type: "open_failed", StreamID: request.StreamID, Message: err.Error()})
		return
	}
	bridge(ctx, newWSConn(wsConn), targetConn)
}

func (c *Client) writeControl(conn *websocket.Conn, message controlMessage) {
	c.writeMu.Lock()
	_ = conn.WriteJSON(message)
	c.writeMu.Unlock()
}

func (c *Client) reportRelayStatus(current runtime.Binding, peerNodeID string, status string, message string) {
	now := time.Now().UTC().Format(time.RFC3339)
	details := map[string]string{"source": "control_plane_relay"}
	if message != "" {
		details["fallbackReason"] = message
	}
	_, _ = controlplane.New(current.ControlPlaneURL, current.NodeAccessToken).UpsertTransport(domain.UpsertNodeTransportInput{
		TransportType:   domain.TransportTypeDirectRelay,
		Direction:       domain.TransportDirectionPeer,
		Address:         peerNodeID,
		Status:          status,
		ConnectedAt:     connectedAt(status, now),
		LastHeartbeatAt: now,
		Details:         details,
	})
}

func connectedAt(status string, now string) string {
	if status == domain.TransportStatusConnected {
		return now
	}
	return ""
}

func relayURL(current runtime.Binding, path string, values url.Values) (string, error) {
	base, err := url.Parse(current.ControlPlaneURL)
	if err != nil {
		return "", err
	}
	if base.Scheme == "https" {
		base.Scheme = "wss"
	} else {
		base.Scheme = "ws"
	}
	base.Path = path
	if values != nil {
		base.RawQuery = values.Encode()
	}
	return base.String(), nil
}

func dialRelay(ctx context.Context, wsURL string, token string) (*websocket.Conn, *http.Response, error) {
	headers := http.Header{}
	headers.Set("X-One-Proxy-Node-Token", token)
	dialer := websocket.Dialer{HandshakeTimeout: openTimeout}
	return dialer.DialContext(ctx, wsURL, headers)
}

func bridge(ctx context.Context, left net.Conn, right net.Conn) {
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

func sleep(ctx context.Context, duration time.Duration) {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
