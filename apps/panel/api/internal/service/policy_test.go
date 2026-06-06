package service

import (
	"testing"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	proxy "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/domain"
)

func TestExtensionBootstrapNodesIncludesAuthorizedChainHops(t *testing.T) {
	allNodes := []domain.Node{
		{ID: "edge", Name: "edge", Mode: domain.NodeModeEdge, ScopeKey: "edge-scope", PublicHost: "edge.example", PublicPort: 2988, Enabled: true},
		{ID: "relay", Name: "relay", Mode: domain.NodeModeRelay, ScopeKey: "target-scope", ParentNodeID: "edge", Enabled: true},
	}
	chains := []proxy.Chain{
		{ID: "chain", DestinationScope: "target-scope", Hops: []string{"edge", "relay"}, Enabled: true},
	}
	rules := []proxy.RouteRule{
		{ID: "route", ActionType: domain.ActionTypeChain, ChainID: "chain", Enabled: true},
	}

	result := extensionBootstrapNodes(nil, allNodes, chains, rules)

	if len(result) != 2 {
		t.Fatalf("len = %d", len(result))
	}
	if result[0].ID != "edge" || result[1].ID != "relay" {
		t.Fatalf("nodes = %+v", result)
	}
}
