package proxy

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
)

func (s *Server) tunnelDirect(w http.ResponseWriter, req *http.Request, tracker *proxySessionTracker) {
	targetHost, _ := targetAddress(req)
	tracker.markForward()
	dialStarted := time.Now().UTC()
	targetConn, err := net.Dial("tcp", req.Host)
	tracker.addLinkTiming(s.nodeIDGetter(), targetHost, dialStarted)
	if err != nil {
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorConnectFailed, proxyErrorConnectFailed)
		writeProxyError(w, req, proxyErrorConnectFailed, http.StatusBadGateway)
		return
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		targetConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorHijackNotSupported, proxyErrorHijackNotSupported)
		writeProxyError(w, req, proxyErrorHijackNotSupported, http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		targetConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorHijackFailed, proxyErrorHijackFailed)
		writeProxyError(w, req, proxyErrorHijackFailed, http.StatusInternalServerError)
		return
	}
	tracker.markResponseReceive()
	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	bridgeTunnelWithMetrics(clientConn, targetConn, targetConn, tracker)
}

func (s *Server) tunnelViaProxy(w http.ResponseWriter, req *http.Request, nextHop domain.Node, tracker *proxySessionTracker) {
	nextHopAuth := nextHopProxyAuthorization(req)
	if nextHopAuth == "" {
		w.Header().Set("Proxy-Authenticate", `Basic realm="one-proxy"`)
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorProxyAuthRequired, proxyErrorProxyAuthRequired)
		writeProxyError(w, req, proxyErrorProxyAuthRequired, http.StatusProxyAuthRequired)
		return
	}
	tracker.markForward()
	connectStarted := time.Now().UTC()
	proxyConn, err := net.Dial("tcp", net.JoinHostPort(nextHop.PublicHost, strconv.Itoa(nextHop.PublicPort)))
	if err != nil {
		tracker.addLinkTiming(s.nodeIDGetter(), nextHop.ID, connectStarted)
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorNextHopConnectFailed, proxyErrorNextHopConnectFailed)
		writeProxyError(w, req, proxyErrorNextHopConnectFailed, http.StatusBadGateway)
		return
	}
	if _, err := fmt.Fprintf(proxyConn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Authorization: %s\r\n\r\n", req.Host, req.Host, nextHopAuth); err != nil {
		proxyConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorNextHopConnectFailed, proxyErrorNextHopConnectFailed)
		writeProxyError(w, req, proxyErrorNextHopConnectFailed, http.StatusBadGateway)
		return
	}
	reader := bufio.NewReader(proxyConn)
	line, err := reader.ReadString('\n')
	tracker.addLinkTiming(s.nodeIDGetter(), nextHop.ID, connectStarted)
	if err != nil || line == "" || len(line) < 12 || line[9:12] != "200" {
		proxyConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorNextHopConnectFailed, proxyErrorNextHopConnectFailed)
		writeProxyError(w, req, proxyErrorNextHopConnectFailed, http.StatusBadGateway)
		return
	}
	for {
		headerLine, readErr := reader.ReadString('\n')
		if readErr != nil {
			proxyConn.Close()
			tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorNextHopConnectFailed, proxyErrorNextHopConnectFailed)
			writeProxyError(w, req, proxyErrorNextHopConnectFailed, http.StatusBadGateway)
			return
		}
		if headerLine == "\r\n" {
			break
		}
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		proxyConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorHijackNotSupported, proxyErrorHijackNotSupported)
		writeProxyError(w, req, proxyErrorHijackNotSupported, http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		proxyConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorHijackFailed, proxyErrorHijackFailed)
		writeProxyError(w, req, proxyErrorHijackFailed, http.StatusInternalServerError)
		return
	}
	tracker.markResponseReceive()
	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	bridgeTunnelWithMetrics(clientConn, proxyConn, reader, tracker)
}

func (s *Server) tunnelViaStream(w http.ResponseWriter, req *http.Request, hop chainHop, tracker *proxySessionTracker) {
	targetHost, targetPort := targetAddress(req)
	tracker.markForward()
	streamStarted := time.Now().UTC()
	streamConn, err := openDirectFirstStream(req.Context(), s.directStream, s.fallbackStreamOpener(), hop, targetHost, targetPort)
	tracker.addLinkTiming(s.nodeIDGetter(), hop.node.ID, streamStarted)
	if err != nil {
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorNextHopConnectFailed, proxyErrorNextHopConnectFailed)
		writeProxyError(w, req, proxyErrorNextHopConnectFailed, http.StatusBadGateway)
		return
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		streamConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorHijackNotSupported, proxyErrorHijackNotSupported)
		writeProxyError(w, req, proxyErrorHijackNotSupported, http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		streamConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorHijackFailed, proxyErrorHijackFailed)
		writeProxyError(w, req, proxyErrorHijackFailed, http.StatusInternalServerError)
		return
	}
	tracker.markResponseReceive()
	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	bridgeTunnelWithMetrics(clientConn, streamConn, streamConn, tracker)
}

func nextHopProxyAuthorization(req *http.Request) string {
	value := strings.TrimSpace(req.Header.Get("Proxy-Authorization"))
	if value == "" || strings.ContainsAny(value, "\r\n") {
		return ""
	}
	return value
}
