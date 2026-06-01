package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
)

func (s *Server) tunnelDirect(w http.ResponseWriter, req *http.Request) {
	targetConn, err := net.Dial("tcp", req.Host)
	if err != nil {
		http.Error(w, "connect_failed", http.StatusBadGateway)
		return
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		targetConn.Close()
		http.Error(w, "hijack_not_supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		targetConn.Close()
		http.Error(w, "hijack_failed", http.StatusInternalServerError)
		return
	}
	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	bridgeTunnel(clientConn, targetConn)
}

func (s *Server) tunnelViaProxy(w http.ResponseWriter, req *http.Request, nextHop domain.Node) {
	proxyConn, err := net.Dial("tcp", net.JoinHostPort(nextHop.PublicHost, strconv.Itoa(nextHop.PublicPort)))
	if err != nil {
		http.Error(w, "next_hop_connect_failed", http.StatusBadGateway)
		return
	}
	if _, err := fmt.Fprintf(proxyConn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", req.Host, req.Host); err != nil {
		proxyConn.Close()
		http.Error(w, "next_hop_connect_failed", http.StatusBadGateway)
		return
	}
	reader := bufio.NewReader(proxyConn)
	line, err := reader.ReadString('\n')
	if err != nil || line == "" || len(line) < 12 || line[9:12] != "200" {
		proxyConn.Close()
		http.Error(w, "next_hop_connect_failed", http.StatusBadGateway)
		return
	}
	for {
		headerLine, readErr := reader.ReadString('\n')
		if readErr != nil {
			proxyConn.Close()
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
		http.Error(w, "hijack_not_supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		proxyConn.Close()
		http.Error(w, "hijack_failed", http.StatusInternalServerError)
		return
	}
	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	go func() {
		defer clientConn.Close()
		defer proxyConn.Close()
		_, _ = io.Copy(proxyConn, clientConn)
	}()
	go func() {
		defer clientConn.Close()
		defer proxyConn.Close()
		_, _ = io.Copy(clientConn, reader)
	}()
}

func (s *Server) tunnelViaStream(w http.ResponseWriter, req *http.Request, hop chainHop) {
	targetHost, targetPort := targetAddress(req)
	streamConn, err := s.tunnelRegistry.OpenStream(hop.node.ID, hop.remainingHops, targetHost, targetPort)
	if err != nil {
		http.Error(w, "next_hop_connect_failed", http.StatusBadGateway)
		return
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		streamConn.Close()
		http.Error(w, "hijack_not_supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		streamConn.Close()
		http.Error(w, "hijack_failed", http.StatusInternalServerError)
		return
	}
	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	bridgeTunnel(clientConn, streamConn)
}
