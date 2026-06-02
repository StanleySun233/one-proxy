package direct

import (
	"context"
	"net"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
)

type SignalingClient interface {
	ReportDirectCandidates(domain.ReportDirectCandidatesInput) (domain.ReportDirectCandidatesResult, error)
	FetchDirectLinkPlan() (domain.DirectLinkPlan, error)
	ReportDirectStatus(domain.ReportDirectStatusInput) (domain.ReportDirectStatusResult, error)
}

type Manager struct {
	conn     *net.UDPConn
	gatherer CandidateGatherer
	client   SignalingClient
	registry *Registry
	now      func() time.Time
}

func NewManager(conn *net.UDPConn, gatherer CandidateGatherer, client SignalingClient, registry *Registry) *Manager {
	if registry == nil {
		registry = NewRegistry()
	}
	return &Manager{
		conn:     conn,
		gatherer: gatherer,
		client:   client,
		registry: registry,
		now:      time.Now,
	}
}

func (m *Manager) RefreshOnce(ctx context.Context) error {
	candidates, err := m.gatherer.Gather(ctx, m.conn)
	if err != nil {
		return err
	}
	probe := ClassifyNAT(candidates)
	if m.client != nil {
		_, err = m.client.ReportDirectCandidates(domain.ReportDirectCandidatesInput{
			UDPListenPort: localUDPPort(m.conn),
			NATType:       probe.NATType,
			Candidates:    candidates,
			ObservedAt:    m.now().UTC().Format(time.RFC3339),
		})
		if err != nil {
			return err
		}
		plan, err := m.client.FetchDirectLinkPlan()
		if err != nil {
			return err
		}
		m.applyPlan(plan)
	}
	return nil
}

func (m *Manager) Registry() *Registry {
	return m.registry
}

func (m *Manager) applyPlan(plan domain.DirectLinkPlan) {
	for _, link := range plan.Links {
		m.registry.Upsert(PeerState{
			LinkID:         link.LinkID,
			PeerNodeID:     link.PeerNodeID,
			Status:         domain.DirectStatusProbing,
			FallbackReason: "",
		})
	}
}
