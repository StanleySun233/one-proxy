package proxy

import (
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-proxy-node/internal/policystore"
)

func (s *Server) forwardChain(w http.ResponseWriter, req *http.Request, snapshot policystore.Snapshot, rule domain.RouteRule) {
	hop, ok := s.resolveChainHop(snapshot, rule.ChainID)
	if !ok {
		http.Error(w, "invalid_chain_route", http.StatusBadGateway)
		return
	}
	if hop.isLast {
		if isWebSocketUpgrade(req) {
			s.upgradeDirect(w, req)
			return
		}
		if req.Method == http.MethodConnect {
			s.tunnelDirect(w, req)
			return
		}
		s.forwardDirect(w, req)
		return
	}
	if s.shouldUseTunnel(hop.node) {
		if isWebSocketUpgrade(req) {
			s.upgradeViaStream(w, req, hop)
			return
		}
		if req.Method == http.MethodConnect {
			s.tunnelViaStream(w, req, hop)
			return
		}
		s.forwardViaStream(w, req, hop)
		return
	}
	if hop.node.PublicHost == "" || hop.node.PublicPort <= 0 {
		http.Error(w, "next_hop_unreachable", http.StatusBadGateway)
		return
	}
	if req.Method == http.MethodConnect {
		s.tunnelViaProxy(w, req, hop.node)
		return
	}
	if isWebSocketUpgrade(req) {
		s.upgradeViaProxy(w, req, hop.node)
		return
	}
	s.forwardViaProxy(w, req, hop.node)
}

func (s *Server) shouldUseTunnel(nextHop domain.Node) bool {
	return s.tunnelRegistry != nil && s.tunnelRegistry.HasChild(nextHop.ID)
}

func (s *Server) resolveChainHop(snapshot policystore.Snapshot, chainID string) (chainHop, bool) {
	var chain domain.Chain
	found := false
	for _, item := range snapshot.Chains {
		if item.ID == chainID {
			chain = item
			found = true
			break
		}
	}
	if !found || len(chain.Hops) == 0 {
		return chainHop{}, false
	}
	index := -1
	nodeID := s.nodeIDGetter()
	for i, hop := range chain.Hops {
		if hop == nodeID {
			index = i
			break
		}
	}
	if index == -1 {
		return chainHop{}, false
	}
	if index == len(chain.Hops)-1 {
		return chainHop{isLast: true}, true
	}
	nextHopID := chain.Hops[index+1]
	for _, node := range snapshot.Nodes {
		if node.ID == nextHopID {
			return chainHop{
				node:          node,
				remainingHops: append([]string(nil), chain.Hops[index+2:]...),
			}, true
		}
	}
	return chainHop{}, false
}
