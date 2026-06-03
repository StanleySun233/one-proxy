package proxy

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
)

func (s *Server) tunnelDirect(w http.ResponseWriter, req *http.Request, tracker *proxySessionTracker) {
	targetConn, err := net.Dial("tcp", req.Host)
	if err != nil {
		tracker.finish(0, 0, domain.ProxySessionStatusError, "connect_failed", "connect_failed")
		http.Error(w, "connect_failed", http.StatusBadGateway)
		return
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		targetConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, "hijack_not_supported", "hijack_not_supported")
		http.Error(w, "hijack_not_supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		targetConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, "hijack_failed", "hijack_failed")
		http.Error(w, "hijack_failed", http.StatusInternalServerError)
		return
	}
	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	bridgeTunnelWithMetrics(clientConn, targetConn, targetConn, tracker)
}

func (s *Server) tunnelViaProxy(w http.ResponseWriter, req *http.Request, nextHop domain.Node, tracker *proxySessionTracker) {
	proxyConn, err := net.Dial("tcp", net.JoinHostPort(nextHop.PublicHost, strconv.Itoa(nextHop.PublicPort)))
	if err != nil {
		tracker.finish(0, 0, domain.ProxySessionStatusError, "next_hop_connect_failed", "next_hop_connect_failed")
		http.Error(w, "next_hop_connect_failed", http.StatusBadGateway)
		return
	}
	if _, err := fmt.Fprintf(proxyConn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", req.Host, req.Host); err != nil {
		proxyConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, "next_hop_connect_failed", "next_hop_connect_failed")
		http.Error(w, "next_hop_connect_failed", http.StatusBadGateway)
		return
	}
	reader := bufio.NewReader(proxyConn)
	line, err := reader.ReadString('\n')
	if err != nil || line == "" || len(line) < 12 || line[9:12] != "200" {
		proxyConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, "next_hop_connect_failed", "next_hop_connect_failed")
		http.Error(w, "next_hop_connect_failed", http.StatusBadGateway)
		return
	}
	for {
		headerLine, readErr := reader.ReadString('\n')
		if readErr != nil {
			proxyConn.Close()
			tracker.finish(0, 0, domain.ProxySessionStatusError, "next_hop_connect_failed", "next_hop_connect_failed")
			http.Error(w, "next_hop_connect_failed", http.StatusBadGateway)
			return
		}
		if headerLine == "\r\n" {
			break
		}
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		proxyConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, "hijack_not_supported", "hijack_not_supported")
		http.Error(w, "hijack_not_supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		proxyConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, "hijack_failed", "hijack_failed")
		http.Error(w, "hijack_failed", http.StatusInternalServerError)
		return
	}
	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	bridgeTunnelWithMetrics(clientConn, proxyConn, reader, tracker)
}

func (s *Server) tunnelViaStream(w http.ResponseWriter, req *http.Request, hop chainHop, tracker *proxySessionTracker) {
	targetHost, targetPort := targetAddress(req)
	streamConn, err := openDirectFirstStream(req.Context(), s.directStream, s.tunnelRegistry, hop, targetHost, targetPort)
	if err != nil {
		tracker.finish(0, 0, domain.ProxySessionStatusError, "next_hop_connect_failed", "next_hop_connect_failed")
		http.Error(w, "next_hop_connect_failed", http.StatusBadGateway)
		return
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		streamConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, "hijack_not_supported", "hijack_not_supported")
		http.Error(w, "hijack_not_supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		streamConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, "hijack_failed", "hijack_failed")
		http.Error(w, "hijack_failed", http.StatusInternalServerError)
		return
	}
	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	bridgeTunnelWithMetrics(clientConn, streamConn, streamConn, tracker)
}
