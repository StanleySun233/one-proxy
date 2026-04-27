package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/policystore"
	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/tunnel"
)

type Server struct {
	store          *policystore.Store
	nodeIDGetter   func() string
	tunnelRegistry *tunnel.Registry
}

type chainHop struct {
	node          domain.Node
	remainingHops []string
	isLast        bool
}

func NewServer(store *policystore.Store, nodeIDGetter func() string, tunnelRegistry *tunnel.Registry) *Server {
	return &Server{store: store, nodeIDGetter: nodeIDGetter, tunnelRegistry: tunnelRegistry}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	_, snapshot := s.store.Current()
	match := Match(snapshot, req)
	if !match.Found {
		http.Error(w, "route_not_found", http.StatusForbidden)
		return
	}
	switch match.Rule.ActionType {
	case "direct":
		if req.Method == http.MethodConnect {
			s.tunnelDirect(w, req)
			return
		}
		s.forwardDirect(w, req)
	case "chain":
		s.forwardChain(w, req, snapshot, match.Rule)
	default:
		http.Error(w, "unsupported_route_action", http.StatusBadRequest)
	}
}

func (s *Server) forwardDirect(w http.ResponseWriter, req *http.Request) {
	target := &url.URL{
		Scheme: req.URL.Scheme,
		Host:   req.URL.Host,
	}
	if target.Scheme == "" {
		target.Scheme = "http"
	}
	if target.Host == "" {
		target.Host = req.Host
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(out *http.Request) {
		originalDirector(out)
		out.Host = target.Host
		out.URL.Path = req.URL.Path
		out.URL.RawQuery = req.URL.RawQuery
	}
	proxy.ServeHTTP(w, req)
}

func (s *Server) forwardChain(w http.ResponseWriter, req *http.Request, snapshot policystore.Snapshot, rule domain.RouteRule) {
	hop, ok := s.resolveChainHop(snapshot, rule.ChainID)
	if !ok {
		http.Error(w, "invalid_chain_route", http.StatusBadGateway)
		return
	}
	if hop.isLast {
		if req.Method == http.MethodConnect {
			s.tunnelDirect(w, req)
			return
		}
		s.forwardDirect(w, req)
		return
	}
	if s.shouldUseTunnel(hop.node) {
		if req.Method == http.MethodConnect {
			s.tunnelViaStream(w, req, hop)
			return
		}
		s.forwardViaStream(w, req, hop)
		return
	}
	if hop.node.PublicHost == "" || hop.node.PublicPort <= 0 {
		http.Error(w, "next_hop_unreachable", http.StatusBadGateway)
		return
	}
	if req.Method == http.MethodConnect {
		s.tunnelViaProxy(w, req, hop.node)
		return
	}
	s.forwardViaProxy(w, req, hop.node)
}

func (s *Server) shouldUseTunnel(nextHop domain.Node) bool {
	return s.tunnelRegistry != nil && s.tunnelRegistry.HasChild(nextHop.ID)
}

func (s *Server) forwardViaProxy(w http.ResponseWriter, req *http.Request, nextHop domain.Node) {
	proxyURL := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(nextHop.PublicHost, strconv.Itoa(nextHop.PublicPort)),
	}
	target := &url.URL{
		Scheme: req.URL.Scheme,
		Host:   req.URL.Host,
	}
	if target.Scheme == "" {
		target.Scheme = "http"
	}
	if target.Host == "" {
		target.Host = req.Host
	}
	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := reverseProxy.Director
	reverseProxy.Transport = &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	reverseProxy.Director = func(out *http.Request) {
		originalDirector(out)
		out.Host = target.Host
		out.URL.Path = req.URL.Path
		out.URL.RawQuery = req.URL.RawQuery
	}
	reverseProxy.ServeHTTP(w, req)
}

func (s *Server) forwardViaStream(w http.ResponseWriter, req *http.Request, hop chainHop) {
	targetHost, targetPort := targetAddress(req)
	streamConn, err := s.tunnelRegistry.OpenStream(hop.node.ID, hop.remainingHops, targetHost, targetPort)
	if err != nil {
		http.Error(w, "next_hop_connect_failed", http.StatusBadGateway)
		return
	}
	defer streamConn.Close()
	outbound := req.Clone(req.Context())
	outbound.RequestURI = ""
	if outbound.URL == nil {
		outbound.URL = &url.URL{}
	}
	outbound.URL.Scheme = ""
	outbound.URL.Host = ""
	if err := outbound.Write(streamConn); err != nil {
		http.Error(w, "stream_write_failed", http.StatusBadGateway)
		return
	}
	reader := bufio.NewReader(streamConn)
	resp, err := http.ReadResponse(reader, outbound)
	if err != nil {
		http.Error(w, "stream_response_failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

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

func (s *Server) resolveChainHop(snapshot policystore.Snapshot, chainID string) (chainHop, bool) {
	var chain domain.Chain
	found := false
	for _, item := range snapshot.Chains {
		if item.ID == chainID {
			chain = item
			found = true
			break
		}
	}
	if !found || len(chain.Hops) == 0 {
		return chainHop{}, false
	}
	index := -1
	nodeID := s.nodeIDGetter()
	for i, hop := range chain.Hops {
		if hop == nodeID {
			index = i
			break
		}
	}
	if index == -1 {
		return chainHop{}, false
	}
	if index == len(chain.Hops)-1 {
		return chainHop{isLast: true}, true
	}
	nextHopID := chain.Hops[index+1]
	for _, node := range snapshot.Nodes {
		if node.ID == nextHopID {
			return chainHop{
				node:          node,
				remainingHops: append([]string(nil), chain.Hops[index+2:]...),
			}, true
		}
	}
	return chainHop{}, false
}

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
