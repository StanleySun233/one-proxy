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
