package proxy

import (
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/policystore"
	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/tunnel"
)

type Server struct {
	store          *policystore.Store
	nodeIDGetter   func() string
	tunnelRegistry *tunnel.Registry
}

type chainHop struct {
	node          domain.Node
	remainingHops []string
	isLast        bool
}

func NewServer(store *policystore.Store, nodeIDGetter func() string, tunnelRegistry *tunnel.Registry) *Server {
	return &Server{store: store, nodeIDGetter: nodeIDGetter, tunnelRegistry: tunnelRegistry}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
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
