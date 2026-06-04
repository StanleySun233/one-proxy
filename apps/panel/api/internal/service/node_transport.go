package service

import "github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"

func (c *ControlPlane) NodeTransports(tenantCtx domain.TenantAuthContext) []domain.NodeTransport {
	allowed := c.tenantNodeIDs(tenantCtx)
	items := make([]domain.NodeTransport, 0)
	for _, item := range c.store.ListNodeTransports() {
		if allowed[item.NodeID] {
			items = append(items, item)
		}
	}
	return items
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
