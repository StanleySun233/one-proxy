package proxycommand

import (
	"bufio"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDialCONNECTSendsProxyToken(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	seen := make(chan string, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		req, readErr := http.ReadRequest(bufio.NewReader(conn))
		if readErr != nil {
			seen <- readErr.Error()
			return
		}
		seen <- req.Header.Get("Proxy-Authorization")
		if _, writeErr := io.WriteString(conn, "HTTP/1.1 200 Connection Established\r\n\r\npong"); writeErr != nil {
			return
		}
	}()

	host, portText, _ := net.SplitHostPort(listener.Addr().String())
	t.Setenv("ONEPROXY_TEST_TOKEN", "secret-token")
	conn, err := DialCONNECT(Config{
		EntryHost:      host,
		EntryPort:      mustPort(t, portText),
		TargetHost:     "10.0.0.2",
		TargetPort:     22,
		TokenEnv:       "ONEPROXY_TEST_TOKEN",
		ConnectTimeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	header := <-seen
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("token:secret-token"))
	if header != want {
		t.Fatalf("unexpected auth header %q", header)
	}
	buffer := make([]byte, 4)
	if _, err := io.ReadFull(conn, buffer); err != nil {
		t.Fatal(err)
	}
	if string(buffer) != "pong" {
		t.Fatalf("unexpected tunnel payload %q", string(buffer))
	}
}

func mustPort(t *testing.T, text string) int {
	t.Helper()
	port, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		t.Fatal(err)
	}
	return port
}
