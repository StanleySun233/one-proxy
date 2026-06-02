package service

import (
	"context"
	"net/http"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/auth"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (c *ControlPlane) Accounts() []domain.Account {
	return c.store.ListAccounts()
}

func (c *ControlPlane) CreateAccount(input domain.CreateAccountInput) (domain.Account, error) {
	if input.Account == "" || input.Password == "" || input.Role == "" {
		return domain.Account{}, invalidInput("invalid_account_payload")
	}
	return c.store.CreateAccount(input)
}

func (c *ControlPlane) UpdateAccount(accountID string, input domain.UpdateAccountInput) (domain.Account, error) {
	if accountID == "" {
		return domain.Account{}, invalidInput("missing_account_id")
	}
	return c.store.UpdateAccount(accountID, input)
}

func (c *ControlPlane) DeleteAccount(accountID string) error {
	if accountID == "" {
		return invalidInput("missing_account_id")
	}
	return c.store.DeleteAccount(accountID)
}

func (c *ControlPlane) Login(account string, password string) (domain.LoginResult, bool) {
	result, ok := c.store.Authenticate(account, password)
	if !ok {
		return domain.LoginResult{}, false
	}
	return c.attachProxyToken(result)
}

func (c *ControlPlane) AuthenticateAccessToken(accessToken string) (domain.Account, bool) {
	return c.store.AuthenticateAccessToken(accessToken)
}

func (c *ControlPlane) RefreshSession(refreshToken string) (domain.LoginResult, bool) {
	result, ok := c.store.RefreshSession(refreshToken)
	if !ok {
		return domain.LoginResult{}, false
	}
	return c.attachProxyToken(result)
}

func (c *ControlPlane) Logout(accessToken string) bool {
	return c.store.Logout(accessToken)
}

func (c *ControlPlane) ResolveTenantAuthContext(account domain.Account, tenantID string, allowSuperAdminBypass bool) (domain.TenantAuthContext, error) {
	if account.Role == domain.AccountRoleSuperAdmin && allowSuperAdminBypass {
		ctx := domain.TenantAuthContext{
			Account:    account,
			SuperAdmin: true,
		}
		if tenantID == "" {
			return ctx, nil
		}
		tenant, ok := c.store.GetTenant(tenantID)
		if !ok {
			return domain.TenantAuthContext{}, newError(http.StatusBadRequest, "tenant_invalid")
		}
		ctx.ActiveTenant = domain.TenantMembership{
			TenantID:   tenant.ID,
			TenantName: tenant.Name,
			Role:       domain.TenantRoleAdmin,
			JoinedAt:   tenant.CreatedAt,
		}
		return ctx, nil
	}
	if tenantID == "" {
		return domain.TenantAuthContext{}, newError(http.StatusBadRequest, "tenant_required")
	}
	if _, ok := c.store.GetTenant(tenantID); !ok {
		return domain.TenantAuthContext{}, newError(http.StatusBadRequest, "tenant_invalid")
	}
	membership, ok := c.store.GetTenantMembership(account.ID, tenantID)
	if !ok {
		return domain.TenantAuthContext{}, newError(http.StatusForbidden, "tenant_forbidden")
	}
	return domain.TenantAuthContext{
		Account:      account,
		ActiveTenant: membership,
	}, nil
}

func (c *ControlPlane) attachProxyToken(result domain.LoginResult) (domain.LoginResult, bool) {
	token, err := auth.RandomToken()
	if err != nil {
		return domain.LoginResult{}, false
	}
	expiresAt := time.Now().UTC().Add(c.sessionTTL).Format(time.RFC3339)
	record := domain.ProxyTokenRecord{
		Account:           result.Account,
		ExpiresAt:         expiresAt,
		TenantMemberships: result.TenantMemberships,
		ActiveTenantID:    result.ActiveTenantID,
	}
	if err := c.proxyTokens.Put(context.Background(), auth.TokenHash(token), record, c.sessionTTL); err != nil {
		return domain.LoginResult{}, false
	}
	result.ProxyToken = token
	result.ProxyTokenExpiresAt = expiresAt
	return result, true
}

func (c *ControlPlane) ValidateProxyTokenHash(tokenHash string) domain.ProxyTokenValidation {
	if tokenHash == "" {
		return domain.ProxyTokenValidation{}
	}
	record, ok := c.proxyTokens.Get(context.Background(), tokenHash)
	if !ok {
		return domain.ProxyTokenValidation{}
	}
	expiresAt, err := time.Parse(time.RFC3339, record.ExpiresAt)
	if err != nil || time.Now().UTC().After(expiresAt) {
		return domain.ProxyTokenValidation{}
	}
	cacheTTL := c.proxyTokenCacheTTL
	remaining := time.Until(expiresAt)
	if remaining < cacheTTL {
		cacheTTL = remaining
	}
	if cacheTTL < time.Second {
		return domain.ProxyTokenValidation{}
	}
	return domain.ProxyTokenValidation{
		Valid:             true,
		Account:           record.Account,
		ExpiresAt:         record.ExpiresAt,
		CacheTTLSeconds:   int(cacheTTL.Seconds()),
		TenantMemberships: record.TenantMemberships,
		ActiveTenantID:    record.ActiveTenantID,
	}
}
