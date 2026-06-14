package service

import (
	"context"
	"time"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

const (
	slaHeartbeatInterval = 10 * time.Second
	slaAggregationDelay  = 15 * time.Second
)

func (c *ControlPlane) RecordNodeSLAMinute() error {
	end := time.Now().UTC().Add(-slaAggregationDelay).Truncate(time.Minute)
	start := end.Add(-time.Minute)
	expected := int(end.Sub(start) / slaHeartbeatInterval)
	for _, node := range c.store.ListNodes() {
		if !node.Enabled {
			continue
		}
		received, err := c.slaHeartbeats.Count(context.Background(), node.ID, start, end)
		if err != nil {
			return err
		}
		success := 0
		if received >= expected {
			success = 1
		}
		if err := c.store.UpsertNodeSLAMinute(domain.NodeSLAMinuteInput{
			NodeID:             node.ID,
			WindowStart:        start.Format(time.RFC3339),
			ExpectedHeartbeats: expected,
			ReceivedHeartbeats: received,
			Success:            success,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (c *ControlPlane) NodeSLAMinutes(tenantCtx domain.TenantAuthContext, window time.Duration) ([]domain.NodeSLAMinute, error) {
	if window <= 0 || window > 7*24*time.Hour {
		window = 24 * time.Hour
	}
	since := time.Now().UTC().Add(-window).Format(time.RFC3339)
	allowed := c.tenantNodeIDs(tenantCtx)
	items, err := c.store.ListNodeSLAMinutes(since)
	if err != nil {
		return nil, err
	}
	filtered := make([]domain.NodeSLAMinute, 0, len(items))
	for _, item := range items {
		if allowed[item.NodeID] {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}
