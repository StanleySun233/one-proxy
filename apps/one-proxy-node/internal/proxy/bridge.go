package proxy

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func targetAddress(req *http.Request) (string, int) {
	host := req.Host
	if host == "" && req.URL != nil {
		host = req.URL.Host
	}
	if host == "" {
		return "", 0
	}
	if strings.Contains(host, ":") {
		parsedHost, parsedPort, err := net.SplitHostPort(host)
		if err == nil {
			port, _ := strconv.Atoi(parsedPort)
			return parsedHost, port
		}
	}
	if req.Method == http.MethodConnect || req.TLS != nil {
		return host, 443
	}
	return host, 80
}

func cloneTargetURL(req *http.Request) *url.URL {
	target := *req.URL
	if target.Scheme == "" {
		target.Scheme = "http"
	}
	if target.Host == "" {
		target.Host = req.Host
	}
	return &target
}

func removeHopByHopHeaders(header http.Header) {
	for _, name := range strings.Split(header.Get("Connection"), ",") {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			header.Del(trimmed)
		}
	}
	header.Del("Connection")
	header.Del("Keep-Alive")
	header.Del("Proxy-Authenticate")
	header.Del("Proxy-Authorization")
	header.Del("Proxy-Connection")
	header.Del("Te")
	header.Del("Trailer")
	header.Del("Transfer-Encoding")
	header.Del("Upgrade")
}

func isWebSocketUpgrade(req *http.Request) bool {
	return strings.EqualFold(req.Header.Get("Upgrade"), "websocket") && headerContainsToken(req.Header.Get("Connection"), "upgrade")
}

func headerContainsToken(value string, token string) bool {
	for _, item := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(item), token) {
			return true
		}
	}
	return false
}

func bridgeTunnel(left net.Conn, right net.Conn) {
	go func() {
		defer left.Close()
		defer right.Close()
		_, _ = io.Copy(right, left)
	}()
	go func() {
		defer left.Close()
		defer right.Close()
		_, _ = io.Copy(left, right)
	}()
}

func bridgeUpgraded(clientConn net.Conn, backendConn net.Conn, backendReader io.Reader) {
	go func() {
		defer clientConn.Close()
		defer backendConn.Close()
		_, _ = io.Copy(backendConn, clientConn)
	}()
	go func() {
		defer clientConn.Close()
		defer backendConn.Close()
		_, _ = io.Copy(clientConn, backendReader)
	}()
}
