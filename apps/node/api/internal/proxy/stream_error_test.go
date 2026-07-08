package proxy

import (
	"net"
	"testing"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/tunnel"
)

type failingReader struct {
	err error
}

func (r failingReader) Read([]byte) (int, error) {
	return 0, r.err
}

func TestProxyErrorForStreamFailureUsesRelayTunnelUnavailable(t *testing.T) {
	if got := proxyErrorForStreamFailure(tunnel.ErrChildTunnelNotFound); got != proxyErrorRelayTunnelUnavailable {
		t.Fatalf("error code = %q, want %q", got, proxyErrorRelayTunnelUnavailable)
	}
}

func TestBridgeTunnelWithMetricsReportsRelayTunnelUnavailable(t *testing.T) {
	reporter := recordingSessionReporter{sessions: make(chan domain.ProxySessionMetric, 1)}
	started := time.Now().UTC()
	tracker := &proxySessionTracker{
		reporter: reporter,
		started:  started,
		metric: domain.ProxySessionMetric{
			ID:         "session-1",
			NodeID:     "node-1",
			TargetHost: "target.local",
			TargetPort: 443,
			Protocol:   domain.ProxySessionProtocolConnect,
			StartedAt:  started.Format(time.RFC3339Nano),
		},
	}
	clientConn, clientPeer := net.Pipe()
	backendConn, backendPeer := net.Pipe()
	defer clientPeer.Close()
	defer backendPeer.Close()

	bridgeTunnelWithMetrics(clientConn, backendConn, failingReader{err: tunnel.ErrChildTunnelClosed}, tracker)

	session := receiveSession(t, reporter.sessions)
	if session.Status != domain.ProxySessionStatusError {
		t.Fatalf("status = %q, want %q", session.Status, domain.ProxySessionStatusError)
	}
	if session.ErrorCode != proxyErrorRelayTunnelUnavailable {
		t.Fatalf("error code = %q, want %q", session.ErrorCode, proxyErrorRelayTunnelUnavailable)
	}
}
