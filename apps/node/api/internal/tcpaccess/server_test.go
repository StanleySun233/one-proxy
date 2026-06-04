package tcpaccess

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/proxy"
)

type testTokenValidator struct {
	validations []string
	valid       bool
}

func (v *testTokenValidator) ValidateProxyToken(_ context.Context, tokenHash string) (proxy.TokenValidation, error) {
	v.validations = append(v.validations, tokenHash)
	return proxy.TokenValidation{Valid: v.valid, ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil
}

type fakeStreamOpener struct {
	nextNodeID string
	remaining  []string
	targetHost string
	targetPort int
}

func (o *fakeStreamOpener) OpenStream(nextNodeID string, remaining []string, targetHost string, targetPort int) (net.Conn, error) {
	o.nextNodeID = nextNodeID
	o.remaining = append([]string(nil), remaining...)
	o.targetHost = targetHost
	o.targetPort = targetPort
	left, right := net.Pipe()
	go echoConn(right)
	return left, nil
}

func TestDirectTCPAccessEcho(t *testing.T) {
	target := listenEcho(t)
	defer target.Close()

	validator := &testTokenValidator{valid: true}
	server := New(proxy.NewTokenAuthorizer(proxy.AuthConfig{Validator: validator}), nil)
	listener := listenTCPAccess(t, server)
	defer listener.Close()

	conn := dialTCPAccess(t, listener.Addr().String(), AuthFrame{
		Token:      "direct-token",
		TargetHost: "127.0.0.1",
		TargetPort: portOf(t, target.Addr().String()),
	})
	defer conn.Close()

	if got := roundTrip(t, conn, "ping"); got != "ping" {
		t.Fatalf("echo = %q", got)
	}
	if len(validator.validations) != 1 || validator.validations[0] != sha256Hex("direct-token") {
		t.Fatalf("validations = %v", validator.validations)
	}
}

func TestTCPAccessRejectsInvalidToken(t *testing.T) {
	validator := &testTokenValidator{valid: false}
	server := New(proxy.NewTokenAuthorizer(proxy.AuthConfig{Validator: validator}), nil)
	listener := listenTCPAccess(t, server)
	defer listener.Close()

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := json.NewEncoder(conn).Encode(AuthFrame{
		Token:      "bad-token",
		TargetHost: "127.0.0.1",
		TargetPort: 1,
	}); err != nil {
		t.Fatal(err)
	}
	response := readResponse(t, conn)
	if response.Status != "failed" || response.Message != "auth_required" {
		t.Fatalf("response = %+v", response)
	}
}

func TestChainedTCPAccessUsesStreamOpener(t *testing.T) {
	opener := &fakeStreamOpener{}
	validator := &testTokenValidator{valid: true}
	server := New(proxy.NewTokenAuthorizer(proxy.AuthConfig{Validator: validator}), opener)
	listener := listenTCPAccess(t, server)
	defer listener.Close()

	conn := dialTCPAccess(t, listener.Addr().String(), AuthFrame{
		Token:        "chain-token",
		TargetHost:   "10.0.0.9",
		TargetPort:   22,
		ChainNodeIDs: []string{"node-2", "node-3"},
	})
	defer conn.Close()

	if got := roundTrip(t, conn, "ssh-probe"); got != "ssh-probe" {
		t.Fatalf("echo = %q", got)
	}
	if opener.nextNodeID != "node-2" || !reflect.DeepEqual(opener.remaining, []string{"node-3"}) {
		t.Fatalf("chain = %s %v", opener.nextNodeID, opener.remaining)
	}
	if opener.targetHost != "10.0.0.9" || opener.targetPort != 22 {
		t.Fatalf("target = %s:%d", opener.targetHost, opener.targetPort)
	}
}

func listenTCPAccess(t *testing.T, server *Server) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		_ = server.Serve(listener)
	}()
	return listener
}

func listenEcho(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go echoConn(conn)
		}
	}()
	return listener
}

func echoConn(conn net.Conn) {
	defer conn.Close()
	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if n > 0 {
			if _, writeErr := conn.Write(buffer[:n]); writeErr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

func dialTCPAccess(t *testing.T, addr string, frame AuthFrame) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.NewEncoder(conn).Encode(frame); err != nil {
		t.Fatal(err)
	}
	response := readResponse(t, conn)
	if response.Status != "connected" {
		t.Fatalf("response = %+v", response)
	}
	return conn
}

func readResponse(t *testing.T, conn net.Conn) responseFrame {
	t.Helper()
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	var response responseFrame
	if err := json.Unmarshal([]byte(line), &response); err != nil {
		t.Fatal(err)
	}
	return response
}

func roundTrip(t *testing.T, conn net.Conn, value string) string {
	t.Helper()
	if _, err := io.WriteString(conn, value); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, len(value))
	if _, err := io.ReadFull(conn, buffer); err != nil {
		t.Fatal(err)
	}
	return string(buffer)
}

func portOf(t *testing.T, addr string) int {
	t.Helper()
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	value, err := net.LookupPort("tcp", port)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
