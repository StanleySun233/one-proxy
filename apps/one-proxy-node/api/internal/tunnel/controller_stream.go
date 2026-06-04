package tunnel

import (
	"encoding/base64"
	"io"
	"net"

	"github.com/gorilla/websocket"
)

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

func (c *Controller) closeStreams() {
	c.streamsMu.Lock()
	streams := c.streams
	c.streams = make(map[string]net.Conn)
	c.streamsMu.Unlock()
	for _, item := range streams {
		_ = item.Close()
	}
}
