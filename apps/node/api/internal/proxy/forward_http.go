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
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
)

var forwardRetryBackoffs = []time.Duration{100 * time.Millisecond, 250 * time.Millisecond}

type forwardResponse struct {
	statusCode int
	header     http.Header
	body       []byte
	stream     io.ReadCloser
}

type closingReadCloser struct {
	io.ReadCloser
	close func() error
}

func (c closingReadCloser) Close() error {
	closeErr := c.close()
	err := c.ReadCloser.Close()
	if err != nil {
		return err
	}
	return closeErr
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
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorForwardFailed, proxyErrorForwardFailed)
		writeProxyError(w, req, proxyErrorForwardFailed, http.StatusBadGateway)
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
		tracker.finish(uploadBytes, 0, domain.ProxySessionStatusError, proxyErrorForwardFailed, proxyErrorForwardFailed)
		writeProxyError(w, req, proxyErrorForwardFailed, http.StatusBadGateway)
		return
	}
	tracker.markResponseReceive()
	downloadBytes := writeForwardResponse(w, resp)
	tracker.finish(uploadBytes, downloadBytes, domain.ProxySessionStatusOK, "", "")
}

func roundTripWithRetry(transport *http.Transport, req *http.Request, body []byte) (forwardResponse, error) {
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
		if attempt+1 < attempts && retryableForwardStatus(resp.StatusCode) {
			_ = resp.Body.Close()
			continue
		}
		forwarded, err := readForwardResponse(resp, outbound.Method)
		if err != nil {
			_ = resp.Body.Close()
			lastErr = err
			continue
		}
		if forwarded.stream == nil {
			_ = resp.Body.Close()
		}
		return forwarded, nil
	}
	return forwardResponse{}, lastErr
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

func readForwardResponse(resp *http.Response, method string) (forwardResponse, error) {
	if shouldStreamForwardResponse(resp, method) {
		return forwardResponse{
			statusCode: resp.StatusCode,
			header:     resp.Header.Clone(),
			stream:     resp.Body,
		}, nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return forwardResponse{}, err
	}
	if method != http.MethodHead && resp.ContentLength >= 0 && resp.ContentLength != int64(len(body)) {
		return forwardResponse{}, errors.New("response_content_length_mismatch")
	}
	return forwardResponse{
		statusCode: resp.StatusCode,
		header:     resp.Header.Clone(),
		body:       body,
	}, nil
}

func shouldStreamForwardResponse(resp *http.Response, method string) bool {
	return method != http.MethodHead && strings.HasPrefix(strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Type"))), "text/event-stream")
}

func writeForwardResponse(w http.ResponseWriter, resp forwardResponse) int64 {
	header := resp.header.Clone()
	removeHopByHopHeaders(header)
	for key, values := range header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.statusCode)
	if resp.stream == nil {
		n, _ := w.Write(resp.body)
		return int64(n)
	}
	defer resp.stream.Close()
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	return copyForwardStream(w, resp.stream)
}

func copyForwardStream(w http.ResponseWriter, stream io.Reader) int64 {
	flusher, _ := w.(http.Flusher)
	buffer := make([]byte, 32*1024)
	var total int64
	for {
		n, readErr := stream.Read(buffer)
		if n > 0 {
			written, writeErr := w.Write(buffer[:n])
			total += int64(written)
			if flusher != nil {
				flusher.Flush()
			}
			if writeErr != nil || written != n {
				return total
			}
		}
		if readErr != nil {
			return total
		}
	}
}

func retryableForwardStatus(statusCode int) bool {
	return statusCode == http.StatusBadGateway || statusCode == http.StatusServiceUnavailable || statusCode == http.StatusGatewayTimeout
}

func (s *Server) forwardViaStream(w http.ResponseWriter, req *http.Request, hop chainHop, tracker *proxySessionTracker) {
	targetHost, targetPort := targetAddress(req)
	body, err := readForwardRequestBody(req)
	if err != nil {
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorStreamResponseFailed, proxyErrorStreamResponseFailed)
		writeProxyError(w, req, proxyErrorStreamResponseFailed, http.StatusBadGateway)
		return
	}
	uploadBytes := int64(len(body))
	tracker.markForward()
	streamStarted := time.Now().UTC()
	resp, err := s.roundTripStreamWithRetry(req, hop, targetHost, targetPort, body)
	tracker.addLinkTiming(s.nodeIDGetter(), hop.node.ID, streamStarted)
	if err != nil {
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorNextHopConnectFailed, proxyErrorNextHopConnectFailed)
		writeProxyError(w, req, proxyErrorNextHopConnectFailed, http.StatusBadGateway)
		return
	}
	tracker.markResponseReceive()
	downloadBytes := writeForwardResponse(w, resp)
	tracker.finish(uploadBytes, downloadBytes, domain.ProxySessionStatusOK, "", "")
}

func (s *Server) roundTripStreamWithRetry(req *http.Request, hop chainHop, targetHost string, targetPort int, body []byte) (forwardResponse, error) {
	attempts := 1 + len(forwardRetryBackoffs)
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			time.Sleep(forwardRetryBackoffs[attempt-1])
		}
		streamConn, err := openDirectFirstStream(req.Context(), s.directStream, s.tunnelRegistry, hop, targetHost, targetPort)
		if err != nil {
			lastErr = err
			continue
		}
		outbound := newStreamForwardRequest(req, body)
		if err := outbound.Write(streamConn); err != nil {
			_ = streamConn.Close()
			lastErr = err
			continue
		}
		reader := bufio.NewReader(streamConn)
		resp, err := http.ReadResponse(reader, outbound)
		if err != nil {
			_ = streamConn.Close()
			lastErr = err
			continue
		}
		if attempt+1 < attempts && retryableForwardStatus(resp.StatusCode) {
			_ = resp.Body.Close()
			_ = streamConn.Close()
			continue
		}
		forwarded, err := readForwardResponse(resp, outbound.Method)
		if err != nil {
			_ = resp.Body.Close()
			_ = streamConn.Close()
			lastErr = err
			continue
		}
		if forwarded.stream != nil {
			forwarded.stream = closingReadCloser{ReadCloser: forwarded.stream, close: streamConn.Close}
			return forwarded, nil
		}
		_ = resp.Body.Close()
		_ = streamConn.Close()
		return forwarded, nil
	}
	return forwardResponse{}, lastErr
}

func newStreamForwardRequest(req *http.Request, body []byte) *http.Request {
	outbound := req.Clone(req.Context())
	outbound.RequestURI = ""
	if outbound.URL == nil {
		outbound.URL = &url.URL{}
	}
	outbound.URL.Scheme = ""
	outbound.URL.Host = ""
	if body != nil {
		outbound.Body = io.NopCloser(bytes.NewReader(body))
		outbound.ContentLength = int64(len(body))
	} else {
		outbound.Body = nil
		outbound.ContentLength = 0
	}
	return outbound
}
