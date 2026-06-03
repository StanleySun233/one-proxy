package service

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

type proxyStatusStore struct {
	mu       sync.RWMutex
	limit    int
	sessions []domain.ProxySessionMetric
	events   []domain.ProxyAuditEvent
}

func newProxyStatusStore(limit int) *proxyStatusStore {
	return &proxyStatusStore{limit: limit}
}

func (c *ControlPlane) IngestProxySessions(nodeID string, input domain.ProxySessionIngestInput) (domain.ProxySessionIngestResult, error) {
	if nodeID == "" {
		return domain.ProxySessionIngestResult{}, unauthorized("invalid_node_token")
	}
	if len(input.Sessions) == 0 {
		return domain.ProxySessionIngestResult{Status: "ok"}, nil
	}
	now := time.Now().UTC()
	items := make([]domain.ProxySessionMetric, 0, len(input.Sessions))
	events := make([]domain.ProxyAuditEvent, 0)
	for _, item := range input.Sessions {
		if item.ID == "" {
			return domain.ProxySessionIngestResult{}, invalidInput("missing_session_id")
		}
		if item.TenantID == "" {
			return domain.ProxySessionIngestResult{}, invalidInput("tenant_required")
		}
		if item.NodeID == "" {
			item.NodeID = nodeID
		}
		if item.NodeID != nodeID {
			return domain.ProxySessionIngestResult{}, newError(http.StatusForbidden, "node_mismatch")
		}
		if _, ok := c.store.NodeBindingPermission(nodeTenantContext(item.TenantID), nodeID); !ok {
			return domain.ProxySessionIngestResult{}, newError(http.StatusForbidden, "tenant_node_forbidden")
		}
		item.TargetHost = strings.ToLower(strings.TrimSpace(item.TargetHost))
		item.Status = normalizeProxyStatus(item.Status)
		item.ReceivedAt = now
		items = append(items, item)
		if item.Status != domain.ProxySessionStatusOK || item.ErrorCode != "" || item.ErrorMessage != "" {
			events = append(events, proxySessionEvent(item, now))
		}
	}
	c.proxyStatus.append(items, events)
	return domain.ProxySessionIngestResult{Status: "ok"}, nil
}

func (c *ControlPlane) ExtensionPageStatus(tenantCtx domain.TenantAuthContext, query domain.ProxyPageStatusQuery) domain.ProxyPageStatus {
	result := c.proxyStatus.pageStatus(tenantCtx.ActiveTenant.TenantID, query)
	result.PolicyRevision = c.latestPolicyRevision(tenantCtx)
	return result
}

func (c *ControlPlane) AuditProxySessions(tenantCtx domain.TenantAuthContext, query domain.ProxyAuditQuery) domain.ProxyAuditSessionsResult {
	return domain.ProxyAuditSessionsResult{Sessions: c.proxyStatus.querySessions(tenantCtx.ActiveTenant.TenantID, query)}
}

func (c *ControlPlane) AuditProxyEvents(tenantCtx domain.TenantAuthContext, query domain.ProxyAuditQuery) domain.ProxyAuditEventsResult {
	return domain.ProxyAuditEventsResult{Events: c.proxyStatus.queryEvents(tenantCtx.ActiveTenant.TenantID, query)}
}

func (c *ControlPlane) latestPolicyRevision(tenantCtx domain.TenantAuthContext) string {
	items := c.store.ListPolicyRevisionsForTenant(tenantCtx)
	if len(items) == 0 {
		return ""
	}
	return items[0].Version
}

func nodeTenantContext(tenantID string) domain.TenantAuthContext {
	return domain.TenantAuthContext{
		ActiveTenant: domain.TenantMembership{TenantID: tenantID},
	}
}

func normalizeProxyStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case domain.ProxySessionStatusOK:
		return domain.ProxySessionStatusOK
	case domain.ProxySessionStatusSlow:
		return domain.ProxySessionStatusSlow
	case domain.ProxySessionStatusError:
		return domain.ProxySessionStatusError
	default:
		return domain.ProxySessionStatusUnknown
	}
}

func proxySessionEvent(item domain.ProxySessionMetric, receivedAt time.Time) domain.ProxyAuditEvent {
	level := "error"
	if item.Status == domain.ProxySessionStatusSlow {
		level = "warn"
	}
	occurredAt := receivedAt
	if item.EndedAt != nil {
		occurredAt = *item.EndedAt
	} else if !item.StartedAt.IsZero() {
		occurredAt = item.StartedAt
	}
	return domain.ProxyAuditEvent{
		ID:         item.ID,
		TenantID:   item.TenantID,
		NodeID:     item.NodeID,
		ChainID:    item.ChainID,
		RouteID:    item.RouteID,
		TargetHost: item.TargetHost,
		Level:      level,
		Code:       item.ErrorCode,
		Message:    item.ErrorMessage,
		OccurredAt: occurredAt,
		SessionID:  item.ID,
	}
}

func (s *proxyStatusStore) append(sessions []domain.ProxySessionMetric, events []domain.ProxyAuditEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = append(s.sessions, sessions...)
	if len(s.sessions) > s.limit {
		s.sessions = s.sessions[len(s.sessions)-s.limit:]
	}
	s.events = append(s.events, events...)
	if len(s.events) > s.limit {
		s.events = s.events[len(s.events)-s.limit:]
	}
}

func (s *proxyStatusStore) pageStatus(tenantID string, query domain.ProxyPageStatusQuery) domain.ProxyPageStatus {
	query.Host = strings.ToLower(strings.TrimSpace(query.Host))
	result := domain.ProxyPageStatus{Status: domain.ProxySessionStatusUnknown}
	if tenantID == "" || query.Host == "" {
		return result
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var latencyTotal int64
	var last domain.ProxySessionMetric
	for _, item := range s.sessions {
		if !matchesProxySession(item, tenantID, query, domain.ProxyAuditQuery{}) {
			continue
		}
		result.Correlated = true
		result.RequestCount++
		result.UploadBytes += item.UploadBytes
		result.DownloadBytes += item.DownloadBytes
		latencyTotal += item.LatencyMs
		if item.Status != domain.ProxySessionStatusOK {
			result.FailureCount++
			result.LastErrorCode = item.ErrorCode
			result.LastErrorMessage = item.ErrorMessage
		}
		if sessionSeenAt(item).After(sessionSeenAt(last)) {
			last = item
		}
	}
	if !result.Correlated {
		return result
	}
	result.LastSeenAt = sessionSeenAt(last)
	if result.RequestCount > 0 {
		result.LatencyMs = latencyTotal / int64(result.RequestCount)
	}
	switch {
	case last.Status == domain.ProxySessionStatusError:
		result.Status = domain.ProxySessionStatusError
	case result.LatencyMs > 1000:
		result.Status = domain.ProxySessionStatusSlow
	default:
		result.Status = domain.ProxySessionStatusOK
	}
	return result
}

func (s *proxyStatusStore) querySessions(tenantID string, query domain.ProxyAuditQuery) []domain.ProxySessionMetric {
	if query.Limit <= 0 || query.Limit > 500 {
		query.Limit = 100
	}
	query.Host = strings.ToLower(strings.TrimSpace(query.Host))
	result := make([]domain.ProxySessionMetric, 0)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := len(s.sessions) - 1; i >= 0 && len(result) < query.Limit; i-- {
		item := s.sessions[i]
		if matchesProxySession(item, tenantID, domain.ProxyPageStatusQuery{}, query) {
			result = append(result, item)
		}
	}
	return result
}

func (s *proxyStatusStore) queryEvents(tenantID string, query domain.ProxyAuditQuery) []domain.ProxyAuditEvent {
	if query.Limit <= 0 || query.Limit > 500 {
		query.Limit = 100
	}
	query.Host = strings.ToLower(strings.TrimSpace(query.Host))
	result := make([]domain.ProxyAuditEvent, 0)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := len(s.events) - 1; i >= 0 && len(result) < query.Limit; i-- {
		item := s.events[i]
		if item.TenantID != tenantID {
			continue
		}
		if query.Host != "" && item.TargetHost != query.Host {
			continue
		}
		if query.ChainID != "" && item.ChainID != query.ChainID {
			continue
		}
		if query.RouteID != "" && item.RouteID != query.RouteID {
			continue
		}
		if query.NodeID != "" && item.NodeID != query.NodeID {
			continue
		}
		if query.Level != "" && item.Level != query.Level {
			continue
		}
		if !query.From.IsZero() && item.OccurredAt.Before(query.From) {
			continue
		}
		if !query.To.IsZero() && item.OccurredAt.After(query.To) {
			continue
		}
		result = append(result, item)
	}
	return result
}

func matchesProxySession(item domain.ProxySessionMetric, tenantID string, page domain.ProxyPageStatusQuery, audit domain.ProxyAuditQuery) bool {
	if item.TenantID != tenantID {
		return false
	}
	if page.Host != "" && item.TargetHost != page.Host {
		return false
	}
	if page.ChainID != "" && item.ChainID != page.ChainID {
		return false
	}
	if page.RouteID != "" && item.RouteID != page.RouteID {
		return false
	}
	if audit.Host != "" && item.TargetHost != audit.Host {
		return false
	}
	if audit.ChainID != "" && item.ChainID != audit.ChainID {
		return false
	}
	if audit.RouteID != "" && item.RouteID != audit.RouteID {
		return false
	}
	if audit.NodeID != "" && item.NodeID != audit.NodeID {
		return false
	}
	if audit.Status != "" && item.Status != audit.Status {
		return false
	}
	seenAt := sessionSeenAt(item)
	if !audit.From.IsZero() && seenAt.Before(audit.From) {
		return false
	}
	if !audit.To.IsZero() && seenAt.After(audit.To) {
		return false
	}
	return true
}

func sessionSeenAt(item domain.ProxySessionMetric) time.Time {
	if item.EndedAt != nil {
		return *item.EndedAt
	}
	if !item.StartedAt.IsZero() {
		return item.StartedAt
	}
	return item.ReceivedAt
}
