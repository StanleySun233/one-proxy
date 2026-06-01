package controlrelay

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/tunnel"
	"github.com/gorilla/websocket"
)

func NewProbeHandler(registry *tunnel.Registry) http.HandlerFunc {
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
		result, statusCode := runProbe(payload, registry)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(result)
	}
}

func runProbe(payload ProbeRequest, registry *tunnel.Registry) (ProbeResponse, int) {
	if len(payload.RemainingHopNodeIDs) > 0 {
		next := payload.RemainingHopNodeIDs[0]
		response, err := registry.ForwardProbe(next, time.Now().UTC().Format(time.RFC3339Nano), payload.RemainingHopNodeIDs[1:], payload.Protocol, payload.TargetHost, payload.TargetPort)
		if err != nil {
			return ProbeResponse{Status: "failed", Message: "relay_unreachable"}, http.StatusBadGateway
		}
		return ProbeResponse{Status: response.Status, Message: response.Message}, http.StatusOK
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
	return probeTarget(payload)
}

func probeTarget(payload ProbeRequest) (ProbeResponse, int) {
	protocol := strings.ToLower(strings.TrimSpace(payload.Protocol))
	if protocol == "" {
		protocol = "tcp"
	}
	switch protocol {
	case "http", "https":
		return probeHTTP(protocol, payload.TargetHost, payload.TargetPort)
	case "ws", "wss":
		return probeWebSocket(protocol, payload.TargetHost, payload.TargetPort)
	case "udp":
		return probeUDP(payload.TargetHost, payload.TargetPort)
	default:
		return probeTCP(payload.TargetHost, payload.TargetPort)
	}
}

func probeHTTP(protocol string, host string, port int) (ProbeResponse, int) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(protocol + "://" + host + ":" + strconv.Itoa(port) + "/")
	if err != nil {
		return ProbeResponse{Status: "failed", Message: "target_unreachable"}, http.StatusBadGateway
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusInternalServerError {
		return ProbeResponse{Status: "failed", Message: "target_unhealthy"}, http.StatusBadGateway
	}
	return ProbeResponse{Status: "connected", Message: "target_reachable"}, http.StatusOK
}

func probeWebSocket(protocol string, host string, port int) (ProbeResponse, int) {
	conn, resp, err := websocket.DefaultDialer.Dial(protocol+"://"+host+":"+strconv.Itoa(port)+"/", nil)
	if err == nil {
		_ = conn.Close()
		return ProbeResponse{Status: "connected", Message: "target_reachable"}, http.StatusOK
	}
	if resp != nil && resp.StatusCode < http.StatusInternalServerError {
		return ProbeResponse{Status: "connected", Message: "target_reachable"}, http.StatusOK
	}
	return ProbeResponse{Status: "failed", Message: "target_unreachable"}, http.StatusBadGateway
}

func probeTCP(host string, port int) (ProbeResponse, int) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 3*time.Second)
	if err != nil {
		return ProbeResponse{Status: "failed", Message: "target_unreachable"}, http.StatusBadGateway
	}
	_ = conn.Close()
	return ProbeResponse{Status: "connected", Message: "target_reachable"}, http.StatusOK
}

func probeUDP(host string, port int) (ProbeResponse, int) {
	conn, err := net.DialTimeout("udp", net.JoinHostPort(host, strconv.Itoa(port)), 3*time.Second)
	if err != nil {
		return ProbeResponse{Status: "failed", Message: "target_unreachable"}, http.StatusBadGateway
	}
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	_, _ = conn.Write([]byte{0})
	_ = conn.Close()
	return ProbeResponse{Status: "connected", Message: "target_packet_sent"}, http.StatusOK
}
