package proxy

import (
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/node/api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/node/api/internal/policystore"
)

func (s *Server) forwardChain(w http.ResponseWriter, req *http.Request, snapshot policystore.Snapshot, rule domain.RouteRule, tracker *proxySessionTracker) {
	hop, ok := s.resolveChainHop(snapshot, rule.ChainID)
	if !ok {
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorInvalidChainRoute, proxyErrorInvalidChainRoute)
		writeProxyError(w, req, proxyErrorInvalidChainRoute, http.StatusBadGateway)
		return
	}
	if hop.isLast {
		if isWebSocketUpgrade(req) {
			s.upgradeDirect(w, req, tracker)
			return
		}
		if req.Method == http.MethodConnect {
			s.tunnelDirect(w, req, tracker)
			return
		}
		s.forwardDirect(w, req, tracker)
		return
	}
	if s.shouldUseStream(hop.node) {
		if isWebSocketUpgrade(req) {
			s.upgradeViaStream(w, req, hop, tracker)
			return
		}
		if req.Method == http.MethodConnect {
			s.tunnelViaStream(w, req, hop, tracker)
			return
		}
		s.forwardViaStream(w, req, hop, tracker)
		return
	}
	if hop.node.PublicHost == "" || hop.node.PublicPort <= 0 {
		tracker.finish(0, 0, domain.ProxySessionStatusError, proxyErrorNextHopUnreachable, proxyErrorNextHopUnreachable)
		writeProxyError(w, req, proxyErrorNextHopUnreachable, http.StatusBadGateway)
		return
	}
	if req.Method == http.MethodConnect {
		s.tunnelViaProxy(w, req, hop.node, tracker)
		return
	}
	if isWebSocketUpgrade(req) {
		s.upgradeViaProxy(w, req, hop.node, tracker)
		return
	}
	s.forwardViaProxy(w, req, hop.node, tracker)
}

func (s *Server) shouldUseStream(nextHop domain.Node) bool {
	privateNextHop := nextHop.PublicHost == "" || nextHop.PublicPort <= 0
	return s.shouldUseTunnel(nextHop) || (privateNextHop && (s.hasDirectPeer(nextHop.ID) || s.directStream != nil))
}

func (s *Server) shouldUseTunnel(nextHop domain.Node) bool {
	return s.tunnelRegistry != nil && s.tunnelRegistry.HasChild(nextHop.ID)
}

func (s *Server) hasDirectPeer(nodeID string) bool {
	available, ok := s.directStream.(directPeerAvailability)
	return ok && available.HasDirectPeer(nodeID)
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
