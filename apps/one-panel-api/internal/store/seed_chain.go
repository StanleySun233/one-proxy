package store

import "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"

func (s *SeedStore) ListChains() []domain.Chain {
	return []domain.Chain{}
}

func (s *SeedStore) GetChainProbeResult(chainID string) (domain.ChainProbeResult, bool) {
	_ = chainID
	return domain.ChainProbeResult{}, false
}

func (s *SeedStore) SaveChainProbeResult(input domain.SaveChainProbeResultInput) (domain.ChainProbeResult, error) {
	return domain.ChainProbeResult{
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

func (s *SeedStore) CreateChain(input domain.CreateChainInput) (domain.Chain, error) {
	return domain.Chain{
		ID:               s.nextID("chain"),
		Name:             input.Name,
		DestinationScope: input.DestinationScope,
		Enabled:          true,
		Hops:             input.Hops,
	}, nil
}

func (s *SeedStore) UpdateChain(chainID string, input domain.UpdateChainInput) (domain.Chain, error) {
	return domain.Chain{
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
