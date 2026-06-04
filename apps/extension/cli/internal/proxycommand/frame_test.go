package proxycommand

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"testing"
	"time"
)

func TestDialTCPFrameSendsAuthFrame(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	seen := make(chan accessFrame, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		var frame accessFrame
		if decodeErr := json.NewDecoder(bufio.NewReader(conn)).Decode(&frame); decodeErr != nil {
			return
		}
		seen <- frame
		_, _ = io.WriteString(conn, `{"status":"connected"}`+"\n"+"pong")
	}()

	host, portText, _ := net.SplitHostPort(listener.Addr().String())
	t.Setenv("ONEPROXY_FRAME_TOKEN", "frame-token")
	conn, err := DialTCPFrame(Config{
		EntryHost:      host,
		EntryPort:      mustPort(t, portText),
		TargetHost:     "10.0.0.4",
		TargetPort:     3389,
		TokenEnv:       "ONEPROXY_FRAME_TOKEN",
		ConnectTimeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	frame := <-seen
	if frame.Token != "frame-token" || frame.TargetHost != "10.0.0.4" || frame.TargetPort != 3389 {
		t.Fatalf("frame = %+v", frame)
	}
	buffer := make([]byte, 4)
	if _, err := io.ReadFull(conn, buffer); err != nil {
		t.Fatal(err)
	}
	if string(buffer) != "pong" {
		t.Fatalf("payload = %q", string(buffer))
	}
}
