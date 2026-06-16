package proxy

import (
	"net/http"
	"net/url"
	"time"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/policystore"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/responsecache"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/tunnel"
)

type Server struct {
	store           *policystore.Store
	nodeIDGetter    func() string
	tunnelRegistry  *tunnel.Registry
	directStream    directPeerStreamOpener
	reverseTarget   *url.URL
	auth            AuthConfig
	authorizer      *TokenAuthorizer
	metricsReporter ProxySessionReporter
	responseCache   *responsecache.Cache
}

type AuthConfig struct {
	Validator TokenValidator
	CacheTTL  time.Duration
}

type chainHop struct {
	node          domain.Node
	remainingHops []string
	isLast        bool
}

func NewServer(store *policystore.Store, nodeIDGetter func() string, tunnelRegistry *tunnel.Registry) *Server {
	return &Server{store: store, nodeIDGetter: nodeIDGetter, tunnelRegistry: tunnelRegistry}
}

func (s *Server) SetDirectStreamOpener(opener directPeerStreamOpener) {
	s.directStream = opener
}

func (s *Server) SetResponseCache(cache *responsecache.Cache) {
	s.responseCache = cache
}

func NewServerWithReverseTarget(store *policystore.Store, nodeIDGetter func() string, tunnelRegistry *tunnel.Registry, reverseTargetURL string) (*Server, error) {
	return NewServerWithOptions(store, nodeIDGetter, tunnelRegistry, reverseTargetURL, AuthConfig{})
}

func NewServerWithOptions(store *policystore.Store, nodeIDGetter func() string, tunnelRegistry *tunnel.Registry, reverseTargetURL string, auth AuthConfig) (*Server, error) {
	return NewServerWithAuthorizer(store, nodeIDGetter, tunnelRegistry, reverseTargetURL, auth, nil)
}

func NewServerWithAuthorizer(store *policystore.Store, nodeIDGetter func() string, tunnelRegistry *tunnel.Registry, reverseTargetURL string, auth AuthConfig, authorizer *TokenAuthorizer) (*Server, error) {
	server := NewServer(store, nodeIDGetter, tunnelRegistry)
	server.auth = auth
	if authorizer == nil {
		authorizer = NewTokenAuthorizer(auth)
	}
	server.authorizer = authorizer
	if reverseTargetURL == "" {
		return server, nil
	}
	target, err := url.Parse(reverseTargetURL)
	if err != nil {
		return nil, err
	}
	if target.Scheme == "" || target.Host == "" {
		return nil, url.InvalidHostError(reverseTargetURL)
	}
	server.reverseTarget = target
	return server, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if s.reverseTarget != nil && !isForwardProxyRequest(req) {
		validation, ok := s.authorizeReverse(w, req)
		if !ok {
			return
		}
		if isWebSocketUpgrade(req) {
			tracker := s.newReverseProxySession(req, validation.TenantID)
			s.upgradeReverse(w, req, tracker)
			return
		}
		tracker := s.newReverseProxySession(req, validation.TenantID)
		s.forwardReverse(w, req, tracker)
		return
	}
	validation, ok := s.authorizeForwardRequest(w, req)
	if !ok {
		return
	}
	revision, snapshot := s.store.Current()
	match := Match(snapshot, req)
	if !match.Found {
		if !validation.AllowLocalProxy {
			tracker := s.newProxySession(req, domain.RouteRule{
				TenantID:   validation.TenantID,
				ActionType: domain.ActionTypeDeny,
				MatchType:  domain.MatchTypeDefault,
				MatchValue: targetHostForAudit(req),
			}, validation.TenantID, revision)
			tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorRouteNotFound, proxyErrorRouteNotFound)
			writeProxyError(w, req, proxyErrorRouteNotFound, http.StatusForbidden)
			return
		}
		match = RouteMatch{Rule: domain.RouteRule{
			TenantID:   validation.TenantID,
			ActionType: domain.ActionTypeDirect,
		}, Found: true}
	}
	tracker := s.newProxySession(req, match.Rule, routeTenantID(snapshot, match.Rule, validation.TenantID), revision)
	switch match.Rule.ActionType {
	case domain.ActionTypeDirect:
		if isWebSocketUpgrade(req) {
			s.upgradeDirect(w, req, tracker)
			return
		}
		if req.Method == http.MethodConnect {
			s.tunnelDirect(w, req, tracker)
			return
		}
		s.forwardDirect(w, req, tracker)
	case domain.ActionTypeChain:
		s.forwardChain(w, req, snapshot, match.Rule, tracker)
	default:
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorUnsupportedRouteAction, proxyErrorUnsupportedRouteAction)
		writeProxyError(w, req, proxyErrorUnsupportedRouteAction, http.StatusBadRequest)
	}
}

func routeTenantID(snapshot policystore.Snapshot, rule domain.RouteRule, fallbackTenantID string) string {
	if rule.TenantID != "" {
		return rule.TenantID
	}
	for _, chain := range snapshot.Chains {
		if chain.ID == rule.ChainID {
			return chain.TenantID
		}
	}
	return fallbackTenantID
}

func targetHostForAudit(req *http.Request) string {
	if req.URL != nil && req.URL.Hostname() != "" {
		return req.URL.Hostname()
	}
	return req.Host
}
