package service

import (
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain/link"
	"strings"
)

func (c *ControlPlane) Scopes() []link.Scope {
	return c.store.ListScopes()
}

func (c *ControlPlane) CreateScope(input link.CreateScopeInput) (link.Scope, error) {
	input.ID = ""
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return link.Scope{}, invalidInput("invalid_scope_payload")
	}
	return c.store.CreateScope(input)
}

func (c *ControlPlane) UpdateScope(scopeID string, input link.UpdateScopeInput) (link.Scope, error) {
	if scopeID == "" || strings.TrimSpace(input.Name) == "" {
		return link.Scope{}, invalidInput("invalid_scope_payload")
	}
	return c.store.UpdateScope(scopeID, input)
}

func (c *ControlPlane) DeleteScope(scopeID string) error {
	if scopeID == "" {
		return invalidInput("missing_scope_id")
	}
	if c.scopeInUse(scopeID) {
		return invalidInput("scope_in_use")
	}
	return c.store.DeleteScope(scopeID)
}

func (c *ControlPlane) scopeExists(scopeID string) bool {
	for _, scope := range c.store.ListScopes() {
		if scope.ID == scopeID {
			return true
		}
	}
	return false
}

func (c *ControlPlane) scopeInUse(scopeID string) bool {
	for _, node := range c.store.ListNodes() {
		if node.ScopeKey == scopeID {
			return true
		}
	}
	for _, chain := range c.store.ListChains() {
		if chain.DestinationScope == scopeID {
			return true
		}
	}
	for _, rule := range c.store.ListRouteRules() {
		if rule.DestinationScope == scopeID {
			return true
		}
	}
	groups, err := c.store.ListGroups()
	if err != nil {
		return true
	}
	for _, group := range groups {
		scopes, err := c.store.GetGroupScopes(group.ID)
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
