package service

import (
	"crypto/rand"
	"encoding/base64"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

type clientDirectSessionRecord struct {
	SessionID    string
	PunchToken   string
	TargetNodeID string
	TargetHost   string
	TargetPort   int
	ExpiresAt    time.Time
}

func (c *ControlPlane) ClientDirectSession(tenantCtx domain.TenantAuthContext, input domain.ClientDirectSessionInput) (domain.ClientDirectSession, error) {
	if tenantCtx.ActiveTenant.TenantID == "" {
		return domain.ClientDirectSession{}, invalidInput("tenant_required")
	}
	if input.AccessPathID == "" {
		return domain.ClientDirectSession{}, invalidInput("invalid_direct_session_payload")
	}
	path, ok := c.accessPathForTenant(tenantCtx, input.AccessPathID)
	if !ok || !path.Enabled {
		return domain.ClientDirectSession{}, newError(http.StatusNotFound, "access_path_not_found")
	}
	if path.TargetNodeID == "" || path.EntryNodeID == "" || path.TargetHost == "" || path.TargetPort <= 0 {
		return domain.ClientDirectSession{}, invalidInput("invalid_direct_session_payload")
	}
	targetHost := path.TargetHost
	targetPort := path.TargetPort
	if input.TargetHost != "" {
		targetHost = input.TargetHost
	}
	if input.TargetPort > 0 {
		targetPort = input.TargetPort
	}
	nodes := c.store.ListNodesForTenant(tenantCtx)
	entryNode, ok := nodeByID(nodes, path.EntryNodeID)
	if !ok || entryNode.PublicHost == "" || entryNode.PublicPort <= 0 {
		return domain.ClientDirectSession{}, invalidInput("relay_entry_unavailable")
	}
	candidates := c.directCandidatesForNode(path.TargetNodeID)
	if len(candidates) == 0 {
		return domain.ClientDirectSession{}, newError(http.StatusConflict, "direct_candidates_unavailable")
	}
	sessionID, err := randomURLToken()
	if err != nil {
		return domain.ClientDirectSession{}, internalFailure("direct_session_issue_failed")
	}
	punchToken, err := randomURLToken()
	if err != nil {
		return domain.ClientDirectSession{}, internalFailure("direct_session_issue_failed")
	}
	expiresAt := time.Now().UTC().Add(2 * time.Minute)
	c.clientDirectMu.Lock()
	c.clientDirect[sessionID] = clientDirectSessionRecord{
		SessionID:    sessionID,
		PunchToken:   punchToken,
		TargetNodeID: path.TargetNodeID,
		TargetHost:   targetHost,
		TargetPort:   targetPort,
		ExpiresAt:    expiresAt,
	}
	c.clientDirectMu.Unlock()
	return domain.ClientDirectSession{
		SessionID:      sessionID,
		AccessPathID:   path.ID,
		TargetNodeID:   path.TargetNodeID,
		TargetHost:     targetHost,
		TargetPort:     targetPort,
		RelayEntryHost: entryNode.PublicHost,
		RelayEntryPort: entryNode.PublicPort,
		PunchToken:     punchToken,
		ExpiresAt:      expiresAt.Format(time.RFC3339),
		NodeCandidates: candidates,
	}, nil
}

func (c *ControlPlane) ValidateClientDirectSession(nodeID string, input domain.ClientDirectSessionValidateInput) (domain.ClientDirectSessionValidateResult, error) {
	if nodeID == "" || input.SessionID == "" || input.PunchToken == "" || input.TargetHost == "" || input.TargetPort <= 0 {
		return domain.ClientDirectSessionValidateResult{}, invalidInput("invalid_direct_session")
	}
	c.clientDirectMu.Lock()
	record, ok := c.clientDirect[input.SessionID]
	if ok && time.Now().UTC().After(record.ExpiresAt) {
		delete(c.clientDirect, input.SessionID)
		ok = false
	}
	c.clientDirectMu.Unlock()
	if !ok || record.PunchToken != input.PunchToken || record.TargetNodeID != nodeID || record.TargetHost != input.TargetHost || record.TargetPort != input.TargetPort {
		return domain.ClientDirectSessionValidateResult{Valid: false}, nil
	}
	return domain.ClientDirectSessionValidateResult{
		Valid:      true,
		TargetHost: record.TargetHost,
		TargetPort: record.TargetPort,
	}, nil
}

func (c *ControlPlane) accessPathForTenant(tenantCtx domain.TenantAuthContext, pathID string) (domain.NodeAccessPath, bool) {
	for _, item := range c.store.ListNodeAccessPathsForTenant(tenantCtx) {
		if item.ID == pathID {
			return item, true
		}
	}
	return domain.NodeAccessPath{}, false
}

func (c *ControlPlane) directCandidatesForNode(nodeID string) []domain.DirectCandidate {
	items := make([]domain.DirectCandidate, 0)
	for _, transport := range c.store.ListNodeTransports() {
		if transport.NodeID != nodeID || transport.TransportType != "direct_udp_candidate" {
			continue
		}
		if transport.Status != domain.TransportStatusAvailable && transport.Status != domain.TransportStatusConnected {
			continue
		}
		host, portText, err := net.SplitHostPort(transport.Address)
		if err != nil {
			continue
		}
		port, err := strconv.Atoi(portText)
		if err != nil || port <= 0 {
			continue
		}
		priority, _ := strconv.Atoi(transport.Details["priority"])
		items = append(items, domain.DirectCandidate{
			Type:       transport.Details["candidateType"],
			Address:    host,
			Port:       port,
			Protocol:   transport.Details["protocol"],
			StunServer: transport.Details["stunServer"],
			Priority:   priority,
		})
	}
	return items
}

func randomURLToken() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
