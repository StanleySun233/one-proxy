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
	Status  string
	Message string
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
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(strings.ToLower(protocol) + "://" + host + ":" + strconv.Itoa(port) + "/")
	if err != nil {
		return Result{Status: "failed", Message: "target_unreachable"}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusInternalServerError {
		return Result{Status: "failed", Message: "target_unhealthy"}
	}
	return Result{Status: "connected", Message: "target_reachable"}
}

func probeWebSocket(protocol string, host string, port int) Result {
	conn, resp, err := websocket.DefaultDialer.Dial(strings.ToLower(protocol)+"://"+host+":"+strconv.Itoa(port)+"/", nil)
	if err == nil {
		_ = conn.Close()
		return Result{Status: "connected", Message: "target_reachable"}
	}
	if resp != nil && resp.StatusCode < http.StatusInternalServerError {
		return Result{Status: "connected", Message: "target_reachable"}
	}
	return Result{Status: "failed", Message: "target_unreachable"}
}

func probeTCP(host string, port int) Result {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 3*time.Second)
	if err != nil {
		return Result{Status: "failed", Message: "target_unreachable"}
	}
	_ = conn.Close()
	return Result{Status: "connected", Message: "target_reachable"}
}

func probeUDP(host string, port int) Result {
	conn, err := net.DialTimeout("udp", net.JoinHostPort(host, strconv.Itoa(port)), 3*time.Second)
	if err != nil {
		return Result{Status: "failed", Message: "target_unreachable"}
	}
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	_, _ = conn.Write([]byte{0})
	_ = conn.Close()
	return Result{Status: "connected", Message: "target_packet_sent"}
}
