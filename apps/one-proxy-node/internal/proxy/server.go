package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
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
	case domain.ActionTypeDirect:
		if isWebSocketUpgrade(req) {
			s.upgradeDirect(w, req)
			return
		}
		if req.Method == http.MethodConnect {
			s.tunnelDirect(w, req)
			return
		}
		s.forwardDirect(w, req)
	case domain.ActionTypeChain:
		s.forwardChain(w, req, snapshot, match.Rule)
	default:
		http.Error(w, "unsupported_route_action", http.StatusBadRequest)
	}
}

func (s *Server) forwardDirect(w http.ResponseWriter, req *http.Request) {
	s.forwardHTTP(w, req, nil)
}

func (s *Server) forwardChain(w http.ResponseWriter, req *http.Request, snapshot policystore.Snapshot, rule domain.RouteRule) {
	hop, ok := s.resolveChainHop(snapshot, rule.ChainID)
	if !ok {
		http.Error(w, "invalid_chain_route", http.StatusBadGateway)
		return
	}
	if hop.isLast {
		if isWebSocketUpgrade(req) {
			s.upgradeDirect(w, req)
			return
		}
		if req.Method == http.MethodConnect {
			s.tunnelDirect(w, req)
			return
		}
		s.forwardDirect(w, req)
		return
	}
	if s.shouldUseTunnel(hop.node) {
		if isWebSocketUpgrade(req) {
			s.upgradeViaStream(w, req, hop)
			return
		}
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
	if isWebSocketUpgrade(req) {
		s.upgradeViaProxy(w, req, hop.node)
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
	s.forwardHTTP(w, req, proxyURL)
}

func (s *Server) forwardHTTP(w http.ResponseWriter, req *http.Request, proxyURL *url.URL) {
	outbound := req.Clone(req.Context())
	outbound.RequestURI = ""
	outbound.URL = cloneTargetURL(req)
	outbound.Host = outbound.URL.Host
	removeHopByHopHeaders(outbound.Header)
	transport := &http.Transport{}
	if proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	resp, err := transport.RoundTrip(outbound)
	if err != nil {
		http.Error(w, "forward_failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	removeHopByHopHeaders(resp.Header)
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
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

func (s *Server) upgradeDirect(w http.ResponseWriter, req *http.Request) {
	targetHost, targetPort := targetAddress(req)
	targetConn, err := net.Dial("tcp", net.JoinHostPort(targetHost, strconv.Itoa(targetPort)))
	if err != nil {
		http.Error(w, "connect_failed", http.StatusBadGateway)
		return
	}
	if err := writeUpgradeRequest(targetConn, req, false); err != nil {
		targetConn.Close()
		http.Error(w, "upgrade_write_failed", http.StatusBadGateway)
		return
	}
	completeUpgrade(w, req, targetConn)
}

func (s *Server) upgradeViaProxy(w http.ResponseWriter, req *http.Request, nextHop domain.Node) {
	proxyConn, err := net.Dial("tcp", net.JoinHostPort(nextHop.PublicHost, strconv.Itoa(nextHop.PublicPort)))
	if err != nil {
		http.Error(w, "next_hop_connect_failed", http.StatusBadGateway)
		return
	}
	if err := writeUpgradeRequest(proxyConn, req, true); err != nil {
		proxyConn.Close()
		http.Error(w, "upgrade_write_failed", http.StatusBadGateway)
		return
	}
	completeUpgrade(w, req, proxyConn)
}

func (s *Server) upgradeViaStream(w http.ResponseWriter, req *http.Request, hop chainHop) {
	targetHost, targetPort := targetAddress(req)
	streamConn, err := s.tunnelRegistry.OpenStream(hop.node.ID, hop.remainingHops, targetHost, targetPort)
	if err != nil {
		http.Error(w, "next_hop_connect_failed", http.StatusBadGateway)
		return
	}
	if err := writeUpgradeRequest(streamConn, req, false); err != nil {
		streamConn.Close()
		http.Error(w, "upgrade_write_failed", http.StatusBadGateway)
		return
	}
	completeUpgrade(w, req, streamConn)
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

func writeUpgradeRequest(conn net.Conn, req *http.Request, absoluteForm bool) error {
	outbound := req.Clone(req.Context())
	outbound.Header = req.Header.Clone()
	outbound.Header.Del("Proxy-Connection")
	outbound.RequestURI = ""
	if absoluteForm {
		outbound.URL = cloneTargetURL(req)
	} else {
		if outbound.URL == nil {
			outbound.URL = &url.URL{}
		}
		outbound.URL.Scheme = ""
		outbound.URL.Host = ""
	}
	return outbound.Write(conn)
}

func completeUpgrade(w http.ResponseWriter, req *http.Request, backendConn net.Conn) {
	reader := bufio.NewReader(backendConn)
	resp, err := http.ReadResponse(reader, req)
	if err != nil {
		backendConn.Close()
		http.Error(w, "upgrade_response_failed", http.StatusBadGateway)
		return
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		defer resp.Body.Close()
		defer backendConn.Close()
		removeHopByHopHeaders(resp.Header)
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
		return
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		backendConn.Close()
		http.Error(w, "hijack_not_supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		backendConn.Close()
		http.Error(w, "hijack_failed", http.StatusInternalServerError)
		return
	}
	if err := resp.Write(clientConn); err != nil {
		clientConn.Close()
		backendConn.Close()
		return
	}
	bridgeUpgraded(clientConn, backendConn, reader)
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
