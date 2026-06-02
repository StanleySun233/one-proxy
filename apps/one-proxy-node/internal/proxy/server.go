package proxy

import (
	"net/http"
	"net/url"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/policystore"
	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/tunnel"
)

type Server struct {
	store          *policystore.Store
	nodeIDGetter   func() string
	tunnelRegistry *tunnel.Registry
	reverseTarget  *url.URL
	auth           AuthConfig
	authCache      *tokenCache
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

func NewServerWithReverseTarget(store *policystore.Store, nodeIDGetter func() string, tunnelRegistry *tunnel.Registry, reverseTargetURL string) (*Server, error) {
	return NewServerWithOptions(store, nodeIDGetter, tunnelRegistry, reverseTargetURL, AuthConfig{})
}

func NewServerWithOptions(store *policystore.Store, nodeIDGetter func() string, tunnelRegistry *tunnel.Registry, reverseTargetURL string, auth AuthConfig) (*Server, error) {
	server := NewServer(store, nodeIDGetter, tunnelRegistry)
	server.auth = auth
	server.authCache = &tokenCache{items: map[string]cachedTokenValidation{}}
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
		if !s.authorizeReverse(w, req) {
			return
		}
		if isWebSocketUpgrade(req) {
			s.upgradeReverse(w, req)
			return
		}
		s.forwardReverse(w, req)
		return
	}
	if !s.authorizeForward(w, req) {
		return
	}
	_, snapshot := s.store.Current()
	match := Match(snapshot, req)
	if !match.Found {
		http.Error(w, "route_not_found", http.StatusForbidden)
		return
	}
	switch match.Rule.ActionType {
	case domain.ActionTypeDirect:
		if isWebSocketUpgrade(req) {
			s.upgradeDirect(w, req)
			return
		}
		if req.Method == http.MethodConnect {
			s.tunnelDirect(w, req)
			return
		}
		s.forwardDirect(w, req)
	case domain.ActionTypeChain:
		s.forwardChain(w, req, snapshot, match.Rule)
	default:
		http.Error(w, "unsupported_route_action", http.StatusBadRequest)
	}
}
