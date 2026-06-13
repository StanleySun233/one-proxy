package proxy

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
)

func (s *Server) upgradeDirect(w http.ResponseWriter, req *http.Request, tracker *proxySessionTracker) {
	targetHost, targetPort := targetAddress(req)
	targetConn, err := net.Dial("tcp", net.JoinHostPort(targetHost, strconv.Itoa(targetPort)))
	if err != nil {
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorConnectFailed, proxyErrorConnectFailed)
		writeProxyError(w, req, proxyErrorConnectFailed, http.StatusBadGateway)
		return
	}
	tracker.markForward()
	if err := writeUpgradeRequest(targetConn, req, false); err != nil {
		targetConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorUpgradeWriteFailed, proxyErrorUpgradeWriteFailed)
		writeProxyError(w, req, proxyErrorUpgradeWriteFailed, http.StatusBadGateway)
		return
	}
	completeUpgrade(w, req, targetConn, tracker)
}

func (s *Server) upgradeViaProxy(w http.ResponseWriter, req *http.Request, nextHop domain.Node, tracker *proxySessionTracker) {
	proxyConn, err := net.Dial("tcp", net.JoinHostPort(nextHop.PublicHost, strconv.Itoa(nextHop.PublicPort)))
	if err != nil {
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorNextHopConnectFailed, proxyErrorNextHopConnectFailed)
		writeProxyError(w, req, proxyErrorNextHopConnectFailed, http.StatusBadGateway)
		return
	}
	tracker.markForward()
	if err := writeUpgradeRequest(proxyConn, req, true); err != nil {
		proxyConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorUpgradeWriteFailed, proxyErrorUpgradeWriteFailed)
		writeProxyError(w, req, proxyErrorUpgradeWriteFailed, http.StatusBadGateway)
		return
	}
	completeUpgrade(w, req, proxyConn, tracker)
}

func (s *Server) upgradeViaStream(w http.ResponseWriter, req *http.Request, hop chainHop, tracker *proxySessionTracker) {
	targetHost, targetPort := targetAddress(req)
	streamConn, err := openDirectFirstStream(req.Context(), s.directStream, s.tunnelRegistry, hop, targetHost, targetPort)
	if err != nil {
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorNextHopConnectFailed, proxyErrorNextHopConnectFailed)
		writeProxyError(w, req, proxyErrorNextHopConnectFailed, http.StatusBadGateway)
		return
	}
	tracker.markForward()
	if err := writeUpgradeRequest(streamConn, req, false); err != nil {
		streamConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorUpgradeWriteFailed, proxyErrorUpgradeWriteFailed)
		writeProxyError(w, req, proxyErrorUpgradeWriteFailed, http.StatusBadGateway)
		return
	}
	completeUpgrade(w, req, streamConn, tracker)
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

func completeUpgrade(w http.ResponseWriter, req *http.Request, backendConn net.Conn, tracker *proxySessionTracker) {
	reader := bufio.NewReader(backendConn)
	resp, err := http.ReadResponse(reader, req)
	if err != nil {
		backendConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorUpgradeResponseFailed, proxyErrorUpgradeResponseFailed)
		writeProxyError(w, req, proxyErrorUpgradeResponseFailed, http.StatusBadGateway)
		return
	}
	tracker.markResponseReceive()
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
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorUpgradeRejected, proxyErrorUpgradeRejected)
		return
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		backendConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorHijackNotSupported, proxyErrorHijackNotSupported)
		writeProxyError(w, req, proxyErrorHijackNotSupported, http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		backendConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorHijackFailed, proxyErrorHijackFailed)
		writeProxyError(w, req, proxyErrorHijackFailed, http.StatusInternalServerError)
		return
	}
	if err := resp.Write(clientConn); err != nil {
		clientConn.Close()
		backendConn.Close()
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorUpgradeWriteFailed, proxyErrorUpgradeWriteFailed)
		return
	}
	tracker.finish(0, 0, domain.ProxySessionStatusOK, "", "")
	bridgeUpgraded(clientConn, backendConn, reader)
}
