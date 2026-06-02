package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (s *MySQLStore) UpsertDirectCandidates(nodeID string, input domain.DirectCandidatesInput) (domain.DirectCandidatesResult, error) {
	now := nowRFC3339()
	if _, err := s.db.Exec(`DELETE FROM node_transports WHERE node_id = ? AND transport_type = ?`, nodeID, "direct_udp_candidate"); err != nil {
		return domain.DirectCandidatesResult{}, err
	}
	for _, candidate := range input.Candidates {
		details := map[string]string{
			"candidateType": candidate.Type,
			"protocol":      candidate.Protocol,
			"natType":       input.NATType,
			"observedAt":    input.ObservedAt,
			"udpListenPort": strconv.Itoa(input.UDPListenPort),
			"priority":      strconv.Itoa(candidate.Priority),
		}
		if candidate.StunServer != "" {
			details["stunServer"] = candidate.StunServer
		}
		if _, err := s.UpsertNodeTransport(domain.UpsertNodeTransportInput{
			NodeID:          nodeID,
			TransportType:   "direct_udp_candidate",
			Direction:       "peer",
			Address:         fmt.Sprintf("%s:%d", candidate.Address, candidate.Port),
			Status:          domain.TransportStatusAvailable,
			LastHeartbeatAt: now,
			LatencyMs:       0,
			Details:         details,
		}); err != nil {
			return domain.DirectCandidatesResult{}, err
		}
	}
	return domain.DirectCandidatesResult{
		NodeID:         nodeID,
		CandidateCount: len(input.Candidates),
		UpdatedAt:      now,
	}, nil
}

func (s *MySQLStore) DirectLinkPlans(nodeID string, ttl time.Duration) (domain.DirectLinkPlanResult, error) {
	rows, err := s.db.Query(
		`SELECT id, source_node_id, target_node_id
		 FROM node_links
		 WHERE trust_state IN (?, ?) AND (source_node_id = ? OR target_node_id = ?)
		 ORDER BY id`,
		domain.TrustStateTrusted, domain.TrustStateActive, nodeID, nodeID,
	)
	if err != nil {
		return domain.DirectLinkPlanResult{}, err
	}
	defer rows.Close()

	expiresAt := time.Now().UTC().Add(ttl).Format(time.RFC3339)
	result := domain.DirectLinkPlanResult{NodeID: nodeID, Links: []domain.DirectLinkPlan{}}
	for rows.Next() {
		var linkID, sourceNodeID, targetNodeID string
		if err := rows.Scan(&linkID, &sourceNodeID, &targetNodeID); err != nil {
			continue
		}
		peerNodeID := sourceNodeID
		role := "dialer"
		if sourceNodeID == nodeID {
			peerNodeID = targetNodeID
			role = "listener"
		}
		token, err := punchToken()
		if err != nil {
			return domain.DirectLinkPlanResult{}, err
		}
		result.Links = append(result.Links, domain.DirectLinkPlan{
			LinkID:             linkID,
			PeerNodeID:         peerNodeID,
			Role:               role,
			PreferredTransport: "direct_quic",
			FallbackTransport:  "relay_ws_parent",
			PunchToken:         token,
			ExpiresAt:          expiresAt,
			PeerCandidates:     s.directCandidates(peerNodeID),
		})
	}
	return result, nil
}

func (s *MySQLStore) UpsertDirectStatus(nodeID string, input domain.DirectStatusInput) (domain.DirectStatusResult, error) {
	ok, err := s.authorizedDirectLink(nodeID, input.LinkID, input.PeerNodeID)
	if err != nil {
		return domain.DirectStatusResult{}, err
	}
	if !ok {
		return domain.DirectStatusResult{}, sql.ErrNoRows
	}
	now := nowRFC3339()
	address := fmt.Sprintf("%s:%d", input.SelectedCandidate.Address, input.SelectedCandidate.Port)
	connectedAt := ""
	if input.Status == domain.TransportStatusConnected {
		connectedAt = input.LastProbeAt
	}
	details := map[string]string{
		"peerNodeId":        input.PeerNodeID,
		"linkId":            input.LinkID,
		"fallbackTransport": "relay_ws_parent",
		"fallbackReason":    input.FallbackReason,
		"candidateType":     input.SelectedCandidate.Type,
		"protocol":          input.SelectedCandidate.Protocol,
	}
	if _, err := s.UpsertNodeTransport(domain.UpsertNodeTransportInput{
		NodeID:          nodeID,
		TransportType:   input.TransportType,
		Direction:       "peer",
		Address:         address,
		Status:          input.Status,
		ConnectedAt:     connectedAt,
		LastHeartbeatAt: input.LastProbeAt,
		LatencyMs:       input.RTTMs,
		Details:         details,
	}); err != nil {
		return domain.DirectStatusResult{}, err
	}
	return domain.DirectStatusResult{
		LinkID:     input.LinkID,
		PeerNodeID: input.PeerNodeID,
		Status:     input.Status,
		UpdatedAt:  now,
	}, nil
}

func (s *MySQLStore) directCandidates(nodeID string) []domain.DirectCandidate {
	rows, err := s.db.Query(
		`SELECT address, details_json
		 FROM node_transports
		 WHERE node_id = ? AND transport_type = ? AND status IN (?, ?)
		 ORDER BY latency_ms ASC, address`,
		nodeID, "direct_udp_candidate", domain.TransportStatusAvailable, domain.TransportStatusConnected,
	)
	if err != nil {
		return []domain.DirectCandidate{}
	}
	defer rows.Close()

	items := make([]domain.DirectCandidate, 0)
	for rows.Next() {
		var address, detailsJSON string
		if err := rows.Scan(&address, &detailsJSON); err != nil {
			continue
		}
		host, port := splitHostPort(address)
		details := decodeJSONMap(detailsJSON)
		priority, _ := strconv.Atoi(details["priority"])
		items = append(items, domain.DirectCandidate{
			Type:       details["candidateType"],
			Address:    host,
			Port:       port,
			Protocol:   details["protocol"],
			StunServer: details["stunServer"],
			Priority:   priority,
		})
	}
	return items
}

func (s *MySQLStore) authorizedDirectLink(nodeID, linkID, peerNodeID string) (bool, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*)
		 FROM node_links
		 WHERE id = ? AND trust_state IN (?, ?) AND (
		   (source_node_id = ? AND target_node_id = ?) OR
		   (source_node_id = ? AND target_node_id = ?)
		 )`,
		linkID, domain.TrustStateTrusted, domain.TrustStateActive, nodeID, peerNodeID, peerNodeID, nodeID,
	).Scan(&count)
	return count > 0, err
}

func splitHostPort(address string) (string, int) {
	for i := len(address) - 1; i >= 0; i-- {
		if address[i] != ':' {
			continue
		}
		port, _ := strconv.Atoi(address[i+1:])
		return address[:i], port
	}
	return address, 0
}

func punchToken() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
