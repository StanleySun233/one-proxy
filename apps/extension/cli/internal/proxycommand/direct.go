package proxycommand

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/quic-go/quic-go"
)

const directALPN = "one-proxy-direct/1"

type directSessionRequest struct {
	AccessPathID string `json:"accessPathId"`
	TargetHost   string `json:"targetHost,omitempty"`
	TargetPort   int    `json:"targetPort,omitempty"`
}

type directSession struct {
	SessionID      string            `json:"sessionId"`
	TargetHost     string            `json:"targetHost"`
	TargetPort     int               `json:"targetPort"`
	PunchToken     string            `json:"punchToken"`
	NodeCandidates []directCandidate `json:"nodeCandidates"`
}

type directCandidate struct {
	Address  string `json:"address"`
	Port     int    `json:"port"`
	Priority int    `json:"priority"`
}

type directStreamRequest struct {
	Mode       string `json:"mode"`
	SessionID  string `json:"sessionId"`
	PunchToken string `json:"punchToken"`
	TargetHost string `json:"targetHost"`
	TargetPort int    `json:"targetPort"`
}

type directStreamAck struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func DialDirect(cfg Config) (net.Conn, error) {
	if err := cfg.validateDirect(); err != nil {
		return nil, err
	}
	session, err := requestDirectSession(cfg)
	if err != nil {
		return nil, err
	}
	timeout := cfg.ConnectTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	candidates := append([]directCandidate(nil), session.NodeCandidates...)
	sort.SliceStable(candidates, func(left, right int) bool {
		return candidates[left].Priority > candidates[right].Priority
	})
	var lastErr error
	for _, candidate := range candidates {
		if candidate.Address == "" || candidate.Port <= 0 {
			continue
		}
		conn, err := openDirectCandidate(ctx, candidate, session)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("direct_candidates_unavailable")
}

func requestDirectSession(cfg Config) (directSession, error) {
	token, err := readAccessToken(cfg)
	if err != nil {
		return directSession{}, err
	}
	body, _ := json.Marshal(directSessionRequest{
		AccessPathID: cfg.AccessPathID,
		TargetHost:   cfg.TargetHost,
		TargetPort:   cfg.TargetPort,
	})
	req, err := http.NewRequest(http.MethodPost, trimURL(cfg.PanelURL)+"/api/proxy/extension/direct/session", bytes.NewReader(body))
	if err != nil {
		return directSession{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-One-Proxy-Access-Token", token)
	req.Header.Set("X-One-Proxy-Tenant-ID", cfg.TenantID)
	return decodeControlPlane[directSession](req)
}

func openDirectCandidate(ctx context.Context, candidate directCandidate, session directSession) (net.Conn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(candidate.Address, strconv.Itoa(candidate.Port)))
	if err != nil {
		return nil, err
	}
	packetConn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, err
	}
	transport := &quic.Transport{Conn: packetConn}
	conn, err := transport.Dial(ctx, udpAddr, &tls.Config{InsecureSkipVerify: true, NextProtos: []string{directALPN}}, &quic.Config{})
	if err != nil {
		_ = packetConn.Close()
		return nil, err
	}
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		_ = conn.CloseWithError(0, "")
		_ = packetConn.Close()
		return nil, err
	}
	request := directStreamRequest{
		Mode:       "client_direct",
		SessionID:  session.SessionID,
		PunchToken: session.PunchToken,
		TargetHost: session.TargetHost,
		TargetPort: session.TargetPort,
	}
	if err := json.NewEncoder(stream).Encode(request); err != nil {
		_ = conn.CloseWithError(0, "")
		_ = packetConn.Close()
		return nil, err
	}
	var ack directStreamAck
	if err := json.NewDecoder(stream).Decode(&ack); err != nil {
		_ = conn.CloseWithError(0, "")
		_ = packetConn.Close()
		return nil, err
	}
	if ack.Status != "connected" {
		_ = conn.CloseWithError(0, "")
		_ = packetConn.Close()
		if ack.Message == "" {
			return nil, errors.New("direct_stream_open_failed")
		}
		return nil, errors.New(ack.Message)
	}
	return quicClientConn{Stream: stream, conn: conn, packetConn: packetConn, localAddr: conn.LocalAddr(), remoteAddr: conn.RemoteAddr()}, nil
}

func (cfg Config) validateDirect() error {
	if strings.TrimSpace(cfg.PanelURL) == "" {
		return fmt.Errorf("missing --direct-panel-url")
	}
	if strings.TrimSpace(cfg.AccessPathID) == "" {
		return fmt.Errorf("missing --direct-access-path-id")
	}
	if strings.TrimSpace(cfg.TenantID) == "" {
		return fmt.Errorf("missing --tenant-id")
	}
	if strings.TrimSpace(cfg.TargetHost) == "" {
		return fmt.Errorf("missing --target-host")
	}
	if cfg.TargetPort <= 0 || cfg.TargetPort > 65535 {
		return fmt.Errorf("invalid --target-port")
	}
	return nil
}

func readAccessToken(cfg Config) (string, error) {
	if cfg.AccessTokenFile != "" {
		return ReadToken("", cfg.AccessTokenFile)
	}
	envName := cfg.AccessTokenEnv
	if envName == "" {
		envName = "ONEPROXY_ACCESS_TOKEN"
	}
	return ReadToken(envName, "")
}

func decodeControlPlane[T any](req *http.Request) (T, error) {
	var result T
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	var envelope struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    T      `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return result, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || envelope.Code != 0 {
		if envelope.Message != "" {
			return result, errors.New(envelope.Message)
		}
		return result, fmt.Errorf("http_%d", resp.StatusCode)
	}
	return envelope.Data, nil
}

type quicClientConn struct {
	*quic.Stream
	conn       *quic.Conn
	packetConn net.PacketConn
	localAddr  net.Addr
	remoteAddr net.Addr
}

func (c quicClientConn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c quicClientConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c quicClientConn) Close() error {
	err := c.Stream.Close()
	_ = c.conn.CloseWithError(0, "")
	_ = c.packetConn.Close()
	return err
}

func trimURL(value string) string {
	return strings.TrimRight(value, "/")
}
