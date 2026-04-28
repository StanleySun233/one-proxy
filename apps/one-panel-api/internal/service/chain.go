package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/controlrelay"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (c *ControlPlane) Chains() []domain.Chain {
	return c.store.ListChains()
}

func (c *ControlPlane) ChainsWithDetails() []domain.ChainWithDetails {
	chains := c.store.ListChains()
	nodes := c.store.ListNodes()
	result := make([]domain.ChainWithDetails, 0, len(chains))

	for _, chain := range chains {
		hopDetails := make([]domain.ChainHopDetail, 0, len(chain.Hops))
		for _, hopID := range chain.Hops {
			node, ok := nodeByID(nodes, hopID)
			if ok {
				hopDetails = append(hopDetails, domain.ChainHopDetail{
					NodeID:   node.ID,
					NodeName: node.Name,
					Mode:     node.Mode,
				})
			}
		}

		result = append(result, domain.ChainWithDetails{
			ID:               chain.ID,
			Name:             chain.Name,
			DestinationScope: chain.DestinationScope,
			Enabled:          chain.Enabled,
			Hops:             chain.Hops,
			HopDetails:       hopDetails,
		})
	}

	return result
}

func (c *ControlPlane) GetChain(chainID string) (domain.ChainWithDetails, error) {
	if chainID == "" {
		return domain.ChainWithDetails{}, invalidInput("missing_chain_id")
	}

	chains := c.store.ListChains()
	chain, ok := chainByID(chains, chainID)
	if !ok {
		return domain.ChainWithDetails{}, invalidInput("chain_not_found")
	}

	nodes := c.store.ListNodes()
	hopDetails := make([]domain.ChainHopDetail, 0, len(chain.Hops))
	for _, hopID := range chain.Hops {
		node, ok := nodeByID(nodes, hopID)
		if ok {
			hopDetails = append(hopDetails, domain.ChainHopDetail{
				NodeID:   node.ID,
				NodeName: node.Name,
				Mode:     node.Mode,
			})
		}
	}

	return domain.ChainWithDetails{
		ID:               chain.ID,
		Name:             chain.Name,
		DestinationScope: chain.DestinationScope,
		Enabled:          chain.Enabled,
		Hops:             chain.Hops,
		HopDetails:       hopDetails,
	}, nil
}

func (c *ControlPlane) LatestChainProbe(chainID string) (domain.ChainProbeResult, bool) {
	if chainID == "" {
		return domain.ChainProbeResult{}, false
	}
	return c.store.GetChainProbeResult(chainID)
}

func (c *ControlPlane) ProbeChain(chainID string) (domain.ChainProbeResult, error) {
	if chainID == "" {
		return domain.ChainProbeResult{}, invalidInput("missing_chain_id")
	}
	chain, ok := chainByID(c.store.ListChains(), chainID)
	if !ok {
		return domain.ChainProbeResult{}, invalidInput("invalid_chain_id")
	}
	nodes := c.store.ListNodes()
	transports := c.store.ListNodeTransports()
	result := domain.ChainProbeResult{
		ChainID:      chainID,
		Status:       domain.ProbeResultStatusConnected,
		Message:      "chain_transport_ready",
		ResolvedHops: make([]domain.ChainProbeHop, 0, len(chain.Hops)),
		ProbedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	prevHopID := ""
	for _, hopID := range chain.Hops {
		node, ok := nodeByID(nodes, hopID)
		if !ok || !node.Enabled {
			result.Status = domain.ProbeResultStatusFailed
			result.Message = "chain_blocked"
			result.BlockingNodeID = hopID
			result.BlockingReason = "unknown_or_disabled_node"
			return c.store.SaveChainProbeResult(toChainProbeInput(result))
		}
		transport, ok := resolveProbeTransport(node, prevHopID, transports)
		if !ok {
			result.Status = domain.ProbeResultStatusFailed
			result.Message = "chain_blocked"
			result.BlockingNodeID = node.ID
			if prevHopID == "" {
				result.BlockingReason = "missing_entry_transport"
			} else {
				result.BlockingReason = "missing_parent_transport"
			}
			return c.store.SaveChainProbeResult(toChainProbeInput(result))
		}
		result.ResolvedHops = append(result.ResolvedHops, domain.ChainProbeHop{
			NodeID:        node.ID,
			NodeName:      node.Name,
			TransportType: transport.TransportType,
			Address:       transport.Address,
			Status:        transport.Status,
		})
		prevHopID = node.ID
	}
	if len(result.ResolvedHops) > 0 && (result.ResolvedHops[0].TransportType == domain.TransportTypePublicHTTP || result.ResolvedHops[0].TransportType == domain.TransportTypePublicHTTPS) {
		probeResult, err := controlrelay.Execute(result.ResolvedHops[0].Address, controlrelay.ProbeRequest{
			RemainingHopNodeIDs: chain.Hops[1:],
		})
		if err != nil {
			result.Status = domain.ProbeResultStatusFailed
			result.Message = "chain_probe_failed"
			result.BlockingNodeID = chain.Hops[0]
			result.BlockingReason = "probe_dispatch_failed"
			return c.store.SaveChainProbeResult(toChainProbeInput(result))
		}
		result.Status = probeResult.Status
		result.Message = probeResult.Message
		if probeResult.Status != domain.ProbeResultStatusConnected && result.BlockingReason == "" && len(chain.Hops) > 0 {
			result.BlockingNodeID = chain.Hops[len(chain.Hops)-1]
			result.BlockingReason = probeResult.Message
		}
	}
	return c.store.SaveChainProbeResult(toChainProbeInput(result))
}

func (c *ControlPlane) CreateChain(input domain.CreateChainInput) (domain.Chain, error) {
	if input.Name == "" || input.DestinationScope == "" || len(input.Hops) == 0 {
		return domain.Chain{}, invalidInput("invalid_chain_payload")
	}
	return c.store.CreateChain(input)
}

func (c *ControlPlane) UpdateChain(chainID string, input domain.UpdateChainInput) (domain.Chain, error) {
	if chainID == "" || input.Name == "" || input.DestinationScope == "" || len(input.Hops) == 0 {
		return domain.Chain{}, invalidInput("invalid_chain_payload")
	}
	return c.store.UpdateChain(chainID, input)
}

func (c *ControlPlane) DeleteChain(chainID string) error {
	return c.store.DeleteChain(chainID)
}

func (c *ControlPlane) ValidateChain(input domain.ValidateChainInput) (domain.ChainValidationResult, error) {
	result := domain.ChainValidationResult{
		Valid:           true,
		Errors:          []string{},
		Warnings:        []string{},
		HopConnectivity: []domain.HopConnectivity{},
	}

	if len(input.Hops) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "Chain must have at least one hop")
		return result, nil
	}

	nodes := c.store.ListNodes()
	links := c.store.ListNodeLinks()

	firstHopNode, ok := nodeByID(nodes, input.Hops[0])
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("First hop node %s not found", input.Hops[0]))
		return result, nil
	}

	if firstHopNode.Mode != domain.NodeModeEdge {
		result.Valid = false
		result.Errors = append(result.Errors, "First hop must be an edge node")
	}

	for i := 0; i < len(input.Hops)-1; i++ {
		fromNodeID := input.Hops[i]
		toNodeID := input.Hops[i+1]
		reachable := false

		for _, link := range links {
			if link.SourceNodeID == fromNodeID && link.TargetNodeID == toNodeID {
				reachable = true
				break
			}
		}

		result.HopConnectivity = append(result.HopConnectivity, domain.HopConnectivity{
			From:      fromNodeID,
			To:        toNodeID,
			Reachable: reachable,
		})

		if !reachable {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Node %s cannot reach node %s", fromNodeID, toNodeID))
		}
	}

	if len(input.Hops) > 0 {
		finalHopNodeID := input.Hops[len(input.Hops)-1]
		finalHopNode, ok := nodeByID(nodes, finalHopNodeID)
		if !ok {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Final hop node %s not found", finalHopNodeID))
		} else {
			scopeValid := finalHopNode.ScopeKey == input.DestinationScope
			result.ScopeOwnership = domain.ScopeOwnership{
				Scope:       input.DestinationScope,
				OwnerNodeID: finalHopNodeID,
				Valid:       scopeValid,
			}

			if !scopeValid {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Scope %s is not owned by final hop node %s", input.DestinationScope, finalHopNodeID))
			}
		}
	}

	return result, nil
}

func (c *ControlPlane) PreviewChain(input domain.PreviewChainInput) (domain.ChainPreviewResult, error) {
	nodes := c.store.ListNodes()
	hopDetails := make([]domain.ChainHopDetail, 0, len(input.Hops))
	routingPath := "user"

	for _, hopID := range input.Hops {
		node, ok := nodeByID(nodes, hopID)
		if !ok {
			return domain.ChainPreviewResult{}, invalidInput(fmt.Sprintf("node %s not found", hopID))
		}

		hopDetails = append(hopDetails, domain.ChainHopDetail{
			NodeID:   node.ID,
			NodeName: node.Name,
			Mode:     node.Mode,
		})

		routingPath += " → " + node.Name
	}

	routingPath += fmt.Sprintf(" → target(%s)", input.DestinationScope)

	return domain.ChainPreviewResult{
		CompiledConfig: domain.CompiledChainConfig{
			ChainID:          "preview",
			Name:             input.Name,
			Hops:             hopDetails,
			DestinationScope: input.DestinationScope,
			RoutingPath:      routingPath,
		},
	}, nil
}

func resolveProbeTransport(node domain.Node, prevHopID string, transports []domain.NodeTransport) (domain.NodeTransport, bool) {
	if prevHopID != "" {
		for _, transport := range transports {
			if transport.NodeID != node.ID || transport.ParentNodeID != prevHopID {
				continue
			}
			if transport.Status != domain.TransportStatusConnected {
				continue
			}
			if strings.HasPrefix(transport.TransportType, domain.TransportTypeReverseWS) || strings.HasPrefix(transport.TransportType, domain.TransportTypeChildWS) {
				return transport, true
			}
		}
	}
	for _, transport := range transports {
		if transport.NodeID != node.ID {
			continue
		}
		if transport.TransportType == domain.TransportTypePublicHTTPS || transport.TransportType == domain.TransportTypePublicHTTP {
			return transport, true
		}
	}
	return domain.NodeTransport{}, false
}

func toChainProbeInput(result domain.ChainProbeResult) domain.SaveChainProbeResultInput {
	return domain.SaveChainProbeResultInput{
		ChainID:        result.ChainID,
		Status:         result.Status,
		Message:        result.Message,
		ResolvedHops:   result.ResolvedHops,
		BlockingNodeID: result.BlockingNodeID,
		BlockingReason: result.BlockingReason,
		TargetHost:     result.TargetHost,
		TargetPort:     result.TargetPort,
		ProbedAt:       result.ProbedAt,
	}
}

func chainByID(items []domain.Chain, chainID string) (domain.Chain, bool) {
	for _, item := range items {
		if item.ID == chainID {
			return item, true
		}
	}
	return domain.Chain{}, false
}
