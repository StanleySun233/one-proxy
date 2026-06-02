package proxy

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
)

func (s *Server) forwardDirect(w http.ResponseWriter, req *http.Request) {
	s.forwardHTTP(w, req, nil)
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
	streamConn, err := openDirectFirstStream(req.Context(), s.directStream, s.tunnelRegistry, hop, targetHost, targetPort)
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
