package proxy

import (
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
)

func (s *Server) forwardReverse(w http.ResponseWriter, req *http.Request, tracker *proxySessionTracker) {
	body, err := readForwardRequestBody(req)
	if err != nil {
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorReverseForwardFailed, proxyErrorReverseForwardFailed)
		writeProxyError(w, req, proxyErrorReverseForwardFailed, http.StatusBadGateway)
		return
	}
	uploadBytes := int64(len(body))

	transport := &http.Transport{}
	defer transport.CloseIdleConnections()
	tracker.markForward()
	resp, err := s.roundTripReverseWithRetry(transport, req, body)
	if err != nil {
		tracker.finish(uploadBytes, 0, domain.ProxySessionStatusError, proxyErrorReverseForwardFailed, proxyErrorReverseForwardFailed)
		writeProxyError(w, req, proxyErrorReverseForwardFailed, http.StatusBadGateway)
		return
	}
	tracker.markResponseReceive()
	downloadBytes := writeForwardResponse(w, resp)
	tracker.finish(uploadBytes, downloadBytes, domain.ProxySessionStatusOK, "", "")
}

func (s *Server) roundTripReverseWithRetry(transport *http.Transport, req *http.Request, body []byte) (forwardResponse, error) {
	attempts := 1 + len(forwardRetryBackoffs)
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			time.Sleep(forwardRetryBackoffs[attempt-1])
		}
		outbound := s.newReverseRequest(req, body)
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

func (s *Server) newReverseRequest(req *http.Request, body []byte) *http.Request {
	outbound := req.Clone(req.Context())
	outbound.Header = req.Header.Clone()
	outbound.RequestURI = ""
	outbound.URL = s.reverseURL(req)
	outbound.Host = outbound.URL.Host
	removeHopByHopHeaders(outbound.Header)
	removeReverseAuthCredentials(outbound)
	setForwardedHeaders(outbound, req)
	if body != nil {
		outbound.Body = io.NopCloser(bytes.NewReader(body))
		outbound.ContentLength = int64(len(body))
	} else {
		outbound.Body = nil
		outbound.ContentLength = 0
	}
	return outbound
}

func (s *Server) upgradeReverse(w http.ResponseWriter, req *http.Request, tracker *proxySessionTracker) {
	targetURL := s.reverseURL(req)
	targetConn, err := dialReverseTarget(targetURL)
	if err != nil {
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorReverseConnectFailed, proxyErrorReverseConnectFailed)
		writeProxyError(w, req, proxyErrorReverseConnectFailed, http.StatusBadGateway)
		return
	}
	tracker.markForward()
	outbound := req.Clone(req.Context())
	outbound.Header = req.Header.Clone()
	outbound.RequestURI = ""
	outbound.URL = targetURL
	outbound.URL.Scheme = ""
	outbound.URL.Host = ""
	outbound.Host = s.reverseTarget.Host
	outbound.Header.Del("Proxy-Connection")
	removeReverseAuthCredentials(outbound)
	setForwardedHeaders(outbound, req)
	rewriteOrigin(outbound, s.reverseTarget)
	if err := outbound.Write(targetConn); err != nil {
		targetConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorReverseUpgradeWriteFailed, proxyErrorReverseUpgradeWriteFailed)
		writeProxyError(w, req, proxyErrorReverseUpgradeWriteFailed, http.StatusBadGateway)
		return
	}
	completeUpgrade(w, outbound, targetConn, tracker)
}

func (s *Server) reverseURL(req *http.Request) *url.URL {
	target := *s.reverseTarget
	if req.URL == nil {
		return &target
	}
	target.Path = joinURLPath(s.reverseTarget.Path, req.URL.Path)
	target.RawPath = ""
	target.RawQuery = joinRawQuery(s.reverseTarget.RawQuery, reverseRawQuery(req.URL))
	target.Fragment = ""
	return &target
}

func reverseRawQuery(url *url.URL) string {
	query := url.Query()
	query.Del(reverseQueryTokenKey)
	return query.Encode()
}

func removeReverseAuthCredentials(req *http.Request) {
	req.Header.Del(reverseHeaderName)
	cookies := req.Cookies()
	if len(cookies) == 0 {
		return
	}
	values := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		if cookie.Name != reverseCookieName {
			values = append(values, cookie.String())
		}
	}
	req.Header.Del("Cookie")
	for _, value := range values {
		req.Header.Add("Cookie", value)
	}
}

func dialReverseTarget(target *url.URL) (net.Conn, error) {
	address := target.Host
	if _, _, err := net.SplitHostPort(address); err != nil {
		switch target.Scheme {
		case "https", "wss":
			address = net.JoinHostPort(address, "443")
		default:
			address = net.JoinHostPort(address, "80")
		}
	}
	if target.Scheme == "https" || target.Scheme == "wss" {
		return tls.Dial("tcp", address, &tls.Config{ServerName: target.Hostname()})
	}
	return net.Dial("tcp", address)
}

func setForwardedHeaders(outbound *http.Request, original *http.Request) {
	if original.Host != "" {
		outbound.Header.Set("X-Forwarded-Host", original.Host)
	}
	if original.RemoteAddr != "" {
		if host, _, err := net.SplitHostPort(original.RemoteAddr); err == nil {
			outbound.Header.Set("X-Forwarded-For", host)
		}
	}
	if original.TLS != nil {
		outbound.Header.Set("X-Forwarded-Proto", "https")
		return
	}
	outbound.Header.Set("X-Forwarded-Proto", "http")
}

func rewriteOrigin(outbound *http.Request, target *url.URL) {
	if outbound.Header.Get("Origin") == "" {
		return
	}
	scheme := target.Scheme
	if scheme == "wss" {
		scheme = "https"
	}
	if scheme == "ws" {
		scheme = "http"
	}
	outbound.Header.Set("Origin", scheme+"://"+target.Host)
}

func joinURLPath(base string, requestPath string) string {
	if base == "" || base == "/" {
		if requestPath == "" {
			return "/"
		}
		return requestPath
	}
	if requestPath == "" || requestPath == "/" {
		return base
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(requestPath, "/")
}

func joinRawQuery(base string, requestQuery string) string {
	if base == "" {
		return requestQuery
	}
	if requestQuery == "" {
		return base
	}
	return base + "&" + requestQuery
}
