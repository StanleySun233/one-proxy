package direct

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net"
	"strconv"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
	"github.com/quic-go/quic-go"
)

const directALPN = "one-proxy-direct/1"

type streamOpenRequest struct {
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

func (r *Registry) AttachQUICTransport(transport *quic.Transport) error {
	listener, err := transport.Listen(serverTLSConfig(), &quic.Config{})
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.transport = transport
	r.listener = listener
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
	conn, err := transport.Dial(ctx, addr, clientTLSConfig(), &quic.Config{})
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

func serverTLSConfig() *tls.Config {
	cert := selfSignedCertificate()
	return &tls.Config{Certificates: []tls.Certificate{cert}, NextProtos: []string{directALPN}}
}

func clientTLSConfig() *tls.Config {
	return &tls.Config{InsecureSkipVerify: true, NextProtos: []string{directALPN}}
}

func selfSignedCertificate() tls.Certificate {
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
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}
