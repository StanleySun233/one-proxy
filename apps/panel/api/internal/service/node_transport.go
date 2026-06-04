package service

import (
	"strings"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func (c *ControlPlane) NodeTransports(tenantCtx domain.TenantAuthContext) []domain.NodeTransport {
	allowed := c.tenantNodeIDs(tenantCtx)
	items := make([]domain.NodeTransport, 0)
	for _, item := range c.store.ListNodeTransports() {
		if allowed[item.NodeID] {
			items = append(items, item)
		}
	}
	return compactNodeTransports(items)
}

func (c *ControlPlane) UpsertNodeTransport(input domain.UpsertNodeTransportInput) (domain.NodeTransport, error) {
	if input.NodeID == "" || input.TransportType == "" || input.Direction == "" || input.Address == "" || input.Status == "" {
		return domain.NodeTransport{}, invalidInput("invalid_node_transport_payload")
	}
	return c.store.UpsertNodeTransport(input)
}

func (c *ControlPlane) UpsertNodeAgentTransport(nodeID string, input domain.UpsertNodeTransportInput) (domain.NodeTransport, error) {
	input.NodeID = nodeID
	return c.UpsertNodeTransport(input)
}

func compactNodeTransports(items []domain.NodeTransport) []domain.NodeTransport {
	result := make([]domain.NodeTransport, 0, len(items))
	reverseIndexByLink := make(map[string]int)
	for _, item := range items {
		if item.TransportType != domain.TransportTypeReverseWSParent || item.ParentNodeID == "" {
			result = append(result, item)
			continue
		}
		key := strings.Join([]string{item.NodeID, item.TransportType, item.Direction, item.ParentNodeID}, "|")
		index, ok := reverseIndexByLink[key]
		if !ok {
			reverseIndexByLink[key] = len(result)
			result = append(result, item)
			continue
		}
		if transportSeenAt(item) >= transportSeenAt(result[index]) {
			result[index] = item
		}
	}
	return result
}

func transportSeenAt(item domain.NodeTransport) string {
	if item.LastHeartbeatAt != "" {
		return item.LastHeartbeatAt
	}
	return item.ConnectedAt
}
