package service

import "github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"

func (c *ControlPlane) Overview(tenantCtx domain.TenantAuthContext) domain.Overview {
	if tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.TenantID == "" {
		return c.store.GetOverview()
	}
	nodes := c.store.ListNodesForTenant(tenantCtx)
	healthy := 0
	degraded := 0
	for _, node := range nodes {
		if node.Status == domain.NodeStatusHealthy {
			healthy++
		} else {
			degraded++
		}
	}
	renewSoon := 0
	for _, item := range c.NodeHealth(tenantCtx) {
		for _, state := range item.CertStatus {
			if state == domain.CertStatusRenewSoon || state == "rotate" {
				renewSoon++
				break
			}
		}
	}
	latest := domain.OverviewPolicies{}
	for _, item := range c.store.ListPolicyRevisionsForTenant(tenantCtx) {
		latest.ActiveRevision = item.Version
		latest.PublishedAt = item.CreatedAt
		break
	}
	return domain.Overview{
		Nodes:        domain.OverviewNodes{Healthy: healthy, Degraded: degraded},
		Policies:     latest,
		Certificates: domain.OverviewCertificates{RenewSoon: renewSoon},
	}
}
