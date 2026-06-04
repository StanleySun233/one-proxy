package service

import (
	"testing"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func TestCompactNodeTransportsKeepsLatestReverseParentLink(t *testing.T) {
	items := []domain.NodeTransport{
		{
			ID:              "old",
			NodeID:          "child",
			TransportType:   domain.TransportTypeReverseWSParent,
			Direction:       "outbound",
			Address:         "ws://parent/api/v1/node-tunnel/connect?parentNodeId=parent",
			ParentNodeID:    "parent",
			LastHeartbeatAt: "2026-06-04T06:35:24Z",
		},
		{
			ID:              "new",
			NodeID:          "child",
			TransportType:   domain.TransportTypeReverseWSParent,
			Direction:       "outbound",
			Address:         "ws://parent/api/node/tunnel/connect?parentNodeId=parent",
			ParentNodeID:    "parent",
			LastHeartbeatAt: "2026-06-04T09:27:58Z",
		},
		{
			ID:            "public",
			NodeID:        "parent",
			TransportType: domain.TransportTypePublicHTTP,
			Direction:     "inbound",
			Address:       "http://parent:2988",
		},
	}

	result := compactNodeTransports(items)

	if len(result) != 2 {
		t.Fatalf("len = %d", len(result))
	}
	if result[0].ID != "new" || result[0].Address != "ws://parent/api/node/tunnel/connect?parentNodeId=parent" {
		t.Fatalf("reverse transport = %+v", result[0])
	}
	if result[1].ID != "public" {
		t.Fatalf("public transport = %+v", result[1])
	}
}
