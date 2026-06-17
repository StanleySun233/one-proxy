package service

import (
	"encoding/json"
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

func TestExtensionBootstrapSnapshotsUseLatestAccessPathContract(t *testing.T) {
	nodesByID := map[string]domain.Node{
		"edge": {ID: "edge", Name: "edge", Mode: domain.NodeModeEdge, ScopeKey: "edge-scope", PublicHost: "edge.example", PublicPort: 2988, Enabled: true},
	}
	chainsByID := map[string]proxy.Chain{
		"chain": {ID: "chain", DestinationScope: "edge-scope", Hops: []string{"edge"}, Enabled: true},
	}
	paths := []domain.NodeAccessPath{
		{
			ID:             "path",
			Name:           "Forward path",
			ChainID:        "chain",
			Mode:           "forward",
			Protocol:       "http",
			ServiceType:    "http_forward_proxy",
			TargetNodeID:   "edge",
			EntryNodeID:    "edge",
			ListenHost:     "127.0.0.1",
			ListenPort:     2988,
			TargetProtocol: "http",
			TargetHost:     "example.test",
			TargetPort:     80,
			AuthMode:       domain.AccessAuthProxyToken,
			Enabled:        true,
		},
	}
	rules := []proxy.RouteRule{
		{ID: "route", Priority: 10, MatchType: domain.MatchTypeDomainSuffix, MatchValue: ".example.test", ActionType: domain.ActionTypeChain, ChainID: "chain", Enabled: true},
		{ID: "orphan", Priority: 20, MatchType: domain.MatchTypeDomain, MatchValue: "orphan.test", ActionType: domain.ActionTypeChain, ChainID: "missing", Enabled: true},
	}

	accessPaths := extensionAccessPaths(paths, chainsByID, nodesByID, "2026-06-17T00:00:00Z")
	if len(accessPaths) != 1 {
		t.Fatalf("accessPaths len = %d", len(accessPaths))
	}
	if accessPaths[0].ID != "path" || accessPaths[0].Health.Status != "ready" || accessPaths[0].Topology[0].Transport != "public_http" {
		t.Fatalf("accessPath = %+v", accessPaths[0])
	}

	routes := extensionRoutes(rules, paths, chainsByID, nodesByID)
	if len(routes) != 1 {
		t.Fatalf("routes len = %d", len(routes))
	}
	if routes[0].ID != "route" || routes[0].AccessPathID != "path" || routes[0].Topology[0].NodeID != "edge" {
		t.Fatalf("route = %+v", routes[0])
	}

	bootstrap := domain.ExtensionBootstrap{
		SchemaVersion: "v2.1.0",
		Nodes:         []domain.Node{nodesByID["edge"]},
		AccessPaths:   accessPaths,
		Routes:        routes,
	}
	raw, err := json.Marshal(bootstrap)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if _, ok := payload["groups"]; ok {
		t.Fatalf("legacy groups field present: %s", raw)
	}
	if _, ok := payload["accessPaths"]; !ok {
		t.Fatalf("missing accessPaths field: %s", raw)
	}
	if _, ok := payload["routes"]; !ok {
		t.Fatalf("missing routes field: %s", raw)
	}
}
