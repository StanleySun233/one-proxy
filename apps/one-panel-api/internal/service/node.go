package service

import "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"

func (c *ControlPlane) Nodes() []domain.Node {
	return c.store.ListNodes()
}

func (c *ControlPlane) UpdateNode(nodeID string, input domain.UpdateNodeInput) (domain.Node, error) {
	if nodeID == "" {
		return domain.Node{}, invalidInput("missing_node_id")
	}
	if err := validateNodeInput(input.Name, input.Mode, input.ScopeKey); err != nil {
		return domain.Node{}, err
	}
	if !c.scopeExists(input.ScopeKey) {
		return domain.Node{}, invalidInput("scope_not_found")
	}
	return c.store.UpdateNode(nodeID, input)
}

func (c *ControlPlane) DeleteNode(nodeID string) error {
	return c.store.DeleteNode(nodeID)
}

func validateNodeInput(name string, mode string, scopeKey string) error {
	if name == "" || mode == "" || scopeKey == "" {
		return invalidInput("invalid_node_payload")
	}
	return nil
}
