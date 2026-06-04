package proxy

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
)

func (s *Server) forwardReverse(w http.ResponseWriter, req *http.Request, tracker *proxySessionTracker) {
	outbound := req.Clone(req.Context())
	outbound.Header = req.Header.Clone()
	outbound.RequestURI = ""
	outbound.URL = s.reverseURL(req)
	outbound.Host = outbound.URL.Host
	removeHopByHopHeaders(outbound.Header)
	removeReverseAuthCredentials(outbound)
	setForwardedHeaders(outbound, req)
	var uploadBytes int64
	if outbound.Body != nil {
		outbound.Body = countingReadCloser{ReadCloser: outbound.Body, bytes: &uploadBytes}
	}

	transport := &http.Transport{}
	defer transport.CloseIdleConnections()
	tracker.markForward()
	resp, err := transport.RoundTrip(outbound)
	if err != nil {
		tracker.finish(uploadBytes, 0, domain.ProxySessionStatusError, "reverse_forward_failed", "reverse_forward_failed")
		http.Error(w, "reverse_forward_failed", http.StatusBadGateway)
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

func (s *Server) upgradeReverse(w http.ResponseWriter, req *http.Request) {
	targetURL := s.reverseURL(req)
	targetConn, err := dialReverseTarget(targetURL)
	if err != nil {
		http.Error(w, "reverse_connect_failed", http.StatusBadGateway)
		return
	}
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
		http.Error(w, "reverse_upgrade_write_failed", http.StatusBadGateway)
		return
	}
	completeUpgrade(w, outbound, targetConn)
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
