package tunnel

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

const streamChunkSize = 32 * 1024

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

func (c *streamConn) LocalAddr() net.Addr                { return tunnelAddr("local") }
func (c *streamConn) RemoteAddr() net.Addr               { return tunnelAddr("remote") }
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
