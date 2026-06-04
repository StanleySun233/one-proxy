package proxyservice

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/controlrelay"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
	proxy "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/features/proxy/domain"
)

func (s *Service) Chains(tenantCtx domain.TenantAuthContext) []proxy.Chain {
	return s.store.ListChainsForTenant(tenantCtx)
}

func (s *Service) ChainsWithDetails(tenantCtx domain.TenantAuthContext) []proxy.ChainWithDetails {
	chains := s.store.ListChainsForTenant(tenantCtx)
	nodes := s.store.ListNodesForTenant(tenantCtx)
	result := make([]proxy.ChainWithDetails, 0, len(chains))

	for _, chain := range chains {
		hopDetails := make([]proxy.ChainHopDetail, 0, len(chain.Hops))
		for _, hopID := range chain.Hops {
			node, ok := nodeByID(nodes, hopID)
			if ok {
				hopDetails = append(hopDetails, proxy.ChainHopDetail{
					NodeID:   node.ID,
					NodeName: node.Name,
					Mode:     node.Mode,
				})
			}
		}

		result = append(result, proxy.ChainWithDetails{
			ID:               chain.ID,
			CreateID:         chain.CreateID,
			OwnerID:          chain.OwnerID,
			Name:             chain.Name,
			DestinationScope: chain.DestinationScope,
			Enabled:          chain.Enabled,
			Hops:             chain.Hops,
			HopDetails:       hopDetails,
			Permission:       chain.Permission,
		})
	}

	return result
}

func (s *Service) GetChain(tenantCtx domain.TenantAuthContext, chainID string) (proxy.ChainWithDetails, error) {
	if chainID == "" {
		return proxy.ChainWithDetails{}, invalidInput("missing_chain_id")
	}

	chains := s.store.ListChainsForTenant(tenantCtx)
	chain, ok := chainByID(chains, chainID)
	if !ok {
		return proxy.ChainWithDetails{}, invalidInput("chain_not_found")
	}

	nodes := s.store.ListNodesForTenant(tenantCtx)
	hopDetails := make([]proxy.ChainHopDetail, 0, len(chain.Hops))
	for _, hopID := range chain.Hops {
		node, ok := nodeByID(nodes, hopID)
		if ok {
			hopDetails = append(hopDetails, proxy.ChainHopDetail{
				NodeID:   node.ID,
				NodeName: node.Name,
				Mode:     node.Mode,
			})
		}
	}

	return proxy.ChainWithDetails{
		ID:               chain.ID,
		CreateID:         chain.CreateID,
		OwnerID:          chain.OwnerID,
		Name:             chain.Name,
		DestinationScope: chain.DestinationScope,
		Enabled:          chain.Enabled,
		Hops:             chain.Hops,
		HopDetails:       hopDetails,
		Permission:       chain.Permission,
	}, nil
}

func (s *Service) LatestChainProbe(tenantCtx domain.TenantAuthContext, chainID string) (proxy.ChainProbeResult, bool) {
	if chainID == "" {
		return proxy.ChainProbeResult{}, false
	}
	if _, ok := s.store.ChainBindingPermission(tenantCtx, chainID); !ok {
		return proxy.ChainProbeResult{}, false
	}
	return s.store.GetChainProbeResult(chainID)
}

func (s *Service) ProbeChain(tenantCtx domain.TenantAuthContext, chainID string) (proxy.ChainProbeResult, error) {
	if chainID == "" {
		return proxy.ChainProbeResult{}, invalidInput("missing_chain_id")
	}
	chain, ok := chainByID(s.store.ListChainsForTenant(tenantCtx), chainID)
	if !ok {
		return proxy.ChainProbeResult{}, invalidInput("invalid_chain_id")
	}
	nodes := s.store.ListNodesForTenant(tenantCtx)
	transports := s.store.ListNodeTransports()
	result := proxy.ChainProbeResult{
		ChainID:      chainID,
		Status:       domain.ProbeResultStatusConnected,
		Message:      "chain_transport_ready",
		ResolvedHops: make([]proxy.ChainProbeHop, 0, len(chain.Hops)),
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
			return s.store.SaveChainProbeResult(toChainProbeInput(result))
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
			return s.store.SaveChainProbeResult(toChainProbeInput(result))
		}
		result.ResolvedHops = append(result.ResolvedHops, proxy.ChainProbeHop{
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
			return s.store.SaveChainProbeResult(toChainProbeInput(result))
		}
		result.Status = probeResult.Status
		result.Message = probeResult.Message
		if probeResult.Status != domain.ProbeResultStatusConnected && result.BlockingReason == "" && len(chain.Hops) > 0 {
			result.BlockingNodeID = chain.Hops[len(chain.Hops)-1]
			result.BlockingReason = probeResult.Message
		}
	}
	return s.store.SaveChainProbeResult(toChainProbeInput(result))
}

func (s *Service) CreateChain(tenantCtx domain.TenantAuthContext, input proxy.CreateChainInput) (proxy.Chain, error) {
	if err := requireActiveTenant(tenantCtx); err != nil {
		return proxy.Chain{}, err
	}
	if !tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.Role != domain.TenantRoleAdmin {
		return proxy.Chain{}, newError(http.StatusForbidden, "tenant_role_forbidden")
	}
	if input.Name == "" || input.DestinationScope == "" || len(input.Hops) == 0 {
		return proxy.Chain{}, invalidInput("invalid_chain_payload")
	}
	if !s.tenantScopeExists(tenantCtx, input.DestinationScope) {
		return proxy.Chain{}, invalidInput("scope_not_found")
	}
	if !s.tenantNodesExist(tenantCtx, input.Hops) {
		return proxy.Chain{}, invalidInput("node_not_found")
	}
	return s.store.CreateChainForTenant(tenantCtx, input)
}

func (s *Service) UpdateChain(tenantCtx domain.TenantAuthContext, chainID string, input proxy.UpdateChainInput) (proxy.Chain, error) {
	if chainID == "" || input.Name == "" || input.DestinationScope == "" || len(input.Hops) == 0 {
		return proxy.Chain{}, invalidInput("invalid_chain_payload")
	}
	if err := s.requireTenantResourceManage(tenantCtx, func() (domain.BindingPermission, bool) {
		return s.store.ChainBindingPermission(tenantCtx, chainID)
	}); err != nil {
		return proxy.Chain{}, err
	}
	if !s.tenantScopeExists(tenantCtx, input.DestinationScope) {
		return proxy.Chain{}, invalidInput("scope_not_found")
	}
	if !s.tenantNodesExist(tenantCtx, input.Hops) {
		return proxy.Chain{}, invalidInput("node_not_found")
	}
	return s.store.UpdateChain(chainID, input)
}

func (s *Service) DeleteChain(tenantCtx domain.TenantAuthContext, chainID string) error {
	if err := s.requireTenantResourceManage(tenantCtx, func() (domain.BindingPermission, bool) {
		return s.store.ChainBindingPermission(tenantCtx, chainID)
	}); err != nil {
		return err
	}
	if !tenantCtx.SuperAdmin && s.store.CountChainBindings(chainID) > 1 {
		return newError(http.StatusConflict, "shared_resource_delete_forbidden")
	}
	return s.store.DeleteChain(chainID)
}

func (s *Service) ValidateChain(tenantCtx domain.TenantAuthContext, input proxy.ValidateChainInput) (proxy.ChainValidationResult, error) {
	result := proxy.ChainValidationResult{
		Valid:           true,
		Errors:          []string{},
		Warnings:        []string{},
		HopConnectivity: []proxy.HopConnectivity{},
	}

	if len(input.Hops) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "Chain must have at least one hop")
		return result, nil
	}

	nodes := s.store.ListNodesForTenant(tenantCtx)
	links := s.store.ListNodeLinksForTenant(tenantCtx)

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

		for _, nodeLink := range links {
			if nodeLink.SourceNodeID == fromNodeID && nodeLink.TargetNodeID == toNodeID {
				reachable = true
				break
			}
		}

		result.HopConnectivity = append(result.HopConnectivity, proxy.HopConnectivity{
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
			result.ScopeOwnership = proxy.ScopeOwnership{
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

func (s *Service) PreviewChain(tenantCtx domain.TenantAuthContext, input proxy.PreviewChainInput) (proxy.ChainPreviewResult, error) {
	nodes := s.store.ListNodesForTenant(tenantCtx)
	hopDetails := make([]proxy.ChainHopDetail, 0, len(input.Hops))
	routingPath := "user"

	for _, hopID := range input.Hops {
		node, ok := nodeByID(nodes, hopID)
		if !ok {
			return proxy.ChainPreviewResult{}, invalidInput(fmt.Sprintf("node %s not found", hopID))
		}

		hopDetails = append(hopDetails, proxy.ChainHopDetail{
			NodeID:   node.ID,
			NodeName: node.Name,
			Mode:     node.Mode,
		})

		routingPath += " → " + node.Name
	}

	routingPath += fmt.Sprintf(" → target(%s)", input.DestinationScope)

	return proxy.ChainPreviewResult{
		CompiledConfig: proxy.CompiledChainConfig{
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

func toChainProbeInput(result proxy.ChainProbeResult) proxy.SaveChainProbeResultInput {
	return proxy.SaveChainProbeResultInput{
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

func chainByID(items []proxy.Chain, chainID string) (proxy.Chain, bool) {
	for _, item := range items {
		if item.ID == chainID {
			return item, true
		}
	}
	return proxy.Chain{}, false
}
