package proxy

import (
	"html/template"
	"net/http"
	"time"
)

const (
	proxyErrorConnectFailed             = "connect_failed"
	proxyErrorForwardFailed             = "forward_failed"
	proxyErrorHijackFailed              = "hijack_failed"
	proxyErrorHijackNotSupported        = "hijack_not_supported"
	proxyErrorInvalidChainRoute         = "invalid_chain_route"
	proxyErrorNextHopConnectFailed      = "next_hop_connect_failed"
	proxyErrorNextHopUnreachable        = "next_hop_unreachable"
	proxyErrorProxyAuthRequired         = "proxy_auth_required"
	proxyErrorRelayTunnelUnavailable    = "relay_tunnel_unavailable"
	proxyErrorReverseAuthRequired       = "reverse_auth_required"
	proxyErrorReverseConnectFailed      = "reverse_connect_failed"
	proxyErrorReverseForwardFailed      = "reverse_forward_failed"
	proxyErrorReverseUpgradeWriteFailed = "reverse_upgrade_write_failed"
	proxyErrorRouteNotFound             = "route_not_found"
	proxyErrorStreamResponseFailed      = "stream_response_failed"
	proxyErrorUnsupportedRouteAction    = "unsupported_route_action"
	proxyErrorUpgradeRejected           = "upgrade_rejected"
	proxyErrorUpgradeResponseFailed     = "upgrade_response_failed"
	proxyErrorUpgradeWriteFailed        = "upgrade_write_failed"
)

type proxyErrorContent struct {
	Title   string
	Summary string
	Checks  []string
}

type proxyErrorPageData struct {
	StatusCode    int
	StatusText    string
	Code          string
	Title         string
	Summary       string
	Checks        []string
	RequestMethod string
	RequestTarget string
	GeneratedAt   string
}

var proxyErrorCatalog = map[string]proxyErrorContent{
	proxyErrorConnectFailed: {
		Title:   "Target Connection Failed",
		Summary: "OneProxy could not open a direct TCP connection to the requested target.",
		Checks: []string{
			"Confirm the target host and port are reachable from this node.",
			"Check DNS resolution, firewall rules, and service availability.",
			"Retry after the target service is healthy.",
		},
	},
	proxyErrorForwardFailed: {
		Title:   "Forwarding Failed",
		Summary: "The proxy could not complete the HTTP request after retrying the forward path.",
		Checks: []string{
			"Check whether the target service returned transient 502, 503, or 504 responses.",
			"Verify the target accepts the request method and payload.",
			"Review node diagnostics for DNS, TCP, or response body read errors.",
		},
	},
	proxyErrorHijackFailed: {
		Title:   "Connection Handoff Failed",
		Summary: "OneProxy could not take over the client connection for a tunnel or upgrade flow.",
		Checks: []string{
			"Retry the request in a fresh browser tab or client session.",
			"Check whether an intermediate server closed the connection early.",
			"Inspect node logs for the matching proxy session timestamp.",
		},
	},
	proxyErrorHijackNotSupported: {
		Title:   "Connection Handoff Unsupported",
		Summary: "The current HTTP runtime cannot hand off the connection required by this request.",
		Checks: []string{
			"Use the native OneProxy node listener for CONNECT and WebSocket traffic.",
			"Avoid wrapping this handler behind middleware that does not support HTTP hijacking.",
			"Route plain HTTP traffic through the standard forward path when possible.",
		},
	},
	proxyErrorInvalidChainRoute: {
		Title:   "Invalid Chain Route",
		Summary: "The matched policy points to a chain that this node cannot resolve.",
		Checks: []string{
			"Confirm the chain exists and contains this node.",
			"Verify every hop in the chain still exists in the active policy.",
			"Refresh policy from the control plane and retry.",
		},
	},
	proxyErrorNextHopConnectFailed: {
		Title:   "Next Hop Connection Failed",
		Summary: "OneProxy found the next hop but could not establish the proxy, tunnel, or direct stream connection.",
		Checks: []string{
			"Check that the next hop node is online and healthy.",
			"Verify public host and port, tunnel child status, or direct QUIC peer status.",
			"Confirm the next hop can reach the final target host and port.",
		},
	},
	proxyErrorNextHopUnreachable: {
		Title:   "Next Hop Unreachable",
		Summary: "The next hop has no usable public endpoint, tunnel child, or direct peer path.",
		Checks: []string{
			"Add or repair the next hop public host and port.",
			"Bring up the child tunnel or direct peer connection for this node pair.",
			"Check that the active chain topology matches the deployed nodes.",
		},
	},
	proxyErrorProxyAuthRequired: {
		Title:   "Proxy Authentication Required",
		Summary: "The forward proxy request did not include a valid OneProxy token.",
		Checks: []string{
			"Refresh the browser extension or client proxy credentials.",
			"Confirm the token has not expired or been revoked.",
			"Verify the token grants access to this node and tenant.",
		},
	},
	proxyErrorRelayTunnelUnavailable: {
		Title:   "Relay Tunnel Unavailable",
		Summary: "The selected relay node is offline, reconnecting, or its parent tunnel closed during the request.",
		Checks: []string{
			"Wait a few seconds and retry after the relay reconnects.",
			"Check the relay node transport status and recent tunnel disconnect logs.",
			"Use a smaller transfer only to confirm whether the issue is sustained load or general connectivity.",
		},
	},
	proxyErrorReverseAuthRequired: {
		Title:   "Reverse Proxy Authentication Required",
		Summary: "The reverse proxy request did not include a valid OneProxy token.",
		Checks: []string{
			"Open the reverse proxy URL with a valid one_proxy_auth token.",
			"Confirm the token has not expired or been revoked.",
			"Retry after the authentication cookie has been set for this site.",
		},
	},
	proxyErrorReverseConnectFailed: {
		Title:   "Reverse Target Connection Failed",
		Summary: "OneProxy could not open the reverse WebSocket target connection.",
		Checks: []string{
			"Confirm the configured reverse target host and port are reachable.",
			"Check TLS settings for https or wss reverse targets.",
			"Verify the target WebSocket service is running.",
		},
	},
	proxyErrorReverseForwardFailed: {
		Title:   "Reverse Forwarding Failed",
		Summary: "OneProxy could not complete the HTTP request to the configured reverse target.",
		Checks: []string{
			"Confirm the reverse target URL is correct and reachable from this node.",
			"Check whether the target returned transient 502, 503, or 504 responses.",
			"Inspect the reverse target logs for the same request time.",
		},
	},
	proxyErrorReverseUpgradeWriteFailed: {
		Title:   "Reverse WebSocket Write Failed",
		Summary: "OneProxy connected to the reverse target but could not write the upgrade request.",
		Checks: []string{
			"Check whether the reverse target closed the socket during handshake.",
			"Verify headers and path expected by the target WebSocket endpoint.",
			"Retry after confirming the reverse target is stable.",
		},
	},
	proxyErrorRouteNotFound: {
		Title:   "No Proxy Route Matched",
		Summary: "The active policy did not allow this request and local proxy fallback is not granted.",
		Checks: []string{
			"Add or enable a route rule for this host, IP, protocol, or default route.",
			"Confirm the requesting token has local proxy fallback permission if no rule should match.",
			"Refresh policy and verify the node is using the expected revision.",
		},
	},
	proxyErrorStreamResponseFailed: {
		Title:   "Stream Forwarding Failed",
		Summary: "OneProxy could not prepare the request body for a chained stream forward.",
		Checks: []string{
			"Retry the request if the browser or client aborted the upload.",
			"Check whether the request body source closed before OneProxy could read it.",
			"Use a smaller payload to separate upload failure from next hop failure.",
		},
	},
	proxyErrorUnsupportedRouteAction: {
		Title:   "Unsupported Route Action",
		Summary: "The matched policy action is not supported by this proxy node.",
		Checks: []string{
			"Update the route action to direct or chain.",
			"Check the control plane policy compiler output.",
			"Refresh the node policy after correcting the route.",
		},
	},
	proxyErrorUpgradeRejected: {
		Title:   "WebSocket Upgrade Rejected",
		Summary: "The target service responded without switching protocols.",
		Checks: []string{
			"Verify the target path is a WebSocket endpoint.",
			"Check Origin, authentication, and upgrade headers expected by the target.",
			"Inspect the upstream response status returned to the browser.",
		},
	},
	proxyErrorUpgradeResponseFailed: {
		Title:   "WebSocket Upgrade Response Failed",
		Summary: "OneProxy sent the upgrade request but could not read a valid response from the target.",
		Checks: []string{
			"Check whether the target closed the connection during handshake.",
			"Verify the target speaks HTTP WebSocket upgrade on this endpoint.",
			"Inspect target logs for handshake parsing or authentication failures.",
		},
	},
	proxyErrorUpgradeWriteFailed: {
		Title:   "WebSocket Upgrade Write Failed",
		Summary: "OneProxy connected to the target or next hop but could not write the upgrade request.",
		Checks: []string{
			"Confirm the target or next hop did not close the socket during handshake.",
			"Check network stability between this node and the next endpoint.",
			"Retry after verifying the WebSocket endpoint is accepting connections.",
		},
	},
}

var fallbackProxyErrorContent = proxyErrorContent{
	Title:   "Proxy Error",
	Summary: "OneProxy could not complete this request.",
	Checks: []string{
		"Review the error code and HTTP status shown on this page.",
		"Open node diagnostics and recent proxy sessions for the same time window.",
		"Retry after the active policy and node connectivity have been verified.",
	},
}

var proxyErrorPageTemplate = template.Must(template.New("proxy-error").Parse(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>{{.Title}} - OneProxy</title>
    <style>
      :root {
        color-scheme: light;
        --bg: #f6f7f9;
        --panel: #ffffff;
        --ink: #20242a;
        --muted: #687281;
        --line: #d9dee7;
        --danger: #b0473d;
        --focus: #2563eb;
        --soft: #eef2f6;
      }
      * {
        box-sizing: border-box;
      }
      body {
        min-height: 100vh;
        margin: 0;
        background: var(--bg);
        color: var(--ink);
        font-family: "Aptos", "Segoe UI", sans-serif;
      }
      .page {
        width: min(960px, calc(100% - 32px));
        margin: 0 auto;
        padding: 48px 0;
      }
      .hero,
      .panel {
        border: 1px solid var(--line);
        border-radius: 8px;
        background: var(--panel);
        box-shadow: 0 18px 48px rgba(32, 36, 42, 0.1);
      }
      .hero {
        padding: 30px;
      }
      .brand {
        display: flex;
        align-items: center;
        gap: 12px;
        color: var(--muted);
        font-size: 13px;
        font-weight: 700;
        letter-spacing: 0;
      }
      .mark {
        width: 38px;
        height: 38px;
        display: grid;
        place-items: center;
        border: 1px solid var(--line);
        border-radius: 8px;
        color: var(--focus);
      }
      .eyebrow {
        margin: 28px 0 8px;
        color: var(--danger);
        font-size: 12px;
        font-weight: 800;
        text-transform: uppercase;
      }
      h1 {
        margin: 0;
        font-size: clamp(32px, 5vw, 56px);
        line-height: 1.04;
        letter-spacing: 0;
      }
      .summary {
        max-width: 720px;
        margin: 16px 0 0;
        color: var(--muted);
        font-size: 18px;
        line-height: 1.55;
      }
      .meta {
        display: grid;
        grid-template-columns: repeat(3, minmax(0, 1fr));
        gap: 12px;
        margin-top: 26px;
      }
      .meta div {
        min-width: 0;
        padding: 14px;
        border: 1px solid var(--line);
        border-radius: 8px;
        background: var(--soft);
      }
      .meta span {
        display: block;
        color: var(--muted);
        font-size: 12px;
        text-transform: uppercase;
      }
      .meta strong {
        display: block;
        margin-top: 6px;
        overflow-wrap: anywhere;
      }
      .panel {
        margin-top: 16px;
        padding: 24px 30px;
      }
      h2 {
        margin: 0 0 14px;
        font-size: 18px;
      }
      ol {
        display: grid;
        gap: 10px;
        margin: 0;
        padding-left: 22px;
      }
      li {
        line-height: 1.5;
      }
      .stamp {
        margin: 14px 0 0;
        color: var(--muted);
        font-size: 12px;
      }
      @media (max-width: 720px) {
        .page {
          width: min(100% - 24px, 960px);
          padding: 24px 0;
        }
        .hero,
        .panel {
          padding: 20px;
        }
        .meta {
          grid-template-columns: 1fr;
        }
      }
    </style>
  </head>
  <body>
    <main class="page">
      <section class="hero">
        <div class="brand"><div class="mark">OP</div><span>OneProxy</span></div>
        <p class="eyebrow">Proxy request failed</p>
        <h1>{{.Title}}</h1>
        <p class="summary">{{.Summary}}</p>
        <div class="meta">
          <div><span>Status</span><strong>{{.StatusCode}} {{.StatusText}}</strong></div>
          <div><span>Error code</span><strong>{{.Code}}</strong></div>
          <div><span>Request</span><strong>{{.RequestMethod}} {{.RequestTarget}}</strong></div>
        </div>
      </section>
      <section class="panel">
        <h2>Check This</h2>
        <ol>{{range .Checks}}<li>{{.}}</li>{{end}}</ol>
        <p class="stamp">Generated {{.GeneratedAt}}</p>
      </section>
    </main>
  </body>
</html>`))

func writeProxyError(w http.ResponseWriter, req *http.Request, code string, status int) {
	content, ok := proxyErrorCatalog[code]
	if !ok {
		content = fallbackProxyErrorContent
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-One-Proxy-Error", code)
	w.WriteHeader(status)
	if req.Method == http.MethodHead {
		return
	}
	_ = proxyErrorPageTemplate.Execute(w, proxyErrorPageData{
		StatusCode:    status,
		StatusText:    http.StatusText(status),
		Code:          code,
		Title:         content.Title,
		Summary:       content.Summary,
		Checks:        content.Checks,
		RequestMethod: req.Method,
		RequestTarget: proxyErrorRequestTarget(req),
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
	})
}

func proxyErrorRequestTarget(req *http.Request) string {
	if req.Method == http.MethodConnect {
		return req.Host
	}
	if req.URL != nil && req.URL.IsAbs() {
		return req.URL.String()
	}
	if req.URL != nil && req.URL.RequestURI() != "" {
		if req.Host != "" {
			return req.Host + req.URL.RequestURI()
		}
		return req.URL.RequestURI()
	}
	return req.Host
}
