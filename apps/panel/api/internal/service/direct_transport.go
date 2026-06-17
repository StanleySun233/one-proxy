package service

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

type directTransportStore interface {
	UpsertDirectCandidates(string, domain.DirectCandidatesInput) (domain.DirectCandidatesResult, error)
	DirectLinkPlans(string, time.Duration) (domain.DirectLinkPlanResult, error)
	UpsertDirectStatus(string, domain.DirectStatusInput) (domain.DirectStatusResult, error)
}

func (c *ControlPlane) UpsertDirectCandidates(nodeID string, input domain.DirectCandidatesInput) (domain.DirectCandidatesResult, error) {
	if nodeID == "" || !validDirectCandidates(nodeID, input) {
		return domain.DirectCandidatesResult{}, invalidInput("invalid_candidate_payload")
	}
	store, ok := c.store.(directTransportStore)
	if !ok {
		return domain.DirectCandidatesResult{}, internalFailure("direct_transport_store_unavailable")
	}
	return store.UpsertDirectCandidates(nodeID, input)
}

func (c *ControlPlane) DirectLinkPlan(nodeID string) (domain.DirectLinkPlanResult, error) {
	if nodeID == "" {
		return domain.DirectLinkPlanResult{}, unauthorized("invalid_node_token")
	}
	store, ok := c.store.(directTransportStore)
	if !ok {
		return domain.DirectLinkPlanResult{}, internalFailure("direct_transport_store_unavailable")
	}
	return store.DirectLinkPlans(nodeID, 5*time.Minute)
}

func (c *ControlPlane) UpsertDirectStatus(nodeID string, input domain.DirectStatusInput) (domain.DirectStatusResult, error) {
	if nodeID == "" {
		return domain.DirectStatusResult{}, unauthorized("invalid_node_token")
	}
	if !validDirectStatus(input) {
		return domain.DirectStatusResult{}, invalidInput("invalid_direct_status")
	}
	store, ok := c.store.(directTransportStore)
	if !ok {
		return domain.DirectStatusResult{}, internalFailure("direct_transport_store_unavailable")
	}
	result, err := store.UpsertDirectStatus(nodeID, input)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return domain.DirectStatusResult{}, err
		}
		return domain.DirectStatusResult{}, newError(http.StatusForbidden, "unauthorized_peer_link")
	}
	return result, nil
}

func validDirectCandidates(nodeID string, input domain.DirectCandidatesInput) bool {
	if input.UDPListenPort <= 0 || input.UDPListenPort > 65535 || input.ObservedAt == "" || len(input.Candidates) == 0 {
		return false
	}
	if !validDirectIdentity(nodeID, input.DirectIdentity) {
		return false
	}
	if _, err := time.Parse(time.RFC3339, input.ObservedAt); err != nil {
		return false
	}
	for _, candidate := range input.Candidates {
		if !validCandidate(candidate, true) {
			return false
		}
	}
	return true
}

func validDirectIdentity(nodeID string, identity domain.DirectNodeIdentity) bool {
	return identity.NodeID == nodeID &&
		identity.ServerName != "" &&
		identity.CertificateFingerprintSHA256 != "" &&
		identity.TrustMaterial != ""
}

func validDirectStatus(input domain.DirectStatusInput) bool {
	if input.LinkID == "" || input.PeerNodeID == "" || input.TransportType != "direct_quic" || input.Status == "" || input.LastProbeAt == "" || input.RTTMs < 0 {
		return false
	}
	if _, err := time.Parse(time.RFC3339, input.LastProbeAt); err != nil {
		return false
	}
	return validCandidate(input.SelectedCandidate, false)
}

func validCandidate(candidate domain.DirectCandidate, requirePriority bool) bool {
	if candidate.Address == "" || candidate.Port <= 0 || candidate.Port > 65535 || candidate.Protocol != "udp" {
		return false
	}
	if candidate.Type != "host" && candidate.Type != "srflx" {
		return false
	}
	return !requirePriority || candidate.Priority > 0
}
