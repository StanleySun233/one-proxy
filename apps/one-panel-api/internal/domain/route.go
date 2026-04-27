package domain

type RouteRule struct {
	ID               string `json:"id"`
	Priority         int    `json:"priority"`
	MatchType        string `json:"matchType"`
	MatchValue       string `json:"matchValue"`
	ActionType       string `json:"actionType"`
	ChainID          string `json:"chainId,omitempty"`
	DestinationScope string `json:"destinationScope,omitempty"`
	Enabled          bool   `json:"enabled"`
}

type RouteRuleWithDetails struct {
	ID               string            `json:"id"`
	Priority         int               `json:"priority"`
	MatchType        string            `json:"matchType"`
	MatchValue       string            `json:"matchValue"`
	ActionType       string            `json:"actionType"`
	ChainID          string            `json:"chainId,omitempty"`
	Chain            *ChainWithDetails `json:"chain,omitempty"`
	DestinationScope string            `json:"destinationScope,omitempty"`
	Enabled          bool              `json:"enabled"`
}

type CreateRouteRuleInput struct {
	Priority         int    `json:"priority"`
	MatchType        string `json:"matchType"`
	MatchValue       string `json:"matchValue"`
	ActionType       string `json:"actionType"`
	ChainID          string `json:"chainId"`
	DestinationScope string `json:"destinationScope"`
}

type UpdateRouteRuleInput struct {
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
	Valid               bool   `json:"valid"`
	ScopeExists         bool   `json:"scopeExists"`
	ScopeOwnerNodeID    string `json:"scopeOwnerNodeId"`
	MatchesChainFinalHop bool  `json:"matchesChainFinalHop"`
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
