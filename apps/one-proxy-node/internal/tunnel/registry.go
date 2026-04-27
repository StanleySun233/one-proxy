package tunnel

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const streamChunkSize = 32 * 1024

type Registry struct {
	mu       sync.RWMutex
	sessions map[string]*childSession
}

type childSession struct {
	nodeID    string
	conn      *websocket.Conn
	writeMu   sync.Mutex
	pending   map[string]chan Message
	pendingMu sync.Mutex
	streams   map[string]*streamConn
	streamsMu sync.RWMutex
	done      chan struct{}
}

type streamConn struct {
	session      *childSession
	streamID     string
	readCh       chan []byte
	ackCh        chan Message
	done         chan struct{}
	closeOnce    sync.Once
	mu           sync.Mutex
	readBuf      bytes.Buffer
	closeErr     error
	localClosed  bool
	remoteClosed bool
}

func NewRegistry() *Registry {
	return &Registry{sessions: make(map[string]*childSession)}
}

func (r *Registry) Add(nodeID string, conn *websocket.Conn) *childSession {
	session := &childSession{
		nodeID:   nodeID,
		conn:     conn,
		pending:  make(map[string]chan Message),
		streams:  make(map[string]*streamConn),
		done:     make(chan struct{}),
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

func (s *childSession) request(message Message) (Message, error) {
	ch := make(chan Message, 1)
	s.pendingMu.Lock()
	s.pending[message.RequestID] = ch
	s.pendingMu.Unlock()
	defer func() {
		s.pendingMu.Lock()
		delete(s.pending, message.RequestID)
		s.pendingMu.Unlock()
	}()
	s.writeMu.Lock()
	err := s.conn.WriteJSON(message)
	s.writeMu.Unlock()
	if err != nil {
		return Message{}, err
	}
	select {
	case response := <-ch:
		return response, nil
	case <-time.After(5 * time.Second):
		return Message{}, errors.New("probe_timeout")
	case <-s.done:
		return Message{}, errors.New("child_tunnel_closed")
	}
}

func (s *childSession) openStream(remaining []string, targetHost string, targetPort int) (net.Conn, error) {
	streamID := time.Now().UTC().Format(time.RFC3339Nano)
	stream := &streamConn{
		session:  s,
		streamID: streamID,
		readCh:   make(chan []byte, 32),
		ackCh:    make(chan Message, 1),
		done:     make(chan struct{}),
	}
	s.streamsMu.Lock()
	s.streams[streamID] = stream
	s.streamsMu.Unlock()
	message := Message{
		Type:                "open_stream",
		StreamID:            streamID,
		RemainingHopNodeIDs: remaining,
		TargetHost:          targetHost,
		TargetPort:          targetPort,
	}
	s.writeMu.Lock()
	err := s.conn.WriteJSON(message)
	s.writeMu.Unlock()
	if err != nil {
		s.removeStream(streamID)
		return nil, err
	}
	select {
	case ack := <-stream.ackCh:
		if ack.Status != "connected" {
			s.removeStream(streamID)
			if ack.Message == "" {
				return nil, errors.New("stream_open_failed")
			}
			return nil, errors.New(ack.Message)
		}
		return stream, nil
	case <-time.After(5 * time.Second):
		s.removeStream(streamID)
		return nil, errors.New("stream_open_timeout")
	case <-s.done:
		s.removeStream(streamID)
		return nil, errors.New("child_tunnel_closed")
	}
}

func (s *childSession) resolve(response Message) {
	s.pendingMu.Lock()
	ch, ok := s.pending[response.RequestID]
	s.pendingMu.Unlock()
	if ok {
		ch <- response
	}
}

func (s *childSession) handleMessage(message Message) {
	if message.StreamID == "" {
		return
	}
	s.streamsMu.RLock()
	stream, ok := s.streams[message.StreamID]
	s.streamsMu.RUnlock()
	if !ok {
		return
	}
	switch message.Type {
	case "open_ack":
		select {
		case stream.ackCh <- message:
		default:
		}
	case "stream_data":
		payload, err := base64.StdEncoding.DecodeString(message.Data)
		if err != nil {
			stream.closeWithError(errors.New("invalid_stream_payload"))
			return
		}
		select {
		case stream.readCh <- payload:
		case <-stream.done:
		}
	case "close_stream":
		stream.markRemoteClosed(message.Message)
	}
}

func (s *childSession) removeStream(streamID string) {
	s.streamsMu.Lock()
	stream, ok := s.streams[streamID]
	if ok {
		delete(s.streams, streamID)
	}
	s.streamsMu.Unlock()
	if ok {
		stream.closeWithError(ioEOF(messageOrDefault(stream.closeErr, "stream_closed")))
	}
}

func (s *childSession) closeAll() {
	s.streamsMu.Lock()
	streams := make([]*streamConn, 0, len(s.streams))
	for id, stream := range s.streams {
		delete(s.streams, id)
		streams = append(streams, stream)
	}
	s.streamsMu.Unlock()
	for _, stream := range streams {
		stream.closeWithError(ioEOF("child_tunnel_closed"))
	}
}

func (c *streamConn) Read(p []byte) (int, error) {
	c.mu.Lock()
	if c.readBuf.Len() > 0 {
		n, _ := c.readBuf.Read(p)
		c.mu.Unlock()
		return n, nil
	}
	if c.remoteClosed {
		err := c.closeErr
		c.mu.Unlock()
		if err == nil {
			return 0, net.ErrClosed
		}
		return 0, err
	}
	c.mu.Unlock()
	data, ok := <-c.readCh
	if !ok {
		c.mu.Lock()
		err := c.closeErr
		c.mu.Unlock()
		if err == nil {
			return 0, net.ErrClosed
		}
		return 0, err
	}
	c.mu.Lock()
	c.readBuf.Write(data)
	n, _ := c.readBuf.Read(p)
	c.mu.Unlock()
	return n, nil
}

func (c *streamConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	if c.localClosed || c.remoteClosed {
		c.mu.Unlock()
		return 0, net.ErrClosed
	}
	c.mu.Unlock()
	total := 0
	for len(p) > 0 {
		chunkLen := len(p)
		if chunkLen > streamChunkSize {
			chunkLen = streamChunkSize
		}
		chunk := p[:chunkLen]
		p = p[chunkLen:]
		c.session.writeMu.Lock()
		err := c.session.conn.WriteJSON(Message{
			Type:     "stream_data",
			StreamID: c.streamID,
			Data:     base64.StdEncoding.EncodeToString(chunk),
		})
		c.session.writeMu.Unlock()
		if err != nil {
			c.closeWithError(err)
			return total, err
		}
		total += chunkLen
	}
	return total, nil
}

func (c *streamConn) Close() error {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.localClosed = true
		c.mu.Unlock()
		c.session.writeMu.Lock()
		_ = c.session.conn.WriteJSON(Message{
			Type:     "close_stream",
			StreamID: c.streamID,
			Message:  "closed",
		})
		c.session.writeMu.Unlock()
		c.session.streamsMu.Lock()
		delete(c.session.streams, c.streamID)
		c.session.streamsMu.Unlock()
		close(c.readCh)
		close(c.done)
	})
	return nil
}

func (c *streamConn) LocalAddr() net.Addr  { return tunnelAddr("local") }
func (c *streamConn) RemoteAddr() net.Addr { return tunnelAddr("remote") }
func (c *streamConn) SetDeadline(_ time.Time) error      { return nil }
func (c *streamConn) SetReadDeadline(_ time.Time) error  { return nil }
func (c *streamConn) SetWriteDeadline(_ time.Time) error { return nil }

func (c *streamConn) closeWithError(err error) {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.localClosed = true
		c.remoteClosed = true
		c.closeErr = err
		c.mu.Unlock()
		c.session.streamsMu.Lock()
		delete(c.session.streams, c.streamID)
		c.session.streamsMu.Unlock()
		close(c.readCh)
		close(c.done)
	})
}

func (c *streamConn) markRemoteClosed(reason string) {
	c.mu.Lock()
	c.remoteClosed = true
	if c.closeErr == nil {
		c.closeErr = ioEOF(reason)
	}
	c.mu.Unlock()
	c.session.streamsMu.Lock()
	delete(c.session.streams, c.streamID)
	c.session.streamsMu.Unlock()
	closeOnce(c.done)
	closeOnce(c.readCh)
}

type tunnelAddr string

func (a tunnelAddr) Network() string { return "tunnel" }
func (a tunnelAddr) String() string  { return string(a) }

func ioEOF(reason string) error {
	if reason == "" || reason == "closed" || reason == "eof" {
		return io.EOF
	}
	return errors.New(reason)
}

func closeOnce[T any](ch chan T) {
	defer func() {
		_ = recover()
	}()
	close(ch)
}

func messageOrDefault(err error, fallback string) string {
	if err != nil {
		return err.Error()
	}
	return fallback
}
