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
	statuses []domain.ReportDirectStatusInput
}

func (c *fakeSignalingClient) ReportDirectCandidates(input domain.ReportDirectCandidatesInput) (domain.ReportDirectCandidatesResult, error) {
	c.reported = input
	return domain.ReportDirectCandidatesResult{CandidateCount: len(input.Candidates)}, nil
}

func (c *fakeSignalingClient) FetchDirectLinkPlan() (domain.DirectLinkPlan, error) {
	return c.plan, nil
}

func (c *fakeSignalingClient) ReportDirectStatus(input domain.ReportDirectStatusInput) (domain.ReportDirectStatusResult, error) {
	c.statuses = append(c.statuses, input)
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
			Links:  []domain.DirectLinkItem{{LinkID: "link-1", PeerNodeID: "node-b", PeerIdentity: testDirectIdentity(t, "node-b")}},
		},
	}
	registry := NewRegistry()
	registry.directIdentity = testDirectIdentity(t, "node-a")
	manager := NewManager(UDPConnPacketIO{Conn: conn}, CandidateGatherer{STUNServers: []string{server.LocalAddr().String()}}, client, registry)
	manager.now = func() time.Time { return time.Unix(1, 0) }
	if err := manager.RefreshOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if client.reported.NATType != domain.NATTypeEndpointIndependent || client.reported.UDPListenPort == 0 {
		t.Fatalf("unexpected candidate report: %#v", client.reported)
	}
	if client.reported.DirectIdentity.NodeID != "node-a" || client.reported.DirectIdentity.TrustMaterial == "" {
		t.Fatalf("missing direct identity report: %#v", client.reported.DirectIdentity)
	}
	state, ok := manager.Registry().Get("node-b")
	if !ok || state.Status != domain.DirectStatusFailed || state.FallbackReason != "direct_candidates_unavailable" {
		t.Fatalf("unexpected peer state: %#v ok=%v", state, ok)
	}
	if state.PeerIdentity.NodeID != "node-b" {
		t.Fatalf("missing peer identity: %#v", state.PeerIdentity)
	}
}

func TestManagerRejectsPlanWithoutPeerIdentity(t *testing.T) {
	client := &fakeSignalingClient{}
	manager := NewManager(nil, CandidateGatherer{}, client, NewRegistry())
	manager.applyPlan(context.Background(), domain.DirectLinkPlan{
		NodeID: "node-a",
		Links:  []domain.DirectLinkItem{{LinkID: "link-1", PeerNodeID: "node-b"}},
	})
	state, ok := manager.Registry().Get("node-b")
	if !ok || state.Status != domain.DirectStatusFailed || state.FallbackReason != "direct_identity_required" {
		t.Fatalf("unexpected peer state: %#v ok=%v", state, ok)
	}
	if len(client.statuses) != 1 || client.statuses[0].FallbackReason != "direct_identity_required" {
		t.Fatalf("unexpected status reports: %#v", client.statuses)
	}
}

func TestManagerProbePeerUsesRepeatedPunchAndReply(t *testing.T) {
	left, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer left.Close()
	right, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer right.Close()

	leftManager := NewManager(UDPConnPacketIO{Conn: left}, CandidateGatherer{}, nil, NewRegistry())
	rightManager := NewManager(UDPConnPacketIO{Conn: right}, CandidateGatherer{}, nil, NewRegistry())
	linkAB := domain.DirectLinkItem{
		LinkID:     "link-1",
		PeerNodeID: "node-b",
		PunchToken: "shared-token",
		PeerCandidates: []domain.DirectCandidate{{
			Type:     domain.CandidateTypeHost,
			Address:  "127.0.0.1",
			Port:     right.LocalAddr().(*net.UDPAddr).Port,
			Protocol: domain.CandidateProtocolUDP,
		}},
	}
	linkBA := domain.DirectLinkItem{
		LinkID:     "link-1",
		PeerNodeID: "node-a",
		PunchToken: "shared-token",
		PeerCandidates: []domain.DirectCandidate{{
			Type:     domain.CandidateTypeHost,
			Address:  "127.0.0.1",
			Port:     left.LocalAddr().(*net.UDPAddr).Port,
			Protocol: domain.CandidateProtocolUDP,
		}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	leftResult := make(chan bool, 1)
	rightResult := make(chan bool, 1)
	go func() {
		_, _, _, ok := leftManager.probePeer(ctx, "node-a", linkAB)
		leftResult <- ok
	}()
	go func() {
		_, _, _, ok := rightManager.probePeer(ctx, "node-b", linkBA)
		rightResult <- ok
	}()
	if !<-leftResult || !<-rightResult {
		t.Fatal("expected both peers to connect")
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

func testDirectIdentity(t *testing.T, nodeID string) domain.DirectNodeIdentity {
	t.Helper()
	_, identity, err := serverTLSConfig(nodeID)
	if err != nil {
		t.Fatal(err)
	}
	return identity
}
