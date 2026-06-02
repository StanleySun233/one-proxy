package direct

import (
	"context"
	"encoding/json"
	"net"
	"time"
)

const PunchMessageType = "direct_punch"

type PunchMessage struct {
	Type       string `json:"type"`
	LinkID     string `json:"linkId"`
	NodeID     string `json:"nodeId"`
	PeerNodeID string `json:"peerNodeId"`
	PunchToken string `json:"punchToken"`
	Nonce      string `json:"nonce"`
	SentAt     string `json:"sentAt"`
}

type PunchResult struct {
	Message PunchMessage
	Addr    *net.UDPAddr
	RTT     time.Duration
}

func NewPunchMessage(linkID string, nodeID string, peerNodeID string, punchToken string, nonce string, now time.Time) PunchMessage {
	return PunchMessage{
		Type:       PunchMessageType,
		LinkID:     linkID,
		NodeID:     nodeID,
		PeerNodeID: peerNodeID,
		PunchToken: punchToken,
		Nonce:      nonce,
		SentAt:     now.UTC().Format(time.RFC3339Nano),
	}
}

func SendPunch(conn *net.UDPConn, addr *net.UDPAddr, message PunchMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = conn.WriteToUDP(payload, addr)
	return err
}

func AwaitPunch(ctx context.Context, conn *net.UDPConn, accept func(PunchMessage, *net.UDPAddr) bool) (PunchResult, error) {
	deadline, ok := ctx.Deadline()
	if ok {
		if err := conn.SetReadDeadline(deadline); err != nil {
			return PunchResult{}, err
		}
	}
	buffer := make([]byte, 2048)
	for {
		startedAt := time.Now()
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			return PunchResult{}, err
		}
		var message PunchMessage
		if err := json.Unmarshal(buffer[:n], &message); err != nil {
			continue
		}
		if message.Type != PunchMessageType {
			continue
		}
		if accept != nil && !accept(message, addr) {
			continue
		}
		return PunchResult{Message: message, Addr: addr, RTT: time.Since(startedAt)}, nil
	}
}
