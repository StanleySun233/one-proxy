package store

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func defaultNodes() []domain.Node {
	return []domain.Node{
		{ID: "1", Name: "edge-a", Mode: "edge", ScopeKey: "public-edge", Enabled: true, Status: "healthy", PublicHost: "edge-a.example.com", PublicPort: 443},
		{ID: "2", Name: "relay-b", Mode: "relay", ScopeKey: "b-lan", ParentNodeID: "1", Enabled: true, Status: "healthy"},
		{ID: "3", Name: "relay-c", Mode: "relay", ScopeKey: "c-k8s", ParentNodeID: "2", Enabled: true, Status: "degraded"},
		{ID: "4", Name: "relay-d", Mode: "relay", ScopeKey: "d-office", ParentNodeID: "1", Enabled: true, Status: "healthy"},
	}
}

func defaultChains() []domain.Chain {
	return []domain.Chain{
		{ID: "chain-corp-k8s", Name: "corp-k8s", DestinationScope: "c-k8s", Enabled: true, Hops: []string{"1", "2", "3"}},
		{ID: "chain-office-tools", Name: "office-tools", DestinationScope: "d-office", Enabled: true, Hops: []string{"1", "4"}},
	}
}

func defaultRouteRules() []domain.RouteRule {
	return []domain.RouteRule{
		{ID: "rule-corp-domain", Priority: 100, MatchType: "domain_suffix", MatchValue: ".corp.internal", ActionType: "chain", ChainID: "chain-corp-k8s", DestinationScope: "c-k8s", Enabled: true},
		{ID: "rule-b-lan-cidr", Priority: 200, MatchType: "cidr", MatchValue: "10.30.0.0/16", ActionType: "chain", ChainID: "chain-corp-k8s", DestinationScope: "b-lan", Enabled: true},
		{ID: "rule-office-host", Priority: 300, MatchType: "domain", MatchValue: "grafana.office.local", ActionType: "chain", ChainID: "chain-office-tools", DestinationScope: "d-office", Enabled: true},
	}
}

func defaultNodeHealth() []domain.NodeHealth {
	return []domain.NodeHealth{
		{
			NodeID:           "1",
			HeartbeatAt:      "2026-04-25T12:00:00Z",
			PolicyRevisionID: "rev-0007",
			ListenerStatus:   map[string]string{"http": "up", "https": "up"},
			CertStatus:       map[string]string{"public": "renew-soon", "internal": "healthy"},
		},
		{
			NodeID:           "2",
			HeartbeatAt:      "2026-04-25T12:00:00Z",
			PolicyRevisionID: "rev-0007",
			ListenerStatus:   map[string]string{"relay": "up"},
			CertStatus:       map[string]string{"internal": "healthy"},
		},
		{
			NodeID:           "3",
			HeartbeatAt:      "2026-04-25T11:58:00Z",
			PolicyRevisionID: "rev-0007",
			ListenerStatus:   map[string]string{"relay": "degraded"},
			CertStatus:       map[string]string{"internal": "rotate"},
		},
	}
}

func defaultNodeLinks() []domain.NodeLink {
	return []domain.NodeLink{
		{ID: "link-1-2", SourceNodeID: "1", TargetNodeID: "2", LinkType: "parent_child", TrustState: "trusted"},
		{ID: "link-2-3", SourceNodeID: "2", TargetNodeID: "3", LinkType: "parent_child", TrustState: "trusted"},
		{ID: "link-1-4", SourceNodeID: "1", TargetNodeID: "4", LinkType: "parent_child", TrustState: "trusted"},
	}
}

func defaultCertificates() []domain.Certificate {
	return []domain.Certificate{
		{ID: "cert-1-public", OwnerType: "node", OwnerID: "1", CertType: "public", Status: "renew-soon", NotBefore: "2026-04-01T00:00:00Z", NotAfter: "2026-05-13T00:00:00Z"},
		{ID: "cert-2-internal", OwnerType: "node", OwnerID: "2", CertType: "internal", Status: "healthy", NotBefore: "2026-04-01T00:00:00Z", NotAfter: "2026-06-05T00:00:00Z"},
	}
}

func decodeJSONMap(raw string) map[string]string {
	if raw == "" {
		return map[string]string{}
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return map[string]string{}
	}
	return result
}

func encodeJSONMap(value map[string]string) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func decodeJSONStringSlice(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var result []string
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return []string{}
	}
	return result
}

func encodeJSONStringSlice(value []string) string {
	value = normalizeStringSlice(value)
	raw, err := json.Marshal(value)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func normalizeStringSlice(value []string) []string {
	if value == nil {
		return []string{}
	}
	return value
}

func newID(prefix string) string {
	if prefix == "node" {
		return "node-auto-increment"
	}
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
