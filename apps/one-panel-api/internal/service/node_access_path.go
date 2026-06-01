package service

import "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"

func (c *ControlPlane) NodeAccessPaths() []domain.NodeAccessPath {
	return c.store.ListNodeAccessPaths()
}

func (c *ControlPlane) CreateNodeAccessPath(input domain.CreateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	if err := c.validateNodeAccessPath(input.Name, input.Mode, input.TargetHost, input.TargetPort); err != nil {
		return domain.NodeAccessPath{}, err
	}
	return c.store.CreateNodeAccessPath(input)
}

func (c *ControlPlane) UpdateNodeAccessPath(pathID string, input domain.UpdateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	if pathID == "" {
		return domain.NodeAccessPath{}, invalidInput("missing_path_id")
	}
	if err := c.validateNodeAccessPath(input.Name, input.Mode, input.TargetHost, input.TargetPort); err != nil {
		return domain.NodeAccessPath{}, err
	}
	return c.store.UpdateNodeAccessPath(pathID, input)
}

func (c *ControlPlane) DeleteNodeAccessPath(pathID string) error {
	if pathID == "" {
		return invalidInput("missing_path_id")
	}
	return c.store.DeleteNodeAccessPath(pathID)
}

func (c *ControlPlane) validateNodeAccessPath(name string, mode string, targetHost string, targetPort int) error {
	if name == "" || mode == "" {
		return invalidInput("invalid_node_access_path_payload")
	}
	if !c.isValidEnum("path_mode", mode) {
		return invalidInput("invalid_node_access_path_payload")
	}
	switch mode {
	case domain.PathModeDirect, domain.PathModeRelayChain:
		if targetHost == "" || targetPort <= 0 {
			return invalidInput("invalid_node_access_path_payload")
		}
	}
	return nil
}
