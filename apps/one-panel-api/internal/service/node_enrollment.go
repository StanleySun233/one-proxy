package service

import (
	"strings"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (c *ControlPlane) PendingNodeEnrollments() []domain.Node {
	return c.store.ListPendingNodes()
}

func (c *ControlPlane) RejectNodeEnrollment(nodeID string, accountID string, reason string) error {
	if nodeID == "" {
		return invalidInput("missing_node_id")
	}
	if accountID == "" {
		return unauthorized("invalid_access_token")
	}
	return c.store.RejectNodeEnrollment(nodeID, accountID, reason)
}

func (c *ControlPlane) EnrollNode(input domain.EnrollNodeInput) (domain.EnrollNodeResult, error) {
	if input.Token == "" {
		return domain.EnrollNodeResult{}, invalidInput("missing_bootstrap_token")
	}
	return c.store.EnrollNode(input)
}

func (c *ControlPlane) ApproveNodeEnrollment(nodeID string, reviewedBy string) (domain.ApproveNodeEnrollmentResult, error) {
	if nodeID == "" {
		return domain.ApproveNodeEnrollmentResult{}, invalidInput("missing_node_id")
	}
	item, err := c.store.ApproveNodeEnrollment(nodeID, reviewedBy)
	if err != nil {
		if strings.Contains(err.Error(), "node_not_pending") {
			return domain.ApproveNodeEnrollmentResult{}, invalidInput("node_not_pending")
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
