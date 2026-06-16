package proxy

type RouteRuleGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	CreateID    string `json:"createId"`
	OwnerID     string `json:"ownerId"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	Permission  string `json:"permission,omitempty"`
	RuleCount   int    `json:"ruleCount"`
}

type CreateRouteRuleGroupInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateRouteRuleGroupInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

type RouteRule struct {
	ID               string `json:"id"`
	GroupID          string `json:"groupId"`
	CreateID         string `json:"createId"`
	OwnerID          string `json:"ownerId"`
	Priority         int    `json:"priority"`
	MatchType        string `json:"matchType"`
	MatchValue       string `json:"matchValue"`
	ActionType       string `json:"actionType"`
	ChainID          string `json:"chainId,omitempty"`
	DestinationScope string `json:"destinationScope,omitempty"`
	Enabled          bool   `json:"enabled"`
	Permission       string `json:"permission,omitempty"`
}

type RouteRuleWithDetails struct {
	ID               string            `json:"id"`
	GroupID          string            `json:"groupId"`
	CreateID         string            `json:"createId"`
	OwnerID          string            `json:"ownerId"`
	Priority         int               `json:"priority"`
	MatchType        string            `json:"matchType"`
	MatchValue       string            `json:"matchValue"`
	ActionType       string            `json:"actionType"`
	ChainID          string            `json:"chainId,omitempty"`
	Chain            *ChainWithDetails `json:"chain,omitempty"`
	DestinationScope string            `json:"destinationScope,omitempty"`
	Enabled          bool              `json:"enabled"`
	Permission       string            `json:"permission,omitempty"`
}

type CreateRouteRuleInput struct {
	GroupID          string `json:"groupId"`
	Priority         int    `json:"priority"`
	MatchType        string `json:"matchType"`
	MatchValue       string `json:"matchValue"`
	ActionType       string `json:"actionType"`
	ChainID          string `json:"chainId"`
	DestinationScope string `json:"destinationScope"`
}

type UpdateRouteRuleInput struct {
	GroupID          string `json:"groupId"`
	Priority         int    `json:"priority"`
	MatchType        string `json:"matchType"`
	MatchValue       string `json:"matchValue"`
	ActionType       string `json:"actionType"`
	ChainID          string `json:"chainId"`
	DestinationScope string `json:"destinationScope"`
	Enabled          bool   `json:"enabled"`
}

type MatchType struct {
	Type            string  `json:"type"`
	Label           string  `json:"label"`
	Description     string  `json:"description"`
	Placeholder     string  `json:"placeholder"`
	ValidationRegex *string `json:"validationRegex"`
}

type ValidateRouteRuleInput struct {
	RuleID           string `json:"ruleId"`
	GroupID          string `json:"groupId"`
	Priority         int    `json:"priority"`
	MatchType        string `json:"matchType"`
	MatchValue       string `json:"matchValue"`
	ActionType       string `json:"actionType"`
	ChainID          string `json:"chainId"`
	DestinationScope string `json:"destinationScope"`
}

type MatchValueValidation struct {
	Valid   bool   `json:"valid"`
	Format  string `json:"format"`
	Message string `json:"message"`
}

type ChainValidation struct {
	Valid        bool     `json:"valid"`
	ChainEnabled bool     `json:"chainEnabled"`
	ChainHops    []string `json:"chainHops"`
}

type ScopeValidation struct {
	Valid                bool   `json:"valid"`
	ScopeExists          bool   `json:"scopeExists"`
	ScopeOwnerNodeID     string `json:"scopeOwnerNodeId"`
	MatchesChainFinalHop bool   `json:"matchesChainFinalHop"`
}

type RouteRuleValidationResult struct {
	Valid                bool                 `json:"valid"`
	Errors               []string             `json:"errors"`
	Warnings             []string             `json:"warnings"`
	MatchValueValidation MatchValueValidation `json:"matchValueValidation"`
	ChainValidation      ChainValidation      `json:"chainValidation"`
	ScopeValidation      ScopeValidation      `json:"scopeValidation"`
}

type RouteRuleSuggestionResult struct {
	MatchType   string   `json:"matchType"`
	Suggestions []string `json:"suggestions"`
}
