package proxy

import (
	"net"
	"net/http"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/policystore"
)

type RouteMatch struct {
	Rule  domain.RouteRule
	Found bool
}

func Match(snapshot policystore.Snapshot, req *http.Request) RouteMatch {
	name := req.Host
	if strings.Contains(req.Host, ":") {
		if parsedHost, _, err := net.SplitHostPort(req.Host); err == nil {
			name = parsedHost
		}
	}
	protocol := requestProtocol(req)
	for _, rule := range snapshot.RouteRules {
		switch rule.MatchType {
		case "domain":
			if strings.EqualFold(name, rule.MatchValue) {
				return RouteMatch{Rule: rule, Found: true}
			}
		case "domain_suffix":
			if strings.HasSuffix(strings.ToLower(name), strings.ToLower(rule.MatchValue)) {
				return RouteMatch{Rule: rule, Found: true}
			}
		case "ip":
			if name == rule.MatchValue {
				return RouteMatch{Rule: rule, Found: true}
			}
		case "cidr":
			ip := net.ParseIP(name)
			_, network, err := net.ParseCIDR(rule.MatchValue)
			if err == nil && ip != nil && network.Contains(ip) {
				return RouteMatch{Rule: rule, Found: true}
			}
		case "protocol":
			if strings.EqualFold(protocol, rule.MatchValue) {
				return RouteMatch{Rule: rule, Found: true}
			}
		}
	}
	return RouteMatch{}
}

func requestProtocol(req *http.Request) string {
	if req.Method == http.MethodConnect {
		return "https"
	}
	if strings.EqualFold(req.Header.Get("Upgrade"), "websocket") {
		return "ws"
	}
	if req.URL.Scheme != "" {
		return strings.ToLower(req.URL.Scheme)
	}
	return "http"
}
