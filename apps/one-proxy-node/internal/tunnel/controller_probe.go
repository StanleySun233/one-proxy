package tunnel

import "github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/probe"

func (c *Controller) handleProbeRequest(message Message) Message {
	if len(message.RemainingHopNodeIDs) == 0 {
		if message.TargetHost != "" && message.TargetPort > 0 {
			return probeTarget(message.Protocol, message.TargetHost, message.TargetPort)
		}
		return Message{Status: "connected", Message: "chain_reachable"}
	}
	nextNodeID := message.RemainingHopNodeIDs[0]
	response, err := c.registry.ForwardProbe(nextNodeID, message.RequestID, message.RemainingHopNodeIDs[1:], message.Protocol, message.TargetHost, message.TargetPort)
	if err != nil {
		return Message{Status: "failed", Message: "next_hop_unreachable"}
	}
	return response
}

func probeTarget(protocol string, host string, port int) Message {
	result := probe.Run(protocol, host, port)
	return Message{Status: result.Status, Message: result.Message}
}
