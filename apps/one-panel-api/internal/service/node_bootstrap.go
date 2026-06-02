package service

import "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"

func (c *ControlPlane) CreateBootstrapToken(tenantCtx domain.TenantAuthContext, input domain.CreateBootstrapTokenInput) (domain.BootstrapToken, error) {
	if input.TargetType == "" {
		return domain.BootstrapToken{}, invalidInput("invalid_bootstrap_payload")
	}
	if input.TargetID == "" {
		if err := validateNodeInput(input.NodeName, input.NodeMode, input.ScopeKey); err != nil {
			return domain.BootstrapToken{}, err
		}
		if !c.ScopeExists(input.ScopeKey) {
			return domain.BootstrapToken{}, invalidInput("scope_not_found")
		}
	}
	if input.NodeMode != "" && !c.isValidEnum("node_mode", input.NodeMode) {
		return domain.BootstrapToken{}, invalidInput("invalid_node_payload")
	}
	return c.store.CreateBootstrapTokenForTenant(tenantCtx, input)
}

func (c *ControlPlane) UnconsumedBootstrapTokens() []domain.BootstrapToken {
	return c.store.ListUnconsumedBootstrapTokens()
}

func (c *ControlPlane) DeleteBootstrapToken(tokenID string) error {
	if tokenID == "" {
		return invalidInput("missing_bootstrap_token_id")
	}
	return c.store.DeleteBootstrapToken(tokenID)
}
