package service

import "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"

func (c *ControlPlane) NodeLinks() []domain.NodeLink {
	return c.store.ListNodeLinks()
}

func (c *ControlPlane) CreateNodeLink(input domain.CreateNodeLinkInput) (domain.NodeLink, error) {
	if input.SourceNodeID == "" || input.TargetNodeID == "" || input.LinkType == "" || input.TrustState == "" {
		return domain.NodeLink{}, invalidInput("invalid_node_link_payload")
	}
	return c.store.CreateNodeLink(input)
}

func (c *ControlPlane) UpdateNodeLink(linkID string, input domain.UpdateNodeLinkInput) (domain.NodeLink, error) {
	if linkID == "" {
		return domain.NodeLink{}, invalidInput("missing_node_link_id")
	}
	if input.SourceNodeID == "" || input.TargetNodeID == "" || input.LinkType == "" || input.TrustState == "" {
		return domain.NodeLink{}, invalidInput("invalid_node_link_payload")
	}
	return c.store.UpdateNodeLink(linkID, input)
}

func (c *ControlPlane) DeleteNodeLink(linkID string) error {
	if linkID == "" {
		return invalidInput("missing_node_link_id")
	}
	return c.store.DeleteNodeLink(linkID)
}
