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

func latencyMs(started time.Time) int64 {
	return durationMs(time.Since(started))
}

func durationMs(elapsed time.Duration) int64 {
	if elapsed <= 0 {
		return 0
	}
	ms := elapsed.Milliseconds()
	if ms == 0 {
		return 1
	}
	return ms
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
		return Result{Status: "failed", Message: "target_unreachable", LatencyMs: latencyMs(started)}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusInternalServerError {
		return Result{Status: "failed", Message: "target_unhealthy", LatencyMs: latencyMs(started)}
	}
	return Result{Status: "connected", Message: "target_reachable", LatencyMs: latencyMs(started)}
}

func probeWebSocket(protocol string, host string, port int) Result {
	started := time.Now()
	conn, resp, err := websocket.DefaultDialer.Dial(strings.ToLower(protocol)+"://"+host+":"+strconv.Itoa(port)+"/", nil)
	if err == nil {
		_ = conn.Close()
		return Result{Status: "connected", Message: "target_reachable", LatencyMs: latencyMs(started)}
	}
	if resp != nil && resp.StatusCode < http.StatusInternalServerError {
		return Result{Status: "connected", Message: "target_reachable", LatencyMs: latencyMs(started)}
	}
	return Result{Status: "failed", Message: "target_unreachable", LatencyMs: latencyMs(started)}
}

func probeTCP(host string, port int) Result {
	started := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 3*time.Second)
	if err != nil {
		return Result{Status: "failed", Message: "target_unreachable", LatencyMs: latencyMs(started)}
	}
	_ = conn.Close()
	return Result{Status: "connected", Message: "target_reachable", LatencyMs: latencyMs(started)}
}

func probeUDP(host string, port int) Result {
	started := time.Now()
	conn, err := net.DialTimeout("udp", net.JoinHostPort(host, strconv.Itoa(port)), 3*time.Second)
	if err != nil {
		return Result{Status: "failed", Message: "target_unreachable", LatencyMs: latencyMs(started)}
	}
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	_, _ = conn.Write([]byte{0})
	_ = conn.Close()
	return Result{Status: "connected", Message: "target_packet_sent", LatencyMs: latencyMs(started)}
}
