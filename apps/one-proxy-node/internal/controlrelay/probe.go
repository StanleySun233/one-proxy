package controlrelay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ProbeRequest struct {
	RemainingRelayURLs []string `json:"remainingRelayUrls"`
	RemainingHopNodeIDs []string `json:"remainingHopNodeIds"`
	TargetHost         string   `json:"targetHost"`
	TargetPort         int      `json:"targetPort"`
}

type ProbeResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func Execute(relayURL string, payload ProbeRequest) (ProbeResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return ProbeResponse{}, err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(relayURL, "/")+"/api/v1/control-relay/probe", bytes.NewReader(body))
	if err != nil {
		return ProbeResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return ProbeResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return ProbeResponse{}, fmt.Errorf("relay_probe_failed")
	}
	var result ProbeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ProbeResponse{}, err
	}
	return result, nil
}
