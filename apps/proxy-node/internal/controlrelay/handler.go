package controlrelay

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

func NewProbeHandler() http.HandlerFunc {
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
		result, statusCode := runProbe(payload)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(result)
	}
}

func runProbe(payload ProbeRequest) (ProbeResponse, int) {
	if payload.TargetHost == "" || payload.TargetPort <= 0 {
		return ProbeResponse{Status: "failed", Message: "invalid_target"}, http.StatusBadRequest
	}
	if len(payload.RemainingRelayURLs) > 0 {
		next := payload.RemainingRelayURLs[0]
		nextPayload := ProbeRequest{
			RemainingRelayURLs: payload.RemainingRelayURLs[1:],
			TargetHost:         payload.TargetHost,
			TargetPort:         payload.TargetPort,
		}
		result, err := Execute(next, nextPayload)
		if err != nil {
			return ProbeResponse{Status: "failed", Message: "relay_unreachable"}, http.StatusBadGateway
		}
		return result, http.StatusOK
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + payload.TargetHost + ":" + strconv.Itoa(payload.TargetPort) + "/healthz")
	if err != nil {
		return ProbeResponse{Status: "failed", Message: "target_unreachable"}, http.StatusBadGateway
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return ProbeResponse{Status: "failed", Message: "target_unhealthy"}, http.StatusBadGateway
	}
	return ProbeResponse{Status: "connected", Message: "target_reachable"}, http.StatusOK
}
