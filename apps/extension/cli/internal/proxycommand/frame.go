package proxycommand

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
)

type accessFrame struct {
	Token      string `json:"token"`
	TargetHost string `json:"targetHost"`
	TargetPort int    `json:"targetPort"`
}

type accessResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func RunTCPFrame(cfg Config, input io.Reader, output io.Writer) error {
	conn, err := DialTCPFrame(cfg)
	if err != nil {
		return err
	}
	defer conn.Close()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(conn, input)
		if halfCloser, ok := conn.(interface{ CloseWrite() error }); ok {
			_ = halfCloser.CloseWrite()
		}
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(output, conn)
	}()
	wg.Wait()
	return nil
}

func DialTCPFrame(cfg Config) (net.Conn, error) {
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
	frame := accessFrame{Token: token, TargetHost: cfg.TargetHost, TargetPort: cfg.TargetPort}
	if err := json.NewEncoder(conn).Encode(frame); err != nil {
		conn.Close()
		return nil, err
	}
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, err
	}
	var response accessResponse
	if err := json.Unmarshal([]byte(line), &response); err != nil {
		conn.Close()
		return nil, err
	}
	if response.Status != "connected" {
		conn.Close()
		if response.Message == "" {
			response.Message = "tcp_access_failed"
		}
		return nil, errors.New(response.Message)
	}
	return bufferedConn{Conn: conn, reader: reader}, nil
}
