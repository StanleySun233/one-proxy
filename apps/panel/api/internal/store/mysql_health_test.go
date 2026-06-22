package store

import (
	"testing"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func TestHeartbeatNodeStatusDegradesOnDownListener(t *testing.T) {
	got := heartbeatNodeStatus(map[string]string{
		"runtime":                     "up",
		"transport:reverse_ws_parent": domain.ListenerStatusDown,
	}, map[string]string{})
	if got != domain.NodeStatusDegraded {
		t.Fatalf("heartbeatNodeStatus = %q, want %q", got, domain.NodeStatusDegraded)
	}
}

func TestHeartbeatNodeStatusAcceptsConnectedTransport(t *testing.T) {
	got := heartbeatNodeStatus(map[string]string{
		"runtime":               "up",
		"transport:public_http": domain.TransportStatusConnected,
	}, map[string]string{})
	if got != domain.NodeStatusHealthy {
		t.Fatalf("heartbeatNodeStatus = %q, want %q", got, domain.NodeStatusHealthy)
	}
}
