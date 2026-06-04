package probe

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type Result struct {
	Status    string
	Message   string
	LatencyMs int64
}

func Run(protocol string, host string, port int) Result {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "http", "https":
		return probeHTTP(protocol, host, port)
	case "ws", "wss":
		return probeWebSocket(protocol, host, port)
	case "udp":
		return probeUDP(host, port)
	default:
		return probeTCP(host, port)
	}
}

func probeHTTP(protocol string, host string, port int) Result {
	started := time.Now()
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(strings.ToLower(protocol) + "://" + host + ":" + strconv.Itoa(port) + "/healthz")
	if err != nil {
		return Result{Status: "failed", Message: "target_unreachable", LatencyMs: time.Since(started).Milliseconds()}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusInternalServerError {
		return Result{Status: "failed", Message: "target_unhealthy", LatencyMs: time.Since(started).Milliseconds()}
	}
	return Result{Status: "connected", Message: "target_reachable", LatencyMs: time.Since(started).Milliseconds()}
}

func probeWebSocket(protocol string, host string, port int) Result {
	started := time.Now()
	conn, resp, err := websocket.DefaultDialer.Dial(strings.ToLower(protocol)+"://"+host+":"+strconv.Itoa(port)+"/", nil)
	if err == nil {
		_ = conn.Close()
		return Result{Status: "connected", Message: "target_reachable", LatencyMs: time.Since(started).Milliseconds()}
	}
	if resp != nil && resp.StatusCode < http.StatusInternalServerError {
		return Result{Status: "connected", Message: "target_reachable", LatencyMs: time.Since(started).Milliseconds()}
	}
	return Result{Status: "failed", Message: "target_unreachable", LatencyMs: time.Since(started).Milliseconds()}
}

func probeTCP(host string, port int) Result {
	started := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 3*time.Second)
	if err != nil {
		return Result{Status: "failed", Message: "target_unreachable", LatencyMs: time.Since(started).Milliseconds()}
	}
	_ = conn.Close()
	return Result{Status: "connected", Message: "target_reachable", LatencyMs: time.Since(started).Milliseconds()}
}

func probeUDP(host string, port int) Result {
	started := time.Now()
	conn, err := net.DialTimeout("udp", net.JoinHostPort(host, strconv.Itoa(port)), 3*time.Second)
	if err != nil {
		return Result{Status: "failed", Message: "target_unreachable", LatencyMs: time.Since(started).Milliseconds()}
	}
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	_, _ = conn.Write([]byte{0})
	_ = conn.Close()
	return Result{Status: "connected", Message: "target_packet_sent", LatencyMs: time.Since(started).Milliseconds()}
}
