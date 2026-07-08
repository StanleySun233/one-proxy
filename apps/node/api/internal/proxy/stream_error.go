package proxy

import "github.com/StanleySun233/python-proxy/apps/node/api/internal/tunnel"

func proxyErrorForStreamFailure(err error) string {
	if tunnel.IsTunnelUnavailable(err) {
		return proxyErrorRelayTunnelUnavailable
	}
	return proxyErrorNextHopConnectFailed
}

func proxyErrorForTunnelCopyErrors(uploadErr error, downloadErr error) string {
	if tunnel.IsTunnelUnavailable(uploadErr) || tunnel.IsTunnelUnavailable(downloadErr) {
		return proxyErrorRelayTunnelUnavailable
	}
	return ""
}
