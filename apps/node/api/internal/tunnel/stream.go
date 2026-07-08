package tunnel

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

const streamChunkSize = 32 * 1024

type streamConn struct {
	session       *childSession
	streamID      string
	readCh        chan []byte
	ackCh         chan Message
	done          chan struct{}
	closeOnce     sync.Once
	mu            sync.Mutex
	readBuf       bytes.Buffer
	closeErr      error
	localClosed   bool
	remoteClosed  bool
	deadline      time.Time
	readDeadline  time.Time
	writeDeadline time.Time
}

func (c *streamConn) Read(p []byte) (int, error) {
	c.mu.Lock()
	if c.readBuf.Len() > 0 {
		n, _ := c.readBuf.Read(p)
		c.mu.Unlock()
		return n, nil
	}
	if c.remoteClosed {
		select {
		case data, ok := <-c.readCh:
			if ok {
				c.readBuf.Write(data)
				n, _ := c.readBuf.Read(p)
				c.mu.Unlock()
				return n, nil
			}
		default:
		}
		err := c.closeErr
		c.mu.Unlock()
		if err == nil {
			return 0, net.ErrClosed
		}
		return 0, err
	}
	c.mu.Unlock()
	deadline := c.currentReadDeadline()
	if !deadline.IsZero() {
		if time.Now().After(deadline) {
			return 0, os.ErrDeadlineExceeded
		}
		timer := time.NewTimer(time.Until(deadline))
		defer timer.Stop()
		select {
		case data, ok := <-c.readCh:
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
		case <-timer.C:
			return 0, os.ErrDeadlineExceeded
		}
	}
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
	deadline := c.effectiveDeadline(c.writeDeadline)
	c.mu.Unlock()
	if !deadline.IsZero() && time.Now().After(deadline) {
		return 0, os.ErrDeadlineExceeded
	}
	total := 0
	for len(p) > 0 {
		writeDeadline := deadline
		if writeDeadline.IsZero() {
			writeDeadline = time.Now().Add(streamTransportWriteTimeout)
		}
		if !writeDeadline.IsZero() && time.Now().After(writeDeadline) {
			return total, os.ErrDeadlineExceeded
		}
		chunkLen := len(p)
		if chunkLen > streamChunkSize {
			chunkLen = streamChunkSize
		}
		chunk := p[:chunkLen]
		p = p[chunkLen:]
		err := c.session.writeMessageWithDeadline(Message{
			Type:     "stream_data",
			StreamID: c.streamID,
			Data:     base64.StdEncoding.EncodeToString(chunk),
		}, writeDeadline)
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
		_ = c.session.writeMessage(Message{
			Type:     "close_stream",
			StreamID: c.streamID,
			Message:  "closed",
		})
		c.session.streamsMu.Lock()
		delete(c.session.streams, c.streamID)
		c.session.streamsMu.Unlock()
		closeOnce(c.readCh)
		closeOnce(c.done)
	})
	return nil
}

func (c *streamConn) LocalAddr() net.Addr  { return tunnelAddr("local") }
func (c *streamConn) RemoteAddr() net.Addr { return tunnelAddr("remote") }

func (c *streamConn) SetDeadline(deadline time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deadline = deadline
	return nil
}

func (c *streamConn) SetReadDeadline(deadline time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readDeadline = deadline
	return nil
}

func (c *streamConn) SetWriteDeadline(deadline time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writeDeadline = deadline
	return nil
}

func (c *streamConn) currentReadDeadline() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.effectiveDeadline(c.readDeadline)
}

func (c *streamConn) effectiveDeadline(specific time.Time) time.Time {
	if c.deadline.IsZero() {
		return specific
	}
	if specific.IsZero() || c.deadline.Before(specific) {
		return c.deadline
	}
	return specific
}

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
		closeOnce(c.readCh)
		closeOnce(c.done)
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
	if reason == ErrChildTunnelClosed.Error() {
		return ErrChildTunnelClosed
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
