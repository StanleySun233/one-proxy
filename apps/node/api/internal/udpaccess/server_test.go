package udpaccess

import (
	"context"
	"encoding/json"
	"net"
	"strconv"
	"testing"
	"time"
)

type testAuthorizer struct {
	valid bool
}

func (a testAuthorizer) Validate(_ context.Context, _ string) bool {
	return a.valid
}

func TestUDPAccessRoundTrip(t *testing.T) {
	target := listenUDPEcho(t)
	defer target.Close()

	serverConn := listenUDPAccess(t, New(testAuthorizer{valid: true}))
	defer serverConn.Close()

	response := sendPacket(t, serverConn.LocalAddr().String(), Packet{
		Token:      "udp-token",
		TargetHost: "127.0.0.1",
		TargetPort: portOf(t, target.LocalAddr().String()),
		Data:       []byte("ping"),
	})
	if response.Status != "ok" || string(response.Data) != "ping" {
		t.Fatalf("response = %+v", response)
	}
}

func TestUDPAccessRejectsInvalidToken(t *testing.T) {
	serverConn := listenUDPAccess(t, New(testAuthorizer{valid: false}))
	defer serverConn.Close()

	response := sendPacket(t, serverConn.LocalAddr().String(), Packet{
		Token:      "bad-token",
		TargetHost: "127.0.0.1",
		TargetPort: 53,
		Data:       []byte("ping"),
	})
	if response.Status != "failed" || response.Message != "auth_required" {
		t.Fatalf("response = %+v", response)
	}
}

func TestUDPAccessRejectsOverMaxInFlight(t *testing.T) {
	server := New(testAuthorizer{valid: true})
	server.SetMaxInFlight(1)
	server.inFlight <- struct{}{}
	serverConn := listenUDPAccess(t, server)
	defer serverConn.Close()

	clientConn, err := net.Dial("udp", serverConn.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer clientConn.Close()
	if err := clientConn.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(Packet{Token: "token", TargetHost: "127.0.0.1", TargetPort: 53, Data: []byte("probe")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := clientConn.Write(payload); err != nil {
		t.Fatal(err)
	}
	var response Response
	if err := json.NewDecoder(clientConn).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Status != "failed" || response.Message != "too_many_requests" {
		t.Fatalf("response = %+v", response)
	}
}

func listenUDPEcho(t *testing.T) *net.UDPConn {
	t.Helper()
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		buffer := make([]byte, 1024)
		for {
			n, client, err := conn.ReadFromUDP(buffer)
			if err != nil {
				return
			}
			_, _ = conn.WriteToUDP(buffer[:n], client)
		}
	}()
	return conn
}

func listenUDPAccess(t *testing.T, server *Server) *net.UDPConn {
	t.Helper()
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		_ = server.Serve(conn)
	}()
	return conn
}

func sendPacket(t *testing.T, addr string, packet Packet) Response {
	t.Helper()
	target, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.DialUDP("udp", nil, target)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(time.Second))
	payload, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Write(payload); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 65535)
	n, err := conn.Read(buffer)
	if err != nil {
		t.Fatal(err)
	}
	var response Response
	if err := json.Unmarshal(buffer[:n], &response); err != nil {
		t.Fatal(err)
	}
	return response
}

func portOf(t *testing.T, addr string) int {
	t.Helper()
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	value, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
