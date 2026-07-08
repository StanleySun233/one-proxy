package proxy

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/responsecache"
)

var forwardRetryBackoffs = []time.Duration{100 * time.Millisecond, 250 * time.Millisecond}

const (
	forwardDialTimeout              = 10 * time.Second
	forwardHeaderTimeout            = 30 * time.Second
	forwardIdleTimeout              = 90 * time.Second
	defaultForwardRetryBodyMaxBytes = 8 * 1024 * 1024
)

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
	s.forwardHTTP(w, req, nil, "", "", tracker)
}

func (s *Server) forwardViaProxy(w http.ResponseWriter, req *http.Request, nextHop domain.Node, tracker *proxySessionTracker) {
	nextHopAuth := nextHopProxyAuthorization(req)
	if nextHopAuth == "" {
		w.Header().Set("Proxy-Authenticate", `Basic realm="one-proxy"`)
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorProxyAuthRequired, proxyErrorProxyAuthRequired)
		writeProxyError(w, req, proxyErrorProxyAuthRequired, http.StatusProxyAuthRequired)
		return
	}
	proxyURL := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(nextHop.PublicHost, strconv.Itoa(nextHop.PublicPort)),
	}
	s.forwardHTTP(w, req, proxyURL, nextHop.ID, nextHopAuth, tracker)
}

func (s *Server) forwardHTTP(w http.ResponseWriter, req *http.Request, proxyURL *url.URL, nextHopID string, nextHopAuth string, tracker *proxySessionTracker) {
	targetHost, _ := targetAddress(req)
	body, bodyStream, err := s.forwardRequestBody(req)
	if err != nil {
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorForwardFailed, proxyErrorForwardFailed)
		writeProxyError(w, req, proxyErrorForwardFailed, http.StatusBadGateway)
		return
	}
	uploadBytes := forwardUploadBytes(req, body)
	transport := newForwardTransport(proxyURL)
	defer transport.CloseIdleConnections()
	tracker.markForward()
	roundTripStarted := time.Now().UTC()
	resp, err := s.roundTripWithRetry(transport, req, body, bodyStream, nextHopAuth)
	timingTarget := nextHopID
	if timingTarget == "" {
		timingTarget = targetHost
	}
	if timingTarget != "" {
		tracker.addLinkTiming(s.nodeIDGetter(), timingTarget, roundTripStarted)
	}
	if err != nil {
		if s.writeCachedResponseOnError(w, req, body, proxyErrorForwardFailed, tracker) {
			return
		}
		log.Printf("proxy forward failed mode=http method=%s target=%s nextHop=%s uploadBytes=%d err=%v", req.Method, requestLogTarget(req), nextHopID, uploadBytes, err)
		tracker.finish(uploadBytes, 0, domain.ProxySessionStatusError, proxyErrorForwardFailed, proxyErrorForwardFailed)
		writeProxyError(w, req, proxyErrorForwardFailed, http.StatusBadGateway)
		return
	}
	tracker.markResponseReceive()
	tracker.markStatusCode(resp.statusCode)
	s.storeResponseCache(req, body, resp)
	downloadBytes := writeForwardResponse(w, resp)
	tracker.finish(uploadBytes, downloadBytes, domain.ProxySessionStatusOK, "", "")
}

func newForwardTransport(proxyURL *url.URL) *http.Transport {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   forwardDialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   forwardDialTimeout,
		ResponseHeaderTimeout: forwardHeaderTimeout,
		ExpectContinueTimeout: forwardDialTimeout,
		IdleConnTimeout:       forwardIdleTimeout,
	}
	if proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	return transport
}

func (s *Server) roundTripWithRetry(transport *http.Transport, req *http.Request, body []byte, bodyStream io.ReadCloser, nextHopAuth string) (forwardResponse, error) {
	attempts := forwardAttemptCount(req.Method, bodyStream == nil)
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			time.Sleep(forwardRetryBackoffs[attempt-1])
		}
		outbound := newForwardRequest(req, body, bodyStream, nextHopAuth)
		resp, err := transport.RoundTrip(outbound)
		if err != nil {
			log.Printf("proxy forward attempt failed mode=http attempt=%d method=%s target=%s err=%v", attempt+1, req.Method, requestLogTarget(req), err)
			lastErr = err
			continue
		}
		if attempt+1 < attempts && retryableForwardStatus(resp.StatusCode) {
			log.Printf("proxy forward retryable status mode=http attempt=%d method=%s target=%s status=%d", attempt+1, req.Method, requestLogTarget(req), resp.StatusCode)
			_ = resp.Body.Close()
			continue
		}
		forwarded, err := s.readForwardResponseForRequest(req, resp, outbound.Method)
		if err != nil {
			log.Printf("proxy forward response read failed mode=http attempt=%d method=%s target=%s status=%d err=%v", attempt+1, req.Method, requestLogTarget(req), resp.StatusCode, err)
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

func (s *Server) forwardRequestBody(req *http.Request) ([]byte, io.ReadCloser, error) {
	if retryableForwardMethod(req.Method) {
		if req.Body == nil || req.Body == http.NoBody {
			return nil, nil, nil
		}
		limit := s.retryBodyMaxBytes
		if limit <= 0 {
			limit = defaultForwardRetryBodyMaxBytes
		}
		if req.ContentLength < 0 || req.ContentLength > limit {
			return nil, req.Body, nil
		}
		body, err := readForwardRequestBody(req, limit)
		return body, nil, err
	}
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil, nil
	}
	return nil, req.Body, nil
}

func readForwardRequestBody(req *http.Request, limit int64) ([]byte, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultForwardRetryBodyMaxBytes
	}
	body, err := io.ReadAll(io.LimitReader(req.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, errors.New("request_body_too_large")
	}
	return body, nil
}

func newForwardRequest(req *http.Request, body []byte, bodyStream io.ReadCloser, nextHopAuth string) *http.Request {
	outbound := req.Clone(req.Context())
	outbound.RequestURI = ""
	outbound.URL = cloneTargetURL(req)
	outbound.Host = outbound.URL.Host
	removeHopByHopHeaders(outbound.Header)
	if nextHopAuth != "" {
		outbound.Header.Set("Proxy-Authorization", nextHopAuth)
	}
	if bodyStream != nil {
		outbound.Body = bodyStream
		outbound.ContentLength = req.ContentLength
	} else if body != nil {
		outbound.Body = io.NopCloser(bytes.NewReader(body))
		outbound.ContentLength = int64(len(body))
	} else {
		outbound.Body = nil
		outbound.ContentLength = 0
	}
	return outbound
}

func forwardUploadBytes(req *http.Request, body []byte) int64 {
	if body != nil {
		return int64(len(body))
	}
	if req.ContentLength > 0 {
		return req.ContentLength
	}
	return 0
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

func (s *Server) readForwardResponseForRequest(req *http.Request, resp *http.Response, method string) (forwardResponse, error) {
	if !s.shouldBufferForwardResponse(req, resp, method) {
		return forwardResponse{
			statusCode: resp.StatusCode,
			header:     resp.Header.Clone(),
			stream:     resp.Body,
		}, nil
	}
	return readForwardResponse(resp, method)
}

func (s *Server) shouldBufferForwardResponse(req *http.Request, resp *http.Response, method string) bool {
	if method == http.MethodHead {
		return true
	}
	if s.responseCache == nil || shouldStreamForwardResponse(resp, method) {
		return false
	}
	if resp.ContentLength < 0 || len(resp.TransferEncoding) > 0 {
		return false
	}
	return responsecache.CanStore(req, resp.StatusCode, resp.Header, int(resp.ContentLength))
}

func shouldStreamForwardResponse(resp *http.Response, method string) bool {
	return method != http.MethodHead && responsecache.IsStreamingContentType(resp.Header.Get("Content-Type"))
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

func forwardAttemptCount(method string, reusableBody bool) int {
	if !reusableBody || !retryableForwardMethod(method) {
		return 1
	}
	return 1 + len(forwardRetryBackoffs)
}

func retryableForwardMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

func (s *Server) forwardViaStream(w http.ResponseWriter, req *http.Request, hop chainHop, tracker *proxySessionTracker) {
	targetHost, targetPort := targetAddress(req)
	body, bodyStream, err := s.forwardRequestBody(req)
	if err != nil {
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorStreamResponseFailed, proxyErrorStreamResponseFailed)
		writeProxyError(w, req, proxyErrorStreamResponseFailed, http.StatusBadGateway)
		return
	}
	uploadBytes := forwardUploadBytes(req, body)
	tracker.markForward()
	streamStarted := time.Now().UTC()
	resp, err := s.roundTripStreamWithRetry(req, hop, targetHost, targetPort, body, bodyStream)
	tracker.addLinkTiming(s.nodeIDGetter(), hop.node.ID, streamStarted)
	if err != nil {
		errorCode := proxyErrorForStreamFailure(err)
		if s.writeCachedResponseOnError(w, req, body, errorCode, tracker) {
			return
		}
		log.Printf("proxy forward failed mode=stream method=%s target=%s nextHop=%s remainingHops=%v uploadBytes=%d errorCode=%s err=%v", req.Method, requestLogTarget(req), hop.node.ID, hop.remainingHops, uploadBytes, errorCode, err)
		tracker.finish(uploadBytes, 0, domain.ProxySessionStatusError, errorCode, errorCode)
		writeProxyError(w, req, errorCode, http.StatusBadGateway)
		return
	}
	tracker.markResponseReceive()
	tracker.markStatusCode(resp.statusCode)
	s.storeResponseCache(req, body, resp)
	downloadBytes := writeForwardResponse(w, resp)
	tracker.finish(uploadBytes, downloadBytes, domain.ProxySessionStatusOK, "", "")
}

func (s *Server) roundTripStreamWithRetry(req *http.Request, hop chainHop, targetHost string, targetPort int, body []byte, bodyStream io.ReadCloser) (forwardResponse, error) {
	attempts := forwardAttemptCount(req.Method, bodyStream == nil)
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			time.Sleep(forwardRetryBackoffs[attempt-1])
		}
		streamConn, err := openDirectFirstStream(req.Context(), s.directStream, s.fallbackStreamOpener(), hop, targetHost, targetPort)
		if err != nil {
			log.Printf("proxy forward attempt failed mode=stream_open attempt=%d method=%s target=%s nextHop=%s remainingHops=%v err=%v", attempt+1, req.Method, requestLogTarget(req), hop.node.ID, hop.remainingHops, err)
			lastErr = err
			continue
		}
		outbound := newStreamForwardRequest(req, body, bodyStream)
		if err := outbound.Write(streamConn); err != nil {
			_ = streamConn.Close()
			log.Printf("proxy forward attempt failed mode=stream_write attempt=%d method=%s target=%s nextHop=%s err=%v", attempt+1, req.Method, requestLogTarget(req), hop.node.ID, err)
			lastErr = err
			continue
		}
		_ = streamConn.SetReadDeadline(time.Now().Add(forwardHeaderTimeout))
		reader := bufio.NewReader(streamConn)
		resp, err := http.ReadResponse(reader, outbound)
		if err != nil {
			_ = streamConn.Close()
			log.Printf("proxy forward attempt failed mode=stream_response attempt=%d method=%s target=%s nextHop=%s err=%v", attempt+1, req.Method, requestLogTarget(req), hop.node.ID, err)
			lastErr = err
			continue
		}
		_ = streamConn.SetReadDeadline(time.Time{})
		if attempt+1 < attempts && retryableForwardStatus(resp.StatusCode) {
			log.Printf("proxy forward retryable status mode=stream attempt=%d method=%s target=%s nextHop=%s status=%d", attempt+1, req.Method, requestLogTarget(req), hop.node.ID, resp.StatusCode)
			_ = resp.Body.Close()
			_ = streamConn.Close()
			continue
		}
		forwarded, err := s.readForwardResponseForRequest(req, resp, outbound.Method)
		if err != nil {
			_ = resp.Body.Close()
			_ = streamConn.Close()
			log.Printf("proxy forward response read failed mode=stream attempt=%d method=%s target=%s nextHop=%s status=%d err=%v", attempt+1, req.Method, requestLogTarget(req), hop.node.ID, resp.StatusCode, err)
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

func newStreamForwardRequest(req *http.Request, body []byte, bodyStream io.ReadCloser) *http.Request {
	outbound := req.Clone(req.Context())
	outbound.RequestURI = ""
	if outbound.URL == nil {
		outbound.URL = &url.URL{}
	}
	outbound.URL.Scheme = ""
	outbound.URL.Host = ""
	if bodyStream != nil {
		outbound.Body = bodyStream
		outbound.ContentLength = req.ContentLength
	} else if body != nil {
		outbound.Body = io.NopCloser(bytes.NewReader(body))
		outbound.ContentLength = int64(len(body))
	} else {
		outbound.Body = nil
		outbound.ContentLength = 0
	}
	return outbound
}
