package proxy

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
)

var forwardRetryBackoffs = []time.Duration{100 * time.Millisecond, 250 * time.Millisecond}

type bufferedForwardResponse struct {
	statusCode int
	header     http.Header
	body       []byte
}

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
	targetHost, _ := targetAddress(req)
	body, err := readForwardRequestBody(req)
	if err != nil {
		tracker.finish(0, 0, domain.ProxySessionStatusError, "forward_failed", "forward_failed")
		http.Error(w, "forward_failed", http.StatusBadGateway)
		return
	}
	uploadBytes := int64(len(body))
	transport := &http.Transport{}
	defer transport.CloseIdleConnections()
	if proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	tracker.markForward()
	roundTripStarted := time.Now().UTC()
	resp, err := roundTripWithRetry(transport, req, body)
	timingTarget := nextHopID
	if timingTarget == "" {
		timingTarget = targetHost
	}
	if timingTarget != "" {
		tracker.addLinkTiming(s.nodeIDGetter(), timingTarget, roundTripStarted)
	}
	if err != nil {
		tracker.finish(uploadBytes, 0, domain.ProxySessionStatusError, "forward_failed", "forward_failed")
		http.Error(w, "forward_failed", http.StatusBadGateway)
		return
	}
	tracker.markResponseReceive()
	removeHopByHopHeaders(resp.header)
	for key, values := range resp.header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.statusCode)
	_, _ = w.Write(resp.body)
	tracker.finish(uploadBytes, int64(len(resp.body)), domain.ProxySessionStatusOK, "", "")
}

func roundTripWithRetry(transport *http.Transport, req *http.Request, body []byte) (bufferedForwardResponse, error) {
	attempts := 1 + len(forwardRetryBackoffs)
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			time.Sleep(forwardRetryBackoffs[attempt-1])
		}
		outbound := newForwardRequest(req, body)
		resp, err := transport.RoundTrip(outbound)
		if err != nil {
			lastErr = err
			continue
		}
		buffered, err := readForwardResponse(resp, outbound.Method)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if attempt+1 < attempts && retryableForwardStatus(buffered.statusCode) {
			continue
		}
		return buffered, nil
	}
	return bufferedForwardResponse{}, lastErr
}

func readForwardRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil
	}
	return io.ReadAll(req.Body)
}

func newForwardRequest(req *http.Request, body []byte) *http.Request {
	outbound := req.Clone(req.Context())
	outbound.RequestURI = ""
	outbound.URL = cloneTargetURL(req)
	outbound.Host = outbound.URL.Host
	removeHopByHopHeaders(outbound.Header)
	if body != nil {
		outbound.Body = io.NopCloser(bytes.NewReader(body))
		outbound.ContentLength = int64(len(body))
	} else {
		outbound.Body = nil
		outbound.ContentLength = 0
	}
	return outbound
}

func readForwardResponse(resp *http.Response, method string) (bufferedForwardResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return bufferedForwardResponse{}, err
	}
	if method != http.MethodHead && resp.ContentLength >= 0 && resp.ContentLength != int64(len(body)) {
		return bufferedForwardResponse{}, errors.New("response_content_length_mismatch")
	}
	return bufferedForwardResponse{
		statusCode: resp.StatusCode,
		header:     resp.Header.Clone(),
		body:       body,
	}, nil
}

func retryableForwardStatus(statusCode int) bool {
	return statusCode == http.StatusBadGateway || statusCode == http.StatusServiceUnavailable || statusCode == http.StatusGatewayTimeout
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
