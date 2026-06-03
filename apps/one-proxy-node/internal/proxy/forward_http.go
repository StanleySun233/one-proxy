package proxy

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
)

func (s *Server) forwardDirect(w http.ResponseWriter, req *http.Request, tracker *proxySessionTracker) {
	s.forwardHTTP(w, req, nil, "", tracker)
}

func (s *Server) forwardViaProxy(w http.ResponseWriter, req *http.Request, nextHop domain.Node, tracker *proxySessionTracker) {
	proxyURL := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(nextHop.PublicHost, strconv.Itoa(nextHop.PublicPort)),
	}
	s.forwardHTTP(w, req, proxyURL, nextHop.ID, tracker)
}

func (s *Server) forwardHTTP(w http.ResponseWriter, req *http.Request, proxyURL *url.URL, nextHopID string, tracker *proxySessionTracker) {
	outbound := req.Clone(req.Context())
	outbound.RequestURI = ""
	outbound.URL = cloneTargetURL(req)
	outbound.Host = outbound.URL.Host
	removeHopByHopHeaders(outbound.Header)
	var uploadBytes int64
	if outbound.Body != nil {
		outbound.Body = countingReadCloser{ReadCloser: outbound.Body, bytes: &uploadBytes}
	}
	transport := &http.Transport{}
	defer transport.CloseIdleConnections()
	if proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	tracker.markForward()
	roundTripStarted := time.Now().UTC()
	resp, err := transport.RoundTrip(outbound)
	if nextHopID != "" {
		tracker.addLinkTiming(s.nodeIDGetter(), nextHopID, roundTripStarted)
	}
	if err != nil {
		tracker.finish(uploadBytes, 0, domain.ProxySessionStatusError, "forward_failed", "forward_failed")
		http.Error(w, "forward_failed", http.StatusBadGateway)
		return
	}
	tracker.markResponseReceive()
	defer resp.Body.Close()
	removeHopByHopHeaders(resp.Header)
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	var downloadBytes int64
	_, _ = io.Copy(countingWriter{Writer: w, bytes: &downloadBytes}, resp.Body)
	tracker.finish(uploadBytes, downloadBytes, domain.ProxySessionStatusOK, "", "")
}

func (s *Server) forwardViaStream(w http.ResponseWriter, req *http.Request, hop chainHop, tracker *proxySessionTracker) {
	targetHost, targetPort := targetAddress(req)
	tracker.markForward()
	streamStarted := time.Now().UTC()
	streamConn, err := openDirectFirstStream(req.Context(), s.directStream, s.tunnelRegistry, hop, targetHost, targetPort)
	tracker.addLinkTiming(s.nodeIDGetter(), hop.node.ID, streamStarted)
	if err != nil {
		tracker.finish(0, 0, domain.ProxySessionStatusError, "next_hop_connect_failed", "next_hop_connect_failed")
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
	var uploadBytes int64
	if outbound.Body != nil {
		outbound.Body = countingReadCloser{ReadCloser: outbound.Body, bytes: &uploadBytes}
	}
	if err := outbound.Write(streamConn); err != nil {
		tracker.finish(uploadBytes, 0, domain.ProxySessionStatusError, "stream_write_failed", "stream_write_failed")
		http.Error(w, "stream_write_failed", http.StatusBadGateway)
		return
	}
	reader := bufio.NewReader(streamConn)
	resp, err := http.ReadResponse(reader, outbound)
	if err != nil {
		tracker.finish(uploadBytes, 0, domain.ProxySessionStatusError, "stream_response_failed", "stream_response_failed")
		http.Error(w, "stream_response_failed", http.StatusBadGateway)
		return
	}
	tracker.markResponseReceive()
	defer resp.Body.Close()
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	var downloadBytes int64
	_, _ = io.Copy(countingWriter{Writer: w, bytes: &downloadBytes}, resp.Body)
	tracker.finish(uploadBytes, downloadBytes, domain.ProxySessionStatusOK, "", "")
}
