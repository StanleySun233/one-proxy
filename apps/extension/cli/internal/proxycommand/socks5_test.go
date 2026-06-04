package proxycommand

import (
	"bufio"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestSocks5ConnectUsesAuthenticatedHTTPConnect(t *testing.T) {
	proxyListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer proxyListener.Close()
	seenTarget := make(chan string, 1)
	go func() {
		conn, acceptErr := proxyListener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		req, readErr := http.ReadRequest(bufio.NewReader(conn))
		if readErr != nil {
			seenTarget <- readErr.Error()
			return
		}
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("token:socks-token"))
		if req.Header.Get("Proxy-Authorization") != wantAuth {
			seenTarget <- "bad-auth"
			return
		}
		seenTarget <- req.Host
		_, _ = io.WriteString(conn, "HTTP/1.1 200 Connection Established\r\n\r\nhello")
	}()

	proxyHost, proxyPort, _ := net.SplitHostPort(proxyListener.Addr().String())
	socksListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer socksListener.Close()
	t.Setenv("ONEPROXY_SOCKS_TOKEN", "socks-token")
	go func() {
		conn, acceptErr := socksListener.Accept()
		if acceptErr != nil {
			return
		}
		handleSocks5(conn, Socks5Config{
			EntryHost:      proxyHost,
			EntryPort:      mustPort(t, proxyPort),
			TokenEnv:       "ONEPROXY_SOCKS_TOKEN",
			ConnectTimeout: time.Second,
		})
	}()

	client, err := net.Dial("tcp", socksListener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		t.Fatal(err)
	}
	method := make([]byte, 2)
	if _, err := io.ReadFull(client, method); err != nil {
		t.Fatal(err)
	}
	request := []byte{0x05, 0x01, 0x00, 0x03, 11}
	request = append(request, []byte("example.com")...)
	request = append(request, 0x00, 0x16)
	if _, err := client.Write(request); err != nil {
		t.Fatal(err)
	}
	reply := make([]byte, 10)
	if _, err := io.ReadFull(client, reply); err != nil {
		t.Fatal(err)
	}
	if reply[1] != 0x00 {
		t.Fatalf("reply = %v", reply)
	}
	buffer := make([]byte, 5)
	if _, err := io.ReadFull(client, buffer); err != nil {
		t.Fatal(err)
	}
	if string(buffer) != "hello" {
		t.Fatalf("payload = %q", string(buffer))
	}
	if got := <-seenTarget; got != "example.com:22" {
		t.Fatalf("target = %q", got)
	}
}
