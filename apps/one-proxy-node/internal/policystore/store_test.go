package policystore

import (
	"encoding/json"
	"testing"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
)

func TestStoreUpdateReadsSnapshotPayload(t *testing.T) {
	store := New("")
	payload, err := json.Marshal(Snapshot{
		Nodes: []domain.Node{{ID: "1"}},
		RouteRules: []domain.RouteRule{{
			ID:         "rule-1",
			MatchType:  domain.MatchTypeIPCIDR,
			MatchValue: "172.20.116.0/24",
			ActionType: domain.ActionTypeChain,
			ChainID:    "chain-1",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Update("1", string(payload)); err != nil {
		t.Fatal(err)
	}

	revision, snapshot := store.Current()
	if revision != "1" {
		t.Fatalf("revision = %q", revision)
	}
	if len(snapshot.Nodes) != 1 || len(snapshot.RouteRules) != 1 {
		t.Fatalf("snapshot = %+v", snapshot)
	}
}

func TestStoreUpdateReadsTenantWrappedPayload(t *testing.T) {
	store := New("")
	payload := `{"snapshots":[{"tenantId":"1","policyRevisionId":"3","payload":{"nodes":[{"id":"1"}],"links":[],"chains":[{"id":"chain-1","hops":["1","2"]}],"routeRules":[{"id":"rule-1","matchType":"ip_cidr","matchValue":"172.20.116.0/24","actionType":"chain","chainId":"chain-1"}]}}]}`

	if err := store.Update("3", payload); err != nil {
		t.Fatal(err)
	}

	revision, snapshot := store.Current()
	if revision != "3" {
		t.Fatalf("revision = %q", revision)
	}
	if len(snapshot.Nodes) != 1 || len(snapshot.Chains) != 1 || len(snapshot.RouteRules) != 1 {
		t.Fatalf("snapshot = %+v", snapshot)
	}
}
