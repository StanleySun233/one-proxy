package direct

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
)

type SignalingClient interface {
	ReportDirectCandidates(domain.ReportDirectCandidatesInput) (domain.ReportDirectCandidatesResult, error)
	FetchDirectLinkPlan() (domain.DirectLinkPlan, error)
	ReportDirectStatus(domain.ReportDirectStatusInput) (domain.ReportDirectStatusResult, error)
}

type Manager struct {
	packetIO PacketIO
	gatherer CandidateGatherer
	client   SignalingClient
	registry *Registry
	now      func() time.Time
}

func NewManager(packetIO PacketIO, gatherer CandidateGatherer, client SignalingClient, registry *Registry) *Manager {
	if registry == nil {
		registry = NewRegistry()
	}
	return &Manager{
		packetIO: packetIO,
		gatherer: gatherer,
		client:   client,
		registry: registry,
		now:      time.Now,
	}
}

func (m *Manager) Run(ctx context.Context, interval time.Duration, onError func(error)) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	if err := m.RefreshOnce(ctx); err != nil && onError != nil {
		onError(err)
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.RefreshOnce(ctx); err != nil && onError != nil {
				onError(err)
			}
		}
	}
}

func (m *Manager) RefreshOnce(ctx context.Context) error {
	candidates, err := m.gatherer.Gather(ctx, m.packetIO)
	if err != nil {
		return err
	}
	probe := ClassifyNAT(candidates)
	if m.client != nil {
		_, err = m.client.ReportDirectCandidates(domain.ReportDirectCandidatesInput{
			UDPListenPort:  localUDPPort(m.packetIO),
			NATType:        probe.NATType,
			Candidates:     candidates,
			ObservedAt:     m.now().UTC().Format(time.RFC3339),
			DirectIdentity: m.registry.DirectIdentity(),
		})
		if err != nil {
			return err
		}
		plan, err := m.client.FetchDirectLinkPlan()
		if err != nil {
			return err
		}
		m.applyPlan(ctx, plan)
	}
	return nil
}

func (m *Manager) Registry() *Registry {
	return m.registry
}

func (m *Manager) applyPlan(ctx context.Context, plan domain.DirectLinkPlan) {
	for _, link := range plan.Links {
		state := PeerState{
			LinkID:         link.LinkID,
			PeerNodeID:     link.PeerNodeID,
			Status:         domain.DirectStatusProbing,
			PeerIdentity:   link.PeerIdentity,
			FallbackReason: "",
		}
		if !validDirectIdentity(link.PeerIdentity) {
			state.Status = domain.DirectStatusFailed
			state.FallbackReason = "direct_identity_required"
		} else if candidate, rtt, ok := m.probePeer(ctx, plan.NodeID, link); ok {
			state.Status = domain.DirectStatusConnected
			state.SelectedCandidate = candidate
			state.RTT = rtt
			state.LastProbeAt = m.now().UTC()
			if m.client != nil {
				_, _ = m.client.ReportDirectStatus(domain.ReportDirectStatusInput{
					LinkID:            link.LinkID,
					PeerNodeID:        link.PeerNodeID,
					TransportType:     domain.TransportTypeDirectQUIC,
					Status:            domain.DirectStatusConnected,
					SelectedCandidate: candidate,
					RTTMs:             int(rtt / time.Millisecond),
					LastProbeAt:       state.LastProbeAt.Format(time.RFC3339),
				})
			}
		} else if current, ok := m.registry.Get(link.PeerNodeID); ok && current.Status == domain.DirectStatusConnected && validDirectIdentity(current.PeerIdentity) {
			state = current
		}
		m.registry.Upsert(state)
	}
}

func validDirectIdentity(identity domain.DirectNodeIdentity) bool {
	return identity.NodeID != "" && identity.ServerName != "" && identity.CertificateFingerprintSHA256 != "" && identity.TrustMaterial != ""
}

func (m *Manager) probePeer(ctx context.Context, nodeID string, link domain.DirectLinkItem) (domain.DirectCandidate, time.Duration, bool) {
	sent := false
	for _, candidate := range link.PeerCandidates {
		if candidate.Protocol != domain.CandidateProtocolUDP || candidate.Address == "" || candidate.Port <= 0 {
			continue
		}
		addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(candidate.Address, strconv.Itoa(candidate.Port)))
		if err != nil {
			continue
		}
		message := NewPunchMessage(link.LinkID, nodeID, link.PeerNodeID, link.PunchToken, strconv.FormatInt(m.now().UnixNano(), 10), m.now())
		_ = SendPunch(m.packetIO, addr, message)
		sent = true
	}
	if !sent {
		return domain.DirectCandidate{}, 0, false
	}
	deadlineCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	result, err := AwaitPunch(deadlineCtx, m.packetIO, func(message PunchMessage, _ *net.UDPAddr) bool {
		return message.LinkID == link.LinkID && message.NodeID == link.PeerNodeID && message.PeerNodeID == nodeID
	})
	if err != nil {
		return domain.DirectCandidate{}, 0, false
	}
	return domain.DirectCandidate{
		Type:     domain.CandidateTypeServerReflexive,
		Address:  result.Addr.IP.String(),
		Port:     result.Addr.Port,
		Protocol: domain.CandidateProtocolUDP,
	}, result.RTT, true
}
