package linkservice

import (
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/features/link/domain"
	"strings"
)

func (s *Service) Scopes() []link.Scope {
	return s.store.ListScopes()
}

func (s *Service) CreateScope(input link.CreateScopeInput) (link.Scope, error) {
	input.ID = ""
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return link.Scope{}, invalidInput("invalid_scope_payload")
	}
	return s.store.CreateScope(input)
}

func (s *Service) UpdateScope(scopeID string, input link.UpdateScopeInput) (link.Scope, error) {
	if scopeID == "" || strings.TrimSpace(input.Name) == "" {
		return link.Scope{}, invalidInput("invalid_scope_payload")
	}
	return s.store.UpdateScope(scopeID, input)
}

func (s *Service) DeleteScope(scopeID string) error {
	if scopeID == "" {
		return invalidInput("missing_scope_id")
	}
	if s.scopeInUse(scopeID) {
		return invalidInput("scope_in_use")
	}
	return s.store.DeleteScope(scopeID)
}

func (s *Service) scopeExists(scopeID string) bool {
	return s.ScopeExists(scopeID)
}

func (s *Service) ScopeExists(scopeID string) bool {
	for _, scope := range s.store.ListScopes() {
		if scope.ID == scopeID {
			return true
		}
	}
	return false
}

func (s *Service) scopeInUse(scopeID string) bool {
	for _, node := range s.store.ListNodes() {
		if node.ScopeKey == scopeID {
			return true
		}
	}
	for _, chain := range s.store.ListChains() {
		if chain.DestinationScope == scopeID {
			return true
		}
	}
	for _, rule := range s.store.ListRouteRules() {
		if rule.DestinationScope == scopeID {
			return true
		}
	}
	groups, err := s.store.ListGroups()
	if err != nil {
		return true
	}
	for _, group := range groups {
		scopes, err := s.store.GetGroupScopes(group.ID)
		if err != nil {
			return true
		}
		for _, item := range scopes {
			if item == scopeID {
				return true
			}
		}
	}
	return false
}
