package linkservice

import "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"

func (s *Service) NodeLinks() []domain.NodeLink {
	return s.store.ListNodeLinks()
}

func (s *Service) CreateNodeLink(input domain.CreateNodeLinkInput) (domain.NodeLink, error) {
	if input.SourceNodeID == "" || input.TargetNodeID == "" || input.LinkType == "" || input.TrustState == "" {
		return domain.NodeLink{}, invalidInput("invalid_node_link_payload")
	}
	return s.store.CreateNodeLink(input)
}

func (s *Service) UpdateNodeLink(linkID string, input domain.UpdateNodeLinkInput) (domain.NodeLink, error) {
	if linkID == "" {
		return domain.NodeLink{}, invalidInput("missing_node_link_id")
	}
	if input.SourceNodeID == "" || input.TargetNodeID == "" || input.LinkType == "" || input.TrustState == "" {
		return domain.NodeLink{}, invalidInput("invalid_node_link_payload")
	}
	return s.store.UpdateNodeLink(linkID, input)
}

func (s *Service) DeleteNodeLink(linkID string) error {
	if linkID == "" {
		return invalidInput("missing_node_link_id")
	}
	return s.store.DeleteNodeLink(linkID)
}
