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

const (
	directProbeWindow   = 8 * time.Second
	directProbeInterval = 200 * time.Millisecond
	directReadSlice     = 250 * time.Millisecond
)

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
		now := m.now().UTC()
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
			state.LastProbeAt = now
			m.reportDirectStatus(link, state)
		} else if candidate, rtt, reason, ok := m.probePeer(ctx, plan.NodeID, link); ok {
			state.Status = domain.DirectStatusConnected
			state.SelectedCandidate = candidate
			state.RTT = rtt
			state.LastProbeAt = now
			m.reportDirectStatus(link, state)
		} else if current, ok := m.registry.Get(link.PeerNodeID); ok && keepCurrentDirectState(current, link) {
			state = current
		} else {
			state.Status = domain.DirectStatusFailed
			state.FallbackReason = reason
			state.LastProbeAt = now
			m.reportDirectStatus(link, state)
		}
		m.registry.Upsert(state)
	}
}

func validDirectIdentity(identity domain.DirectNodeIdentity) bool {
	return identity.NodeID != "" && identity.ServerName != "" && identity.CertificateFingerprintSHA256 != "" && identity.TrustMaterial != ""
}

func (m *Manager) reportDirectStatus(link domain.DirectLinkItem, state PeerState) {
	if m.client == nil {
		return
	}
	_, _ = m.client.ReportDirectStatus(domain.ReportDirectStatusInput{
		LinkID:            link.LinkID,
		PeerNodeID:        link.PeerNodeID,
		TransportType:     domain.TransportTypeDirectQUIC,
		Status:            state.Status,
		SelectedCandidate: state.SelectedCandidate,
		RTTMs:             int(state.RTT / time.Millisecond),
		LastProbeAt:       state.LastProbeAt.Format(time.RFC3339),
		FallbackReason:    state.FallbackReason,
	})
}

func keepCurrentDirectState(current PeerState, link domain.DirectLinkItem) bool {
	return current.Status == domain.DirectStatusConnected &&
		validDirectIdentity(current.PeerIdentity) &&
		candidateListContains(link.PeerCandidates, current.SelectedCandidate)
}

func (m *Manager) probePeer(ctx context.Context, nodeID string, link domain.DirectLinkItem) (domain.DirectCandidate, time.Duration, string, bool) {
	targets := directProbeTargets(link.PeerCandidates)
	if len(targets) == 0 {
		return domain.DirectCandidate{}, 0, "direct_candidates_unavailable", false
	}
	startedAt := m.now()
	deadline := time.Now().Add(directProbeWindow)
	nextSend := time.Now()
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return domain.DirectCandidate{}, 0, "probe_canceled", false
		default:
		}
		if time.Now().After(nextSend) || time.Now().Equal(nextSend) {
			sendDirectPunches(m.packetIO, targets, link, nodeID)
			nextSend = time.Now().Add(directProbeInterval)
		}
		result, ok := m.awaitDirectPunch(ctx, link, nodeID)
		if !ok {
			continue
		}
		_ = SendPunch(m.packetIO, result.Addr, NewPunchMessage(link.LinkID, nodeID, link.PeerNodeID, link.PunchToken, result.Message.Nonce, m.now()))
		return domain.DirectCandidate{
			Type:     domain.CandidateTypeServerReflexive,
			Address:  result.Addr.IP.String(),
			Port:     result.Addr.Port,
			Protocol: domain.CandidateProtocolUDP,
		}, m.now().Sub(startedAt), "", true
	}
	return domain.DirectCandidate{}, 0, "punch_timeout", false
}

func (m *Manager) awaitDirectPunch(ctx context.Context, link domain.DirectLinkItem, nodeID string) (PunchResult, bool) {
	readCtx, cancel := context.WithTimeout(ctx, directReadSlice)
	defer cancel()
	result, err := AwaitPunch(readCtx, m.packetIO, func(message PunchMessage, _ *net.UDPAddr) bool {
		return message.LinkID == link.LinkID &&
			message.NodeID == link.PeerNodeID &&
			message.PeerNodeID == nodeID &&
			message.PunchToken == link.PunchToken
	})
	return result, err == nil
}

type directProbeTarget struct {
	addr *net.UDPAddr
}

func directProbeTargets(candidates []domain.DirectCandidate) []directProbeTarget {
	targets := make([]directProbeTarget, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Protocol != domain.CandidateProtocolUDP || candidate.Address == "" || candidate.Port <= 0 {
			continue
		}
		addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(candidate.Address, strconv.Itoa(candidate.Port)))
		if err != nil {
			continue
		}
		targets = append(targets, directProbeTarget{addr: addr})
	}
	return targets
}

func sendDirectPunches(packetIO PacketIO, targets []directProbeTarget, link domain.DirectLinkItem, nodeID string) {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)
	message := NewPunchMessage(link.LinkID, nodeID, link.PeerNodeID, link.PunchToken, nonce, time.Now())
	for _, target := range targets {
		_ = SendPunch(packetIO, target.addr, message)
	}
}

func candidateListContains(candidates []domain.DirectCandidate, selected domain.DirectCandidate) bool {
	for _, candidate := range candidates {
		if candidate.Address == selected.Address && candidate.Port == selected.Port && candidate.Protocol == selected.Protocol {
			return true
		}
	}
	return false
}
