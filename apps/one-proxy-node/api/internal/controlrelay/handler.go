package controlrelay

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/probe"
	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/tunnel"
)

func NewProbeHandler(registry *tunnel.Registry, nodeIDGetter func() string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var payload ProbeRequest
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		result, statusCode := runProbe(payload, registry, nodeIDGetter)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(result)
	}
}

func runProbe(payload ProbeRequest, registry *tunnel.Registry, nodeIDGetter func() string) (ProbeResponse, int) {
	if len(payload.RemainingHopNodeIDs) > 0 {
		next := payload.RemainingHopNodeIDs[0]
		response, err := registry.ForwardProbe(currentNodeID(nodeIDGetter), next, time.Now().UTC().Format(time.RFC3339Nano), payload.RemainingHopNodeIDs[1:], payload.Protocol, payload.TargetHost, payload.TargetPort)
		if err != nil {
			return ProbeResponse{Status: "failed", Message: "relay_unreachable"}, http.StatusBadGateway
		}
		return ProbeResponse{Status: response.Status, Message: response.Message, PathTimings: response.PathTimings}, http.StatusOK
	}
	if payload.TargetHost == "" || payload.TargetPort <= 0 {
		return ProbeResponse{Status: "connected", Message: "chain_reachable"}, http.StatusOK
	}
	if len(payload.RemainingRelayURLs) > 0 {
		next := payload.RemainingRelayURLs[0]
		nextPayload := ProbeRequest{
			RemainingRelayURLs: payload.RemainingRelayURLs[1:],
			Protocol:           payload.Protocol,
			TargetHost:         payload.TargetHost,
			TargetPort:         payload.TargetPort,
		}
		result, err := Execute(next, nextPayload)
		if err != nil {
			return ProbeResponse{Status: "failed", Message: "relay_unreachable"}, http.StatusBadGateway
		}
		return result, http.StatusOK
	}
	return probeTarget(currentNodeID(nodeIDGetter), payload)
}

func currentNodeID(nodeIDGetter func() string) string {
	if nodeIDGetter == nil {
		return ""
	}
	return nodeIDGetter()
}

func probeTarget(nodeID string, payload ProbeRequest) (ProbeResponse, int) {
	result := probe.Run(payload.Protocol, payload.TargetHost, payload.TargetPort)
	statusCode := http.StatusOK
	if result.Status == "failed" {
		statusCode = http.StatusBadGateway
	}
	response := ProbeResponse{Status: result.Status, Message: result.Message}
	if nodeID != "" && payload.TargetHost != "" {
		now := time.Now().UTC()
		response.PathTimings = []tunnel.PathTiming{{
			FromNodeID:  nodeID,
			ToNodeID:    payload.TargetHost,
			RoundTripMs: result.LatencyMs,
			SampleTSMs:  now.UnixMilli(),
			Count:       1,
		}}
	}
	return response, statusCode
}
