package direct

import "github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"

type NATProbeResult struct {
	NATType       string
	ObservedPairs []string
}

func ClassifyNAT(candidates []domain.DirectCandidate) NATProbeResult {
	pairs := make([]string, 0)
	seen := make(map[string]bool)
	for _, candidate := range candidates {
		if candidate.Type != domain.CandidateTypeServerReflexive {
			continue
		}
		pair := candidateAddress(candidate)
		if seen[pair] {
			continue
		}
		seen[pair] = true
		pairs = append(pairs, pair)
	}
	if len(pairs) == 0 {
		return NATProbeResult{NATType: domain.NATTypeBlocked, ObservedPairs: pairs}
	}
	if len(pairs) == 1 {
		return NATProbeResult{NATType: domain.NATTypeEndpointIndependent, ObservedPairs: pairs}
	}
	return NATProbeResult{NATType: domain.NATTypeAddressDependent, ObservedPairs: pairs}
}
