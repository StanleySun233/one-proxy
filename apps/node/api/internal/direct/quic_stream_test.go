package direct

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
	"github.com/quic-go/quic-go"
)

func TestRegistryOpenDirectStreamRequiresPeerIdentity(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	leftConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer leftConn.Close()

	left := NewRegistry()
	leftTransport := &quic.Transport{Conn: leftConn}
	if err := left.AttachQUICTransport(leftTransport, "node-a"); err != nil {
		t.Fatal(err)
	}

	left.Upsert(PeerState{
		PeerNodeID: "node-b",
		Status:     domain.DirectStatusConnected,
		SelectedCandidate: domain.DirectCandidate{
			Type:     domain.CandidateTypeServerReflexive,
			Address:  "127.0.0.1",
			Port:     2992,
			Protocol: domain.CandidateProtocolUDP,
		},
	})

	_, err = left.OpenDirectStream(ctx, domain.Node{ID: "node-b"}, nil, "127.0.0.1", 80)
	if err == nil || err.Error() != "invalid_direct_node_identity" {
		t.Fatalf("err = %v", err)
	}
}

func TestClientTLSConfigUsesPeerIdentity(t *testing.T) {
	_, identity, err := serverTLSConfig("node-b")
	if err != nil {
		t.Fatal(err)
	}
	config, err := clientTLSConfig(identity)
	if err != nil {
		t.Fatal(err)
	}
	if config.InsecureSkipVerify {
		t.Fatal("insecure TLS verification is enabled")
	}
	if config.ServerName != identity.ServerName {
		t.Fatalf("server name = %q", config.ServerName)
	}
	if config.RootCAs == nil {
		t.Fatal("expected peer-scoped root CAs")
	}
}

func TestClientTLSConfigRejectsFingerprintMismatch(t *testing.T) {
	_, identity, err := serverTLSConfig("node-b")
	if err != nil {
		t.Fatal(err)
	}
	identity.CertificateFingerprintSHA256 = "00"
	_, err = clientTLSConfig(identity)
	if err == nil || err.Error() != "invalid_direct_node_identity" {
		t.Fatalf("err = %v", err)
	}
}
