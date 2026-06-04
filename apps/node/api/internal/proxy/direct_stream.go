package proxy

import (
	"context"
	"errors"
	"net"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
)

var errStreamFallbackUnavailable = errors.New("stream_fallback_unavailable")

type directPeerStreamOpener interface {
	OpenDirectStream(ctx context.Context, nextHop domain.Node, remaining []string, targetHost string, targetPort int) (net.Conn, error)
}

type fallbackStreamOpener interface {
	OpenStream(nextNodeID string, remaining []string, targetHost string, targetPort int) (net.Conn, error)
}

type directPeerAvailability interface {
	HasDirectPeer(peerNodeID string) bool
}

func openDirectFirstStream(ctx context.Context, direct directPeerStreamOpener, fallback fallbackStreamOpener, hop chainHop, targetHost string, targetPort int) (net.Conn, error) {
	if direct != nil {
		conn, err := direct.OpenDirectStream(ctx, hop.node, append([]string(nil), hop.remainingHops...), targetHost, targetPort)
		if err == nil {
			return conn, nil
		}
	}
	if fallback == nil {
		return nil, errStreamFallbackUnavailable
	}
	return fallback.OpenStream(hop.node.ID, append([]string(nil), hop.remainingHops...), targetHost, targetPort)
}
