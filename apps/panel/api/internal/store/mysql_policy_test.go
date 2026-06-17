package store

import (
	"reflect"
	"testing"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	proxy "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/domain"
)

func TestDefaultAccessPathInputUsesPublicEntryNode(t *testing.T) {
	nodes := map[string]domain.Node{
		"edge":  {ID: "edge", PublicHost: "103.214.172.211", PublicPort: 2988},
		"relay": {ID: "relay"},
	}
	chain := proxy.Chain{ID: "chain", Name: "hk2astar", Hops: []string{"edge", "relay"}}

	got, ok := defaultAccessPathInput(chain, nodes)
	if !ok {
		t.Fatalf("defaultAccessPathInput ok = false")
	}

	if got.ChainID != "chain" || got.Name != "hk2astar default" {
		t.Fatalf("identity = %+v", got)
	}
	if got.Mode != domain.PathModeForward || got.Protocol != domain.AccessProtocolHTTP || got.ServiceType != domain.AccessServiceHTTPForwardProxy {
		t.Fatalf("mode/protocol/service = %+v", got)
	}
	if got.EntryNodeID != "edge" || got.TargetNodeID != "relay" {
		t.Fatalf("nodes = %+v", got)
	}
	if got.ListenHost != "103.214.172.211" || got.ListenPort != 2988 {
		t.Fatalf("listen = %+v", got)
	}
	if got.TargetProtocol != domain.AccessProtocolHTTP || got.TargetPort != 2988 || got.AuthMode != domain.AccessAuthProxyToken {
		t.Fatalf("target/auth = %+v", got)
	}
	if len(got.RelayNodeIDs) != 0 {
		t.Fatalf("relay nodes = %+v", got.RelayNodeIDs)
	}
}

func TestDefaultAccessPathInputKeepsIntermediateRelayNodes(t *testing.T) {
	nodes := map[string]domain.Node{
		"edge":   {ID: "edge", PublicHost: "203.0.113.10", PublicPort: 2988},
		"relay1": {ID: "relay1"},
		"relay2": {ID: "relay2"},
		"target": {ID: "target"},
	}
	chain := proxy.Chain{ID: "chain", Name: "multi-hop", Hops: []string{"edge", "relay1", "relay2", "target"}}

	got, ok := defaultAccessPathInput(chain, nodes)
	if !ok {
		t.Fatalf("defaultAccessPathInput ok = false")
	}

	if got.EntryNodeID != "edge" || got.TargetNodeID != "target" {
		t.Fatalf("nodes = %+v", got)
	}
	if want := []string{"relay1", "relay2"}; !reflect.DeepEqual(got.RelayNodeIDs, want) {
		t.Fatalf("relay nodes = %+v, want %+v", got.RelayNodeIDs, want)
	}
}

func TestDefaultAccessPathInputRejectsMissingHopNode(t *testing.T) {
	nodes := map[string]domain.Node{
		"edge": {ID: "edge", PublicHost: "203.0.113.10", PublicPort: 2988},
	}
	chain := proxy.Chain{ID: "chain", Name: "broken", Hops: []string{"edge", "missing"}}

	if _, ok := defaultAccessPathInput(chain, nodes); ok {
		t.Fatalf("defaultAccessPathInput ok = true")
	}
}
