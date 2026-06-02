package store

import "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/features/link/domain"

func (s *SeedStore) ListChains() []link.Chain {
	return []link.Chain{}
}

func (s *SeedStore) GetChainProbeResult(chainID string) (link.ChainProbeResult, bool) {
	_ = chainID
	return link.ChainProbeResult{}, false
}

func (s *SeedStore) SaveChainProbeResult(input link.SaveChainProbeResultInput) (link.ChainProbeResult, error) {
	return link.ChainProbeResult{
		ChainID:        input.ChainID,
		Status:         input.Status,
		Message:        input.Message,
		ResolvedHops:   input.ResolvedHops,
		BlockingNodeID: input.BlockingNodeID,
		BlockingReason: input.BlockingReason,
		TargetHost:     input.TargetHost,
		TargetPort:     input.TargetPort,
		ProbedAt:       input.ProbedAt,
	}, nil
}

func (s *SeedStore) CreateChain(input link.CreateChainInput) (link.Chain, error) {
	return link.Chain{
		ID:               s.nextID("chain"),
		Name:             input.Name,
		DestinationScope: input.DestinationScope,
		Enabled:          true,
		Hops:             input.Hops,
	}, nil
}

func (s *SeedStore) UpdateChain(chainID string, input link.UpdateChainInput) (link.Chain, error) {
	return link.Chain{
		ID:               chainID,
		Name:             input.Name,
		DestinationScope: input.DestinationScope,
		Enabled:          input.Enabled,
		Hops:             input.Hops,
	}, nil
}

func (s *SeedStore) DeleteChain(chainID string) error {
	_ = chainID
	return nil
}
