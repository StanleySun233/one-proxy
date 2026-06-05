package proxycommand

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	EntryHost       string
	EntryPort       int
	TargetHost      string
	TargetPort      int
	TokenEnv        string
	TokenFile       string
	PanelURL        string
	AccessTokenEnv  string
	AccessTokenFile string
	TenantID        string
	AccessPathID    string
	ConnectTimeout  time.Duration
}

func DialUpstream(cfg Config) (net.Conn, error) {
	if cfg.PanelURL != "" || cfg.AccessPathID != "" {
		return DialDirect(cfg)
	}
	return DialCONNECT(cfg)
}

func DialCONNECT(cfg Config) (net.Conn, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	token, err := ReadToken(cfg.TokenEnv, cfg.TokenFile)
	if err != nil {
		return nil, err
	}
	timeout := cfg.ConnectTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(cfg.EntryHost, strconv.Itoa(cfg.EntryPort)), timeout)
	if err != nil {
		return nil, err
	}
	if err := writeCONNECT(conn, cfg.targetAddress(), token); err != nil {
		conn.Close()
		return nil, err
	}
	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, &http.Request{Method: http.MethodConnect})
	if err != nil {
		conn.Close()
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		conn.Close()
		return nil, fmt.Errorf("proxy connect failed: %s", response.Status)
	}
	return bufferedConn{Conn: conn, reader: reader}, nil
}

func (cfg Config) validate() error {
	if strings.TrimSpace(cfg.EntryHost) == "" {
		return fmt.Errorf("missing --entry-host")
	}
	if cfg.EntryPort <= 0 || cfg.EntryPort > 65535 {
		return fmt.Errorf("invalid --entry-port")
	}
	if strings.TrimSpace(cfg.TargetHost) == "" {
		return fmt.Errorf("missing --target-host")
	}
	if cfg.TargetPort <= 0 || cfg.TargetPort > 65535 {
		return fmt.Errorf("invalid --target-port")
	}
	return nil
}

func (cfg Config) targetAddress() string {
	return net.JoinHostPort(cfg.TargetHost, strconv.Itoa(cfg.TargetPort))
}

func writeCONNECT(conn net.Conn, target, token string) error {
	auth := base64.StdEncoding.EncodeToString([]byte("token:" + token))
	_, err := fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Authorization: Basic %s\r\n\r\n", target, target, auth)
	return err
}

type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c bufferedConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

func (c bufferedConn) CloseWrite() error {
	tcpConn, ok := c.Conn.(*net.TCPConn)
	if !ok {
		return nil
	}
	return tcpConn.CloseWrite()
}
