package tunnel

import (
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/probe"
)

func (c *Controller) handleProbeRequest(message Message) Message {
	if len(message.RemainingHopNodeIDs) == 0 {
		if message.TargetHost != "" && message.TargetPort > 0 {
			return probeTarget(c.currentNodeID(), message.Protocol, message.TargetHost, message.TargetPort)
		}
		return Message{Status: "connected", Message: "chain_reachable"}
	}
	nextNodeID := message.RemainingHopNodeIDs[0]
	response, err := c.registry.ForwardProbe(c.currentNodeID(), nextNodeID, message.RequestID, message.RemainingHopNodeIDs[1:], message.Protocol, message.TargetHost, message.TargetPort)
	if err != nil {
		return Message{Status: "failed", Message: "next_hop_unreachable"}
	}
	return response
}

func (c *Controller) currentNodeID() string {
	if c.manager == nil || !c.manager.Bound() {
		return ""
	}
	return c.manager.Current().NodeID
}

func probeTarget(nodeID string, protocol string, host string, port int) Message {
	result := probe.Run(protocol, host, port)
	message := Message{Status: result.Status, Message: result.Message}
	if nodeID != "" && host != "" {
		now := time.Now().UTC()
		message.PathTimings = []PathTiming{{
			FromNodeID:  nodeID,
			ToNodeID:    host,
			RoundTripMs: result.LatencyMs,
			SampleTSMs:  now.UnixMilli(),
			Count:       1,
		}}
	}
	return message
}
