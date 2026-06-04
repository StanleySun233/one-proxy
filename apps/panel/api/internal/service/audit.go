package service

import (
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func (c *ControlPlane) RecordBusinessAuditEvent(input domain.CreateBusinessAuditEventInput) (domain.BusinessAuditEvent, error) {
	if input.Action == "" {
		return domain.BusinessAuditEvent{}, invalidInput("action_required")
	}
	if input.ResourceType == "" {
		return domain.BusinessAuditEvent{}, invalidInput("resource_type_required")
	}
	if input.Outcome == "" {
		input.Outcome = domain.AuditOutcomeSuccess
	}
	return c.store.CreateBusinessAuditEvent(input)
}

func (c *ControlPlane) BusinessAuditEvents(tenantCtx domain.TenantAuthContext, query domain.BusinessAuditQuery) (domain.BusinessAuditEventsResult, error) {
	if err := requireAuditAdmin(tenantCtx); err != nil {
		return domain.BusinessAuditEventsResult{}, err
	}
	query = scopeBusinessAuditQuery(tenantCtx, query)
	return c.store.ListBusinessAuditEvents(query)
}

func (c *ControlPlane) RecordNetworkAuditSession(input domain.CreateNetworkAuditSessionInput) (domain.NetworkAuditSession, error) {
	if input.TenantID == "" {
		return domain.NetworkAuditSession{}, invalidInput("tenant_required")
	}
	if input.EntryNodeID == "" {
		return domain.NetworkAuditSession{}, invalidInput("entry_node_required")
	}
	if input.Decision == "" {
		input.Decision = domain.NetworkDecisionAllow
	}
	if input.Decision != domain.NetworkDecisionAllow && input.Decision != domain.NetworkDecisionDeny {
		return domain.NetworkAuditSession{}, invalidInput("decision_invalid")
	}
	return c.store.CreateNetworkAuditSession(input)
}

func (c *ControlPlane) NetworkAuditSessions(tenantCtx domain.TenantAuthContext, query domain.NetworkAuditQuery) (domain.NetworkAuditSessionsResult, error) {
	if err := requireAuditAdmin(tenantCtx); err != nil {
		return domain.NetworkAuditSessionsResult{}, err
	}
	query = scopeNetworkAuditQuery(tenantCtx, query)
	return c.store.ListNetworkAuditSessions(query)
}

func (c *ControlPlane) AuditDashboard(tenantCtx domain.TenantAuthContext, query domain.AuditDashboardQuery) (domain.AuditDashboard, error) {
	if err := requireAuditAdmin(tenantCtx); err != nil {
		return domain.AuditDashboard{}, err
	}
	query = scopeAuditDashboardQuery(tenantCtx, query)
	return c.store.GetAuditDashboard(query)
}

func scopeBusinessAuditQuery(tenantCtx domain.TenantAuthContext, query domain.BusinessAuditQuery) domain.BusinessAuditQuery {
	if tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.TenantID == "" {
		return query
	}
	query.TenantID = tenantCtx.ActiveTenant.TenantID
	return query
}

func scopeNetworkAuditQuery(tenantCtx domain.TenantAuthContext, query domain.NetworkAuditQuery) domain.NetworkAuditQuery {
	if tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.TenantID == "" {
		return query
	}
	query.TenantID = tenantCtx.ActiveTenant.TenantID
	return query
}

func scopeAuditDashboardQuery(tenantCtx domain.TenantAuthContext, query domain.AuditDashboardQuery) domain.AuditDashboardQuery {
	if tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.TenantID == "" {
		return query
	}
	query.TenantID = tenantCtx.ActiveTenant.TenantID
	return query
}

func requireAuditAdmin(tenantCtx domain.TenantAuthContext) error {
	if tenantCtx.SuperAdmin || tenantCtx.ActiveTenant.Role == domain.TenantRoleAdmin {
		return nil
	}
	return newError(http.StatusForbidden, "audit_forbidden")
}
