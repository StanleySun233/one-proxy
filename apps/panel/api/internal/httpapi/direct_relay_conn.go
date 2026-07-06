package httpapi

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type relayWSConn struct {
	conn   *websocket.Conn
	reader io.Reader
	write  sync.Mutex
}

func newRelayWSConn(conn *websocket.Conn) net.Conn {
	return &relayWSConn{conn: conn}
}

func (c *relayWSConn) Read(p []byte) (int, error) {
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

func (c *relayWSConn) Write(p []byte) (int, error) {
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

func (c *relayWSConn) Close() error {
	return c.conn.Close()
}

func (c *relayWSConn) LocalAddr() net.Addr {
	return relayNetAddr("panel")
}

func (c *relayWSConn) RemoteAddr() net.Addr {
	return relayNetAddr("node")
}

func (c *relayWSConn) SetDeadline(t time.Time) error {
	if err := c.conn.SetReadDeadline(t); err != nil {
		return err
	}
	return c.conn.SetWriteDeadline(t)
}

func (c *relayWSConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *relayWSConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

type relayNetAddr string

func (a relayNetAddr) Network() string { return "direct_relay" }
func (a relayNetAddr) String() string  { return string(a) }
