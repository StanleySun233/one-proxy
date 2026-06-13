package proxy

import (
	"net"
	"net/http"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/policystore"
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
		case domain.MatchTypeDomain:
			if strings.EqualFold(name, rule.MatchValue) {
				return RouteMatch{Rule: rule, Found: true}
			}
		case domain.MatchTypeDomainSuffix:
			if domainSuffixMatches(rule.MatchValue, name) {
				return RouteMatch{Rule: rule, Found: true}
			}
		case domain.MatchTypeIP:
			if name == rule.MatchValue {
				return RouteMatch{Rule: rule, Found: true}
			}
		case domain.MatchTypeIPCIDR:
			ip := net.ParseIP(name)
			_, network, err := net.ParseCIDR(rule.MatchValue)
			if err == nil && ip != nil && network.Contains(ip) {
				return RouteMatch{Rule: rule, Found: true}
			}
		case domain.MatchTypeProtocol:
			if strings.EqualFold(protocol, rule.MatchValue) {
				return RouteMatch{Rule: rule, Found: true}
			}
		case domain.MatchTypeDefault:
			return RouteMatch{Rule: rule, Found: true}
		}
	}
	return RouteMatch{}
}

func domainSuffixMatches(value string, host string) bool {
	suffix := strings.TrimPrefix(strings.TrimPrefix(strings.ToLower(value), "*."), ".")
	name := strings.ToLower(host)
	return suffix != "" && (name == suffix || strings.HasSuffix(name, "."+suffix))
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
