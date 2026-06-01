package store

import "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"

func (s *SeedStore) ListNodeLinks() []domain.NodeLink {
	return []domain.NodeLink{}
}

func (s *SeedStore) CreateNodeLink(input domain.CreateNodeLinkInput) (domain.NodeLink, error) {
	return domain.NodeLink{
		ID:           s.nextID("node_link"),
		SourceNodeID: input.SourceNodeID,
		TargetNodeID: input.TargetNodeID,
		LinkType:     input.LinkType,
		TrustState:   input.TrustState,
	}, nil
}

func (s *SeedStore) UpdateNodeLink(linkID string, input domain.UpdateNodeLinkInput) (domain.NodeLink, error) {
	return domain.NodeLink{
		ID:           linkID,
		SourceNodeID: input.SourceNodeID,
		TargetNodeID: input.TargetNodeID,
		LinkType:     input.LinkType,
		TrustState:   input.TrustState,
	}, nil
}

func (s *SeedStore) DeleteNodeLink(linkID string) error {
	_ = linkID
	return nil
}
