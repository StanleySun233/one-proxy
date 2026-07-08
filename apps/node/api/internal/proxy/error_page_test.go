package proxy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteProxyErrorRendersHintPage(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/app", nil)
	resp := httptest.NewRecorder()

	writeProxyError(resp, req, proxyErrorNextHopConnectFailed, http.StatusBadGateway)

	if resp.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", resp.Code)
	}
	if resp.Header().Get("X-One-Proxy-Error") != proxyErrorNextHopConnectFailed {
		t.Fatalf("error header = %q", resp.Header().Get("X-One-Proxy-Error"))
	}
	if !strings.HasPrefix(resp.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("content type = %q", resp.Header().Get("Content-Type"))
	}
	body := resp.Body.String()
	for _, expected := range []string{
		"Next Hop Connection Failed",
		"next_hop_connect_failed",
		"Check that the next hop node is online and healthy.",
		"GET http://example.com/app",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("body missing %q: %s", expected, body)
		}
	}
}

func TestWriteProxyErrorKeepsHeadBodyEmpty(t *testing.T) {
	req := httptest.NewRequest(http.MethodHead, "http://example.com/app", nil)
	resp := httptest.NewRecorder()

	writeProxyError(resp, req, proxyErrorForwardFailed, http.StatusBadGateway)

	if resp.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", resp.Code)
	}
	if resp.Body.Len() != 0 {
		t.Fatalf("body = %q", resp.Body.String())
	}
}

func TestProxyErrorCatalogCoversAllCodes(t *testing.T) {
	codes := []string{
		proxyErrorConnectFailed,
		proxyErrorForwardFailed,
		proxyErrorHijackFailed,
		proxyErrorHijackNotSupported,
		proxyErrorInvalidChainRoute,
		proxyErrorNextHopConnectFailed,
		proxyErrorNextHopUnreachable,
		proxyErrorProxyAuthRequired,
		proxyErrorRelayTunnelUnavailable,
		proxyErrorReverseAuthRequired,
		proxyErrorReverseConnectFailed,
		proxyErrorReverseForwardFailed,
		proxyErrorReverseUpgradeWriteFailed,
		proxyErrorRouteNotFound,
		proxyErrorStreamResponseFailed,
		proxyErrorUnsupportedRouteAction,
		proxyErrorUpgradeRejected,
		proxyErrorUpgradeResponseFailed,
		proxyErrorUpgradeWriteFailed,
	}
	for _, code := range codes {
		content, ok := proxyErrorCatalog[code]
		if !ok {
			t.Fatalf("missing catalog entry for %s", code)
		}
		if content.Title == "" || content.Summary == "" || len(content.Checks) == 0 {
			t.Fatalf("incomplete catalog entry for %s: %+v", code, content)
		}
	}
}
