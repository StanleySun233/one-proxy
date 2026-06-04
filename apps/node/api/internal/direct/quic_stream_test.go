package direct

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
	"github.com/quic-go/quic-go"
)

func TestRegistryOpenDirectStreamOverQUIC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	target, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	go func() {
		conn, err := target.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = io.Copy(conn, conn)
	}()

	leftConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer leftConn.Close()
	rightConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer rightConn.Close()

	left := NewRegistry()
	right := NewRegistry()
	leftTransport := &quic.Transport{Conn: leftConn}
	rightTransport := &quic.Transport{Conn: rightConn}
	if err := left.AttachQUICTransport(leftTransport); err != nil {
		t.Fatal(err)
	}
	if err := right.AttachQUICTransport(rightTransport); err != nil {
		t.Fatal(err)
	}
	go left.RunQUICServer(ctx)
	go right.RunQUICServer(ctx)

	rightUDP := rightConn.LocalAddr().(*net.UDPAddr)
	left.Upsert(PeerState{
		PeerNodeID: "node-b",
		Status:     domain.DirectStatusConnected,
		SelectedCandidate: domain.DirectCandidate{
			Type:     domain.CandidateTypeServerReflexive,
			Address:  rightUDP.IP.String(),
			Port:     rightUDP.Port,
			Protocol: domain.CandidateProtocolUDP,
		},
	})

	_, targetPortText, err := net.SplitHostPort(target.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	targetPort, err := net.LookupPort("tcp", targetPortText)
	if err != nil {
		t.Fatal(err)
	}
	stream, err := left.OpenDirectStream(ctx, domain.Node{ID: "node-b"}, nil, "127.0.0.1", targetPort)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	if _, err := stream.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 4)
	if _, err := io.ReadFull(stream, buffer); err != nil {
		t.Fatal(err)
	}
	if string(buffer) != "ping" {
		t.Fatalf("unexpected echo: %q", string(buffer))
	}
}
