package tunnel

import (
	"encoding/base64"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	ErrStreamOpenTimeout   = errors.New("stream_open_timeout")
	ErrChildTunnelClosed   = errors.New("child_tunnel_closed")
	ErrChildTunnelNotFound = errors.New("child_tunnel_not_found")
	errStreamBackpressure  = errors.New("stream_backpressure")
)

const streamOpenAckTimeout = 5 * time.Second
const streamReconnectWaitTimeout = 2 * time.Second
const streamDataQueueTimeout = 5 * time.Second
const streamTransportWriteTimeout = 30 * time.Second

func IsTunnelUnavailable(err error) bool {
	return errors.Is(err, ErrStreamOpenTimeout) || errors.Is(err, ErrChildTunnelClosed) || errors.Is(err, ErrChildTunnelNotFound)
}

func (s *childSession) writeMessage(message Message) error {
	return s.writeMessageWithDeadline(message, time.Now().Add(streamTransportWriteTimeout))
}

func (s *childSession) writeMessageWithDeadline(message Message, deadline time.Time) error {
	s.writeMu.Lock()
	if !deadline.IsZero() {
		_ = s.conn.SetWriteDeadline(deadline)
	}
	err := s.conn.WriteJSON(message)
	if !deadline.IsZero() {
		_ = s.conn.SetWriteDeadline(time.Time{})
	}
	s.writeMu.Unlock()
	return err
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
	closeOnce sync.Once
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
	if err := s.writeMessage(message); err != nil {
		return Message{}, err
	}
	select {
	case response := <-ch:
		return response, nil
	case <-time.After(streamOpenAckTimeout):
		return Message{}, errors.New("probe_timeout")
	case <-s.done:
		return Message{}, ErrChildTunnelClosed
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
	if err := s.writeMessage(message); err != nil {
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
	case <-time.After(streamOpenAckTimeout):
		s.removeStream(streamID)
		return nil, ErrStreamOpenTimeout
	case <-s.done:
		s.removeStream(streamID)
		return nil, ErrChildTunnelClosed
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
		case <-time.After(streamDataQueueTimeout):
			stream.closeWithError(errStreamBackpressure)
			if s.conn != nil {
				_ = s.writeMessage(Message{
					Type:     "close_stream",
					StreamID: message.StreamID,
					Message:  errStreamBackpressure.Error(),
				})
			}
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
		stream.closeWithError(ErrChildTunnelClosed)
	}
}

func (s *childSession) close() {
	s.closeOnce.Do(func() {
		s.closeAll()
		close(s.done)
		if s.conn != nil {
			_ = s.conn.Close()
		}
	})
}
