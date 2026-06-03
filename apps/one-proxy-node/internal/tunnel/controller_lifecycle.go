package tunnel

import (
	"log"
	"net/http"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/runtime"
	"github.com/gorilla/websocket"
)

func (c *Controller) Run() {
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
	wsURL, err := c.websocketURL(current, current.NodeParentID)
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
	go c.readMessages(conn, current, done)
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

func (c *Controller) readMessages(conn *websocket.Conn, current runtime.Binding, done chan<- error) {
	for {
		var message Message
		if err := conn.ReadJSON(&message); err != nil {
			done <- err
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(c.heartbeatInterval * 3))
		switch message.Type {
		case "heartbeat_ack", "register_ack":
			c.report(current, "connected", time.Now().UTC().Format(time.RFC3339))
		case "probe_request":
			response := c.handleProbeRequest(message)
			response.RequestID = message.RequestID
			response.Type = "probe_response"
			if err := c.writeMessage(conn, response); err != nil {
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
}
