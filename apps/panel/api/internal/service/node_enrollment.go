package service

import (
	"strings"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func (c *ControlPlane) PendingNodeEnrollments(tenantCtx domain.TenantAuthContext) ([]domain.Node, error) {
	if !tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.Role != domain.TenantRoleAdmin {
		return nil, newError(403, "tenant_role_forbidden")
	}
	allowed := c.tenantNodeIDs(tenantCtx)
	items := make([]domain.Node, 0)
	for _, node := range c.store.ListPendingNodes() {
		if allowed[node.ID] {
			items = append(items, node)
		}
	}
	return items, nil
}

func (c *ControlPlane) RejectNodeEnrollment(tenantCtx domain.TenantAuthContext, nodeID string, accountID string, reason string) error {
	if nodeID == "" {
		return invalidInput("missing_node_id")
	}
	if accountID == "" {
		return unauthorized("invalid_access_token")
	}
	if err := c.requireTenantResourceManage(tenantCtx, func() (domain.BindingPermission, bool) {
		return c.store.NodeBindingPermission(tenantCtx, nodeID)
	}); err != nil {
		return err
	}
	if err := c.store.RejectNodeEnrollment(nodeID, accountID, reason); err != nil {
		if strings.Contains(err.Error(), "node_not_pending") {
			return invalidInput("node_not_pending")
		}
		if strings.Contains(err.Error(), "node_not_enrolled") {
			return invalidInput("node_not_enrolled")
		}
		return err
	}
	return nil
}

func (c *ControlPlane) EnrollNode(input domain.EnrollNodeInput) (domain.EnrollNodeResult, error) {
	if input.Token == "" {
		return domain.EnrollNodeResult{}, invalidInput("missing_bootstrap_token")
	}
	return c.store.EnrollNode(input)
}

func (c *ControlPlane) ApproveNodeEnrollment(tenantCtx domain.TenantAuthContext, nodeID string, reviewedBy string) (domain.ApproveNodeEnrollmentResult, error) {
	if nodeID == "" {
		return domain.ApproveNodeEnrollmentResult{}, invalidInput("missing_node_id")
	}
	if err := c.requireTenantResourceManage(tenantCtx, func() (domain.BindingPermission, bool) {
		return c.store.NodeBindingPermission(tenantCtx, nodeID)
	}); err != nil {
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	item, err := c.store.ApproveNodeEnrollment(nodeID, reviewedBy)
	if err != nil {
		if strings.Contains(err.Error(), "node_not_pending") {
			return domain.ApproveNodeEnrollmentResult{}, invalidInput("node_not_pending")
		}
		if strings.Contains(err.Error(), "node_not_enrolled") {
			return domain.ApproveNodeEnrollmentResult{}, invalidInput("node_not_enrolled")
		}
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	return item, nil
}

func (c *ControlPlane) ExchangeNodeEnrollment(input domain.ExchangeNodeEnrollmentInput) (domain.ApproveNodeEnrollmentResult, error) {
	if input.NodeID == "" || input.EnrollmentSecret == "" {
		return domain.ApproveNodeEnrollmentResult{}, invalidInput("invalid_enrollment_exchange_payload")
	}
	item, err := c.store.ExchangeNodeEnrollment(input)
	if err != nil {
		if strings.Contains(err.Error(), "node_enrollment_pending") {
			return domain.ApproveNodeEnrollmentResult{}, invalidInput("node_enrollment_pending")
		}
		if strings.Contains(err.Error(), "invalid_enrollment_secret") {
			return domain.ApproveNodeEnrollmentResult{}, invalidInput("invalid_enrollment_secret")
		}
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	return item, nil
}

func (c *ControlPlane) AuthenticateNodeToken(accessToken string) (string, bool) {
	return c.store.AuthenticateNodeToken(accessToken)
}
