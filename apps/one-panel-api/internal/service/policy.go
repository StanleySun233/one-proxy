package service

import (
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/policy"
)

func (c *ControlPlane) PolicyRevisions() []domain.PolicyRevision {
	return c.store.ListPolicyRevisions()
}

func (c *ControlPlane) PublishPolicy(accountID string) (domain.PolicyRevision, error) {
	if accountID == "" {
		return domain.PolicyRevision{}, unauthorized("invalid_access_token")
	}
	if _, err := policy.Compile(c.store.ListNodes(), c.store.ListNodeLinks(), c.store.ListChains(), c.store.ListRouteRules(), nil); err != nil {
		return domain.PolicyRevision{}, invalidInput("invalid_policy_graph")
	}
	return c.store.PublishPolicy(accountID)
}

func (c *ControlPlane) NodeAgentPolicy(nodeID string) (domain.NodeAgentPolicy, bool) {
	return c.store.GetNodeAgentPolicy(nodeID)
}
