package direct

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestPunchMessageRoundTrip(t *testing.T) {
	left, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer left.Close()
	right, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer right.Close()
	message := NewPunchMessage("link-1", "node-a", "node-b", "token", "nonce", time.Unix(1, 0))
	if err := SendPunch(UDPConnPacketIO{Conn: left}, right.LocalAddr().(*net.UDPAddr), message); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, err := AwaitPunch(ctx, UDPConnPacketIO{Conn: right}, func(candidate PunchMessage, addr *net.UDPAddr) bool {
		return candidate.LinkID == "link-1" && candidate.PunchToken == "token"
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Message.Nonce != "nonce" || result.Addr == nil {
		t.Fatalf("unexpected punch result: %#v", result)
	}
}
