package direct

import (
	"context"
	"encoding/json"
	"net"
	"time"
)

const PunchMessageType = "direct_punch"
const punchPacketPrefix byte = 0

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

func SendPunch(packetIO PacketIO, addr *net.UDPAddr, message PunchMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	packet := make([]byte, 0, len(payload)+1)
	packet = append(packet, punchPacketPrefix)
	packet = append(packet, payload...)
	_, err = packetIO.WriteTo(packet, addr)
	return err
}

func AwaitPunch(ctx context.Context, packetIO PacketIO, accept func(PunchMessage, *net.UDPAddr) bool) (PunchResult, error) {
	buffer := make([]byte, 2048)
	for {
		startedAt := time.Now()
		n, addr, err := packetIO.ReadNonQUICPacket(ctx, buffer)
		if err != nil {
			return PunchResult{}, err
		}
		if n == 0 || buffer[0] != punchPacketPrefix {
			continue
		}
		var message PunchMessage
		if err := json.Unmarshal(buffer[1:n], &message); err != nil {
			continue
		}
		if message.Type != PunchMessageType {
			continue
		}
		udpAddr, ok := addr.(*net.UDPAddr)
		if !ok {
			continue
		}
		if accept != nil && !accept(message, udpAddr) {
			continue
		}
		return PunchResult{Message: message, Addr: udpAddr, RTT: time.Since(startedAt)}, nil
	}
}
