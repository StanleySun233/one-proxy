package direct

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
)

type fakeSignalingClient struct {
	reported domain.ReportDirectCandidatesInput
	plan     domain.DirectLinkPlan
}

func (c *fakeSignalingClient) ReportDirectCandidates(input domain.ReportDirectCandidatesInput) (domain.ReportDirectCandidatesResult, error) {
	c.reported = input
	return domain.ReportDirectCandidatesResult{CandidateCount: len(input.Candidates)}, nil
}

func (c *fakeSignalingClient) FetchDirectLinkPlan() (domain.DirectLinkPlan, error) {
	return c.plan, nil
}

func (c *fakeSignalingClient) ReportDirectStatus(input domain.ReportDirectStatusInput) (domain.ReportDirectStatusResult, error) {
	return domain.ReportDirectStatusResult{LinkID: input.LinkID, PeerNodeID: input.PeerNodeID, Status: input.Status}, nil
}

func TestManagerRefreshReportsCandidatesAndAppliesPlan(t *testing.T) {
	server, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	go func() {
		buffer := make([]byte, 1500)
		n, addr, err := server.ReadFromUDP(buffer)
		if err != nil {
			return
		}
		_, _ = server.WriteToUDP(stunResponse(buffer[:n], net.IPv4(203, 0, 113, 8), 45124), addr)
	}()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := &fakeSignalingClient{
		plan: domain.DirectLinkPlan{
			NodeID: "node-a",
			Links:  []domain.DirectLinkItem{{LinkID: "link-1", PeerNodeID: "node-b"}},
		},
	}
	manager := NewManager(UDPConnPacketIO{Conn: conn}, CandidateGatherer{STUNServers: []string{server.LocalAddr().String()}}, client, nil)
	manager.now = func() time.Time { return time.Unix(1, 0) }
	if err := manager.RefreshOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if client.reported.NATType != domain.NATTypeEndpointIndependent || client.reported.UDPListenPort == 0 {
		t.Fatalf("unexpected candidate report: %#v", client.reported)
	}
	state, ok := manager.Registry().Get("node-b")
	if !ok || state.Status != domain.DirectStatusProbing {
		t.Fatalf("unexpected peer state: %#v ok=%v", state, ok)
	}
}

func TestRegistryOpenStreamRequiresConnectedPeer(t *testing.T) {
	registry := NewRegistry()
	if err := registry.OpenStream("node-b"); err == nil {
		t.Fatal("expected missing peer error")
	}
	registry.Upsert(PeerState{PeerNodeID: "node-b", Status: domain.DirectStatusProbing})
	if err := registry.OpenStream("node-b"); err == nil {
		t.Fatal("expected disconnected peer error")
	}
	registry.Upsert(PeerState{PeerNodeID: "node-b", Status: domain.DirectStatusConnected})
	if err := registry.OpenStream("node-b"); err == nil {
		t.Fatal("expected quic not ready error")
	}
}
