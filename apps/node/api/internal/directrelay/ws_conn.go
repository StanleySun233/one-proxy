package directrelay

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type wsConn struct {
	conn      *websocket.Conn
	reader    io.Reader
	write     sync.Mutex
	done      chan struct{}
	closeOnce sync.Once
}

func newWSConn(conn *websocket.Conn) net.Conn {
	wrapped := &wsConn{conn: conn, done: make(chan struct{})}
	go wrapped.keepAlive()
	return wrapped
}

func (c *wsConn) Read(p []byte) (int, error) {
	for {
		if c.reader == nil {
			messageType, reader, err := c.conn.NextReader()
			if err != nil {
				return 0, err
			}
			if messageType != websocket.BinaryMessage {
				continue
			}
			c.reader = reader
		}
		n, err := c.reader.Read(p)
		if err == io.EOF {
			c.reader = nil
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
}

func (c *wsConn) Write(p []byte) (int, error) {
	c.write.Lock()
	defer c.write.Unlock()
	writer, err := c.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return 0, err
	}
	n, err := writer.Write(p)
	closeErr := writer.Close()
	if err != nil {
		return n, err
	}
	return n, closeErr
}

func (c *wsConn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.done)
		err = c.conn.Close()
	})
	return err
}

func (c *wsConn) LocalAddr() net.Addr {
	return relayAddr("local")
}

func (c *wsConn) RemoteAddr() net.Addr {
	return relayAddr("remote")
}

func (c *wsConn) SetDeadline(t time.Time) error {
	if err := c.conn.SetReadDeadline(t); err != nil {
		return err
	}
	return c.conn.SetWriteDeadline(t)
}

func (c *wsConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *wsConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

type relayAddr string

func (a relayAddr) Network() string { return "direct_relay" }
func (a relayAddr) String() string  { return string(a) }

func (c *wsConn) keepAlive() {
	ticker := time.NewTicker(relayKeepAliveInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.write.Lock()
			err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(relayWriteTimeout))
			c.write.Unlock()
			if err != nil {
				_ = c.Close()
				return
			}
		}
	}
}
