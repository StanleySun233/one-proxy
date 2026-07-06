package proxy

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
)

type fakeDirectStreamOpener struct {
	err        error
	called     int
	nextHop    domain.Node
	remaining  []string
	targetHost string
	targetPort int
	conn       net.Conn
}

func (o *fakeDirectStreamOpener) OpenDirectStream(_ context.Context, nextHop domain.Node, remaining []string, targetHost string, targetPort int) (net.Conn, error) {
	o.called++
	o.nextHop = nextHop
	o.remaining = append([]string(nil), remaining...)
	o.targetHost = targetHost
	o.targetPort = targetPort
	if o.err != nil {
		return nil, o.err
	}
	if o.conn == nil {
		o.conn, _ = net.Pipe()
	}
	return o.conn, nil
}

type fakeFallbackStreamOpener struct {
	err        error
	called     int
	nextNodeID string
	remaining  []string
	targetHost string
	targetPort int
	conn       net.Conn
}

func (o *fakeFallbackStreamOpener) OpenStream(nextNodeID string, remaining []string, targetHost string, targetPort int) (net.Conn, error) {
	o.called++
	o.nextNodeID = nextNodeID
	o.remaining = append([]string(nil), remaining...)
	o.targetHost = targetHost
	o.targetPort = targetPort
	if o.err != nil {
		return nil, o.err
	}
	if o.conn == nil {
		o.conn, _ = net.Pipe()
	}
	return o.conn, nil
}

func TestOpenDirectFirstStreamUsesDirectPeerWhenAvailable(t *testing.T) {
	directConn, directPeer := net.Pipe()
	defer directPeer.Close()
	direct := &fakeDirectStreamOpener{conn: directConn}
	fallback := &fakeFallbackStreamOpener{}
	hop := chainHop{
		node: domain.Node{
			ID:         "node-2",
			PublicHost: "peer.local",
			PublicPort: 9443,
		},
		remainingHops: []string{"node-3"},
	}

	conn, err := openDirectFirstStream(context.Background(), direct, fallback, hop, "10.0.0.9", 22)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if conn != directConn {
		t.Fatal("direct connection was not returned")
	}
	if direct.called != 1 {
		t.Fatalf("direct calls = %d", direct.called)
	}
	if fallback.called != 0 {
		t.Fatalf("fallback calls = %d", fallback.called)
	}
	if direct.nextHop.ID != "node-2" || direct.nextHop.PublicHost != "peer.local" || direct.nextHop.PublicPort != 9443 {
		t.Fatalf("next hop = %+v", direct.nextHop)
	}
	if !reflect.DeepEqual(direct.remaining, []string{"node-3"}) {
		t.Fatalf("remaining = %v", direct.remaining)
	}
	if direct.targetHost != "10.0.0.9" || direct.targetPort != 22 {
		t.Fatalf("target = %s:%d", direct.targetHost, direct.targetPort)
	}
}

func TestOpenDirectFirstStreamFallsBackWhenDirectPeerFails(t *testing.T) {
	streamConn, streamPeer := net.Pipe()
	defer streamPeer.Close()
	direct := &fakeDirectStreamOpener{err: errors.New("direct_failed")}
	fallback := &fakeFallbackStreamOpener{conn: streamConn}
	hop := chainHop{
		node: domain.Node{
			ID:         "node-2",
			PublicHost: "peer.local",
			PublicPort: 9443,
		},
		remainingHops: []string{"node-3", "node-4"},
	}

	conn, err := openDirectFirstStream(context.Background(), direct, fallback, hop, "db.internal", 5432)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if conn != streamConn {
		t.Fatal("fallback connection was not returned")
	}
	if direct.called != 1 {
		t.Fatalf("direct calls = %d", direct.called)
	}
	if fallback.called != 1 {
		t.Fatalf("fallback calls = %d", fallback.called)
	}
	if fallback.nextNodeID != "node-2" {
		t.Fatalf("fallback next node = %q", fallback.nextNodeID)
	}
	if !reflect.DeepEqual(fallback.remaining, []string{"node-3", "node-4"}) {
		t.Fatalf("fallback remaining = %v", fallback.remaining)
	}
	if fallback.targetHost != "db.internal" || fallback.targetPort != 5432 {
		t.Fatalf("fallback target = %s:%d", fallback.targetHost, fallback.targetPort)
	}
}

func TestOpenDirectFirstStreamUsesDirectPeerWithoutPublicEndpoint(t *testing.T) {
	directConn, directPeer := net.Pipe()
	defer directPeer.Close()
	direct := &fakeDirectStreamOpener{conn: directConn}
	fallback := &fakeFallbackStreamOpener{}
	hop := chainHop{
		node:          domain.Node{ID: "node-2"},
		remainingHops: []string{"node-3"},
	}

	conn, err := openDirectFirstStream(context.Background(), direct, fallback, hop, "10.0.0.9", 22)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if conn != directConn {
		t.Fatal("direct connection was not returned")
	}
	if direct.called != 1 {
		t.Fatalf("direct calls = %d", direct.called)
	}
	if fallback.called != 0 {
		t.Fatalf("fallback calls = %d", fallback.called)
	}
}

func TestOpenDirectFirstStreamFallsBackWithoutPublicEndpoint(t *testing.T) {
	streamConn, streamPeer := net.Pipe()
	defer streamPeer.Close()
	direct := &fakeDirectStreamOpener{}
	direct.err = errors.New("direct_not_connected")
	fallback := &fakeFallbackStreamOpener{conn: streamConn}
	hop := chainHop{
		node:          domain.Node{ID: "node-2"},
		remainingHops: []string{"node-3"},
	}

	conn, err := openDirectFirstStream(context.Background(), direct, fallback, hop, "10.0.0.9", 22)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if direct.called != 1 {
		t.Fatalf("direct calls = %d", direct.called)
	}
	if fallback.called != 1 {
		t.Fatalf("fallback calls = %d", fallback.called)
	}
}

func TestOpenDirectFirstStreamRequiresFallbackAfterDirectMiss(t *testing.T) {
	direct := &fakeDirectStreamOpener{err: errors.New("direct_failed")}
	hop := chainHop{
		node: domain.Node{
			ID:         "node-2",
			PublicHost: "peer.local",
			PublicPort: 9443,
		},
	}

	_, err := openDirectFirstStream(context.Background(), direct, nil, hop, "10.0.0.9", 22)
	if !errors.Is(err, errStreamFallbackUnavailable) {
		t.Fatalf("err = %v", err)
	}
}

func TestShouldUseStreamAllowsPrivateNextHopWithDirectStreamOpener(t *testing.T) {
	server := &Server{directStream: &fakeDirectStreamOpener{}}
	if !server.shouldUseStream(domain.Node{ID: "node-2"}) {
		t.Fatal("private next hop did not use stream")
	}
	if server.shouldUseStream(domain.Node{ID: "node-2", PublicHost: "node-2.example", PublicPort: 9443}) {
		t.Fatal("public next hop used stream without a direct peer")
	}
}
