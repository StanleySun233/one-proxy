package direct

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
	"github.com/quic-go/quic-go"
)

const directALPN = "one-proxy-direct/1"

type streamOpenRequest struct {
	Mode          string   `json:"mode,omitempty"`
	SessionID     string   `json:"sessionId,omitempty"`
	PunchToken    string   `json:"punchToken,omitempty"`
	RemainingHops []string `json:"remainingHops,omitempty"`
	TargetHost    string   `json:"targetHost"`
	TargetPort    int      `json:"targetPort"`
}

type streamOpenAck struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type quicStreamConn struct {
	*quic.Stream
	conn       *quic.Conn
	localAddr  net.Addr
	remoteAddr net.Addr
}

func (c quicStreamConn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c quicStreamConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c quicStreamConn) Close() error {
	err := c.Stream.Close()
	if c.conn != nil {
		_ = c.conn.CloseWithError(0, "")
	}
	return err
}

func (r *Registry) AttachQUICTransport(transport *quic.Transport, nodeID string) error {
	tlsConfig, identity, err := serverTLSConfig(nodeID)
	if err != nil {
		return err
	}
	listener, err := transport.Listen(tlsConfig, &quic.Config{})
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.transport = transport
	r.listener = listener
	r.directIdentity = identity
	r.mu.Unlock()
	return nil
}

func (r *Registry) RunQUICServer(ctx context.Context) {
	r.mu.RLock()
	listener := r.listener
	r.mu.RUnlock()
	if listener == nil {
		return
	}
	for {
		conn, err := listener.Accept(ctx)
		if err != nil {
			return
		}
		go r.handleQUICConn(ctx, conn)
	}
}

func (r *Registry) handleQUICConn(ctx context.Context, conn *quic.Conn) {
	defer conn.CloseWithError(0, "")
	for {
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			return
		}
		go r.handleQUICStream(ctx, conn, stream)
	}
}

func (r *Registry) handleQUICStream(ctx context.Context, conn *quic.Conn, stream *quic.Stream) {
	reader := bufio.NewReader(stream)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		_ = stream.Close()
		return
	}
	var request streamOpenRequest
	if err := json.Unmarshal(line, &request); err != nil {
		_ = writeStreamAck(stream, "failed", "invalid_direct_stream_request")
		_ = stream.Close()
		return
	}
	if request.Mode != "" && request.Mode != "client_direct" {
		_ = writeStreamAck(stream, "failed", "invalid_direct_stream_mode")
		_ = stream.Close()
		return
	}
	if request.Mode == "client_direct" && (request.SessionID == "" || request.PunchToken == "") {
		_ = writeStreamAck(stream, "failed", "invalid_direct_session")
		_ = stream.Close()
		return
	}
	if request.Mode == "client_direct" {
		validated, ok := r.validateClientSession(ctx, request)
		if !ok {
			_ = writeStreamAck(stream, "failed", "invalid_direct_session")
			_ = stream.Close()
			return
		}
		request.TargetHost = validated.TargetHost
		request.TargetPort = validated.TargetPort
	}
	if len(request.RemainingHops) > 0 {
		_ = writeStreamAck(stream, "failed", "direct_remaining_hops_not_supported")
		_ = stream.Close()
		return
	}
	targetConn, err := net.Dial("tcp", net.JoinHostPort(request.TargetHost, strconv.Itoa(request.TargetPort)))
	if err != nil {
		_ = writeStreamAck(stream, "failed", err.Error())
		_ = stream.Close()
		return
	}
	if err := writeStreamAck(stream, "connected", "stream_ready"); err != nil {
		_ = targetConn.Close()
		_ = stream.Close()
		return
	}
	bridgeQUICStream(ctx, quicStreamConn{Stream: stream, localAddr: conn.LocalAddr(), remoteAddr: conn.RemoteAddr()}, reader, targetConn)
}

func (r *Registry) validateClientSession(ctx context.Context, request streamOpenRequest) (ClientSessionValidationResult, bool) {
	r.mu.RLock()
	validator := r.clientValidator
	r.mu.RUnlock()
	if validator == nil {
		return ClientSessionValidationResult{}, false
	}
	result, err := validator.ValidateClientDirectSession(ctx, ClientSessionValidationRequest{
		SessionID:  request.SessionID,
		PunchToken: request.PunchToken,
		TargetHost: request.TargetHost,
		TargetPort: request.TargetPort,
	})
	if err != nil || !result.Valid {
		return ClientSessionValidationResult{}, false
	}
	return result, true
}

func (r *Registry) OpenDirectStream(ctx context.Context, nextHop domain.Node, remaining []string, targetHost string, targetPort int) (net.Conn, error) {
	state, ok := r.Get(nextHop.ID)
	if !ok {
		return nil, errors.New("direct_peer_not_found")
	}
	if state.Status != domain.DirectStatusConnected {
		return nil, errors.New("direct_peer_not_connected")
	}
	if state.SelectedCandidate.Address == "" || state.SelectedCandidate.Port <= 0 {
		return nil, errors.New("direct_peer_candidate_not_found")
	}
	r.mu.RLock()
	transport := r.transport
	r.mu.RUnlock()
	if transport == nil {
		return nil, errors.New("direct_quic_not_ready")
	}
	addr, err := net.ResolveUDPAddr("udp", candidateAddress(state.SelectedCandidate))
	if err != nil {
		return nil, err
	}
	tlsConfig, err := clientTLSConfig(state.PeerIdentity)
	if err != nil {
		return nil, err
	}
	conn, err := transport.Dial(ctx, addr, tlsConfig, &quic.Config{})
	if err != nil {
		return nil, err
	}
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		_ = conn.CloseWithError(0, "")
		return nil, err
	}
	request := streamOpenRequest{RemainingHops: remaining, TargetHost: targetHost, TargetPort: targetPort}
	if err := json.NewEncoder(stream).Encode(request); err != nil {
		_ = conn.CloseWithError(0, "")
		return nil, err
	}
	var ack streamOpenAck
	if err := json.NewDecoder(stream).Decode(&ack); err != nil {
		_ = conn.CloseWithError(0, "")
		return nil, err
	}
	if ack.Status != "connected" {
		_ = conn.CloseWithError(0, "")
		if ack.Message == "" {
			return nil, errors.New("direct_stream_open_failed")
		}
		return nil, errors.New(ack.Message)
	}
	return quicStreamConn{Stream: stream, conn: conn, localAddr: conn.LocalAddr(), remoteAddr: conn.RemoteAddr()}, nil
}

func writeStreamAck(stream io.Writer, status string, message string) error {
	return json.NewEncoder(stream).Encode(streamOpenAck{Status: status, Message: message})
}

func bridgeQUICStream(ctx context.Context, stream net.Conn, reader *bufio.Reader, target net.Conn) {
	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(target, reader)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(stream, target)
		done <- struct{}{}
	}()
	select {
	case <-ctx.Done():
	case <-done:
	}
	_ = stream.Close()
	_ = target.Close()
}

func serverTLSConfig(nodeID string) (*tls.Config, domain.DirectNodeIdentity, error) {
	serverName := directServerName(nodeID)
	if serverName == "" {
		return nil, domain.DirectNodeIdentity{}, errors.New("invalid_direct_node_identity")
	}
	cert, certPEM, fingerprint := selfSignedCertificate(serverName)
	identity := domain.DirectNodeIdentity{
		NodeID:                       nodeID,
		ServerName:                   serverName,
		CertificateFingerprintSHA256: fingerprint,
		TrustMaterial:                certPEM,
	}
	return &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{cert}, NextProtos: []string{directALPN}}, identity, nil
}

func clientTLSConfig(identity domain.DirectNodeIdentity) (*tls.Config, error) {
	if identity.ServerName == "" || identity.TrustMaterial == "" || identity.CertificateFingerprintSHA256 == "" {
		return nil, errors.New("invalid_direct_node_identity")
	}
	block, _ := pem.Decode([]byte(identity.TrustMaterial))
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("invalid_direct_node_identity")
	}
	fingerprint := sha256.Sum256(block.Bytes)
	if !strings.EqualFold(hex.EncodeToString(fingerprint[:]), identity.CertificateFingerprintSHA256) {
		return nil, errors.New("invalid_direct_node_identity")
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM([]byte(identity.TrustMaterial)) {
		return nil, errors.New("invalid_direct_node_identity")
	}
	return &tls.Config{MinVersion: tls.VersionTLS12, ServerName: identity.ServerName, RootCAs: roots, NextProtos: []string{directALPN}}, nil
}

func directServerName(nodeID string) string {
	value := strings.TrimSpace(strings.ToLower(nodeID))
	if value == "" {
		return ""
	}
	var builder strings.Builder
	for _, item := range value {
		if (item >= 'a' && item <= 'z') || (item >= '0' && item <= '9') || item == '-' {
			builder.WriteRune(item)
		} else {
			builder.WriteByte('-')
		}
	}
	name := strings.Trim(builder.String(), "-")
	if name == "" {
		return ""
	}
	return name + ".direct.oneproxy.local"
}

func selfSignedCertificate(serverName string) (tls.Certificate, string, string) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{serverName},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	certPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	fingerprint := sha256.Sum256(der)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, certPEM, hex.EncodeToString(fingerprint[:])
}
