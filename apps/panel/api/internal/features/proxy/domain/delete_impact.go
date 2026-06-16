package proxy

type DeleteImpactItem struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Detail string `json:"detail,omitempty"`
}

type ChainDeleteImpact struct {
	ChainID string                  `json:"chainId"`
	Delete  ChainDeleteImpactDelete `json:"delete"`
}

type ChainDeleteImpactDelete struct {
	Chain             []DeleteImpactItem `json:"chain"`
	ChainHops         []DeleteImpactItem `json:"chainHops"`
	RouteRuleGroups   []DeleteImpactItem `json:"routeRuleGroups"`
	RouteRules        []DeleteImpactItem `json:"routeRules"`
	AccessPaths       []DeleteImpactItem `json:"accessPaths"`
	OnboardingTasks   []DeleteImpactItem `json:"onboardingTasks"`
	ChainProbeResults []DeleteImpactItem `json:"chainProbeResults"`
	TenantBindings    []DeleteImpactItem `json:"tenantBindings"`
}

type NodeAccessPathDeleteImpact struct {
	PathID string                         `json:"pathId"`
	Delete NodeAccessPathDeleteImpactBody `json:"delete"`
}

type NodeAccessPathDeleteImpactBody struct {
	AccessPath      []DeleteImpactItem `json:"accessPath"`
	OnboardingTasks []DeleteImpactItem `json:"onboardingTasks"`
	TenantBindings  []DeleteImpactItem `json:"tenantBindings"`
}

type RouteRuleGroupDeleteImpact struct {
	GroupID string                           `json:"groupId"`
	Delete  RouteRuleGroupDeleteImpactDelete `json:"delete"`
}

type RouteRuleGroupDeleteImpactDelete struct {
	Group          []DeleteImpactItem `json:"group"`
	RouteRules     []DeleteImpactItem `json:"routeRules"`
	TenantBindings []DeleteImpactItem `json:"tenantBindings"`
}
