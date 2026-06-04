package service

import (
	"database/sql"
	"net/http"
	"sort"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

type CreateTenantInput struct {
	Name                  string `json:"name"`
	InitialAdminAccountID string `json:"initialAdminAccountId"`
}

type UpdateTenantInput struct {
	Name string `json:"name"`
}

type UpsertTenantMembershipInput struct {
	Role domain.TenantRole `json:"role"`
}

type TenantCreatedResult struct {
	Tenant     domain.Tenant           `json:"tenant"`
	Membership TenantMembershipAccount `json:"membership"`
}

type TenantMembershipAccount struct {
	AccountID  string            `json:"accountId"`
	Account    string            `json:"account"`
	TenantID   string            `json:"tenantId"`
	TenantName string            `json:"tenantName"`
	Role       domain.TenantRole `json:"role"`
	JoinedAt   string            `json:"joinedAt"`
}

func (c *ControlPlane) Tenants(account domain.Account) []domain.Tenant {
	return c.store.ListTenants(account)
}

func (c *ControlPlane) GrantTenants(account domain.Account, tenantCtx domain.TenantAuthContext) ([]domain.Tenant, error) {
	if account.Role != domain.AccountRoleSuperAdmin && tenantCtx.ActiveTenant.Role != domain.TenantRoleAdmin {
		return nil, newError(http.StatusForbidden, "tenant_role_forbidden")
	}
	return c.store.ListAllTenants(), nil
}

func (c *ControlPlane) CreateTenant(actor domain.Account, input CreateTenantInput) (TenantCreatedResult, error) {
	if actor.Role != domain.AccountRoleSuperAdmin {
		return TenantCreatedResult{}, newError(http.StatusForbidden, "tenant_role_forbidden")
	}
	if input.Name == "" || input.InitialAdminAccountID == "" {
		return TenantCreatedResult{}, invalidInput("invalid_tenant_payload")
	}
	if account, ok := c.accountByID(input.InitialAdminAccountID); !ok || account.Status != domain.AccountStatusActive {
		return TenantCreatedResult{}, invalidInput("invalid_initial_admin_account")
	}
	item, err := c.store.CreateTenant(input.Name, input.InitialAdminAccountID, actor.ID)
	if err != nil {
		return TenantCreatedResult{}, err
	}
	membership, ok := c.store.GetTenantMembership(input.InitialAdminAccountID, item.ID)
	if !ok {
		return TenantCreatedResult{}, sql.ErrNoRows
	}
	return TenantCreatedResult{
		Tenant:     item,
		Membership: c.membershipAccount(input.InitialAdminAccountID, membership),
	}, nil
}

func (c *ControlPlane) Tenant(account domain.Account, tenantCtx domain.TenantAuthContext, tenantID string) (domain.Tenant, error) {
	if err := c.requirePathTenant(account, tenantCtx, tenantID, false); err != nil {
		return domain.Tenant{}, err
	}
	item, ok := c.store.GetTenant(tenantID)
	if !ok {
		return domain.Tenant{}, sql.ErrNoRows
	}
	return item, nil
}

func (c *ControlPlane) UpdateTenant(account domain.Account, tenantCtx domain.TenantAuthContext, tenantID string, input UpdateTenantInput) (domain.Tenant, error) {
	if input.Name == "" {
		return domain.Tenant{}, invalidInput("invalid_tenant_payload")
	}
	if err := c.requirePathTenant(account, tenantCtx, tenantID, true); err != nil {
		return domain.Tenant{}, err
	}
	return c.store.UpdateTenant(tenantID, input.Name)
}

func (c *ControlPlane) DeleteTenant(account domain.Account, tenantCtx domain.TenantAuthContext, tenantID string) error {
	if err := c.requirePathTenant(account, tenantCtx, tenantID, true); err != nil {
		return err
	}
	return c.store.DeleteTenant(tenantID)
}

func (c *ControlPlane) TenantMembers(account domain.Account, tenantCtx domain.TenantAuthContext, tenantID string) ([]TenantMembershipAccount, error) {
	if err := c.requirePathTenant(account, tenantCtx, tenantID, false); err != nil {
		return nil, err
	}
	return c.tenantMembers(tenantID), nil
}

func (c *ControlPlane) UpsertTenantMembership(account domain.Account, tenantCtx domain.TenantAuthContext, tenantID string, memberAccountID string, input UpsertTenantMembershipInput) (TenantMembershipAccount, error) {
	if memberAccountID == "" || (input.Role != domain.TenantRoleAdmin && input.Role != domain.TenantRoleUser) {
		return TenantMembershipAccount{}, invalidInput("invalid_membership_payload")
	}
	if _, ok := c.accountByID(memberAccountID); !ok {
		return TenantMembershipAccount{}, invalidInput("invalid_membership_account")
	}
	if err := c.requirePathTenant(account, tenantCtx, tenantID, true); err != nil {
		return TenantMembershipAccount{}, err
	}
	membership, err := c.store.UpsertTenantMembership(tenantID, memberAccountID, input.Role, account.ID)
	if err != nil {
		return TenantMembershipAccount{}, err
	}
	return c.membershipAccount(memberAccountID, membership), nil
}

func (c *ControlPlane) DeleteTenantMembership(account domain.Account, tenantCtx domain.TenantAuthContext, tenantID string, memberAccountID string) error {
	if memberAccountID == "" {
		return invalidInput("missing_account_id")
	}
	if err := c.requirePathTenant(account, tenantCtx, tenantID, true); err != nil {
		return err
	}
	removed, ok := c.store.GetTenantMembership(memberAccountID, tenantID)
	if !ok {
		return sql.ErrNoRows
	}
	if err := c.store.DeleteTenantMembership(tenantID, memberAccountID); err != nil {
		return err
	}
	if removed.Role == domain.TenantRoleAdmin {
		return c.ensureTenantAdminContinuity(tenantID, account.ID)
	}
	return nil
}

func (c *ControlPlane) requirePathTenant(account domain.Account, tenantCtx domain.TenantAuthContext, tenantID string, requireAdmin bool) error {
	if tenantID == "" {
		return invalidInput("missing_tenant_id")
	}
	if account.Role == domain.AccountRoleSuperAdmin {
		if _, ok := c.store.GetTenant(tenantID); !ok {
			return sql.ErrNoRows
		}
		return nil
	}
	if tenantCtx.ActiveTenant.TenantID != tenantID {
		return newError(http.StatusForbidden, "tenant_forbidden")
	}
	if requireAdmin && tenantCtx.ActiveTenant.Role != domain.TenantRoleAdmin {
		return newError(http.StatusForbidden, "tenant_role_forbidden")
	}
	return nil
}

func (c *ControlPlane) ensureTenantAdminContinuity(tenantID string, createID string) error {
	members := c.tenantMembers(tenantID)
	for _, member := range members {
		if member.Role == domain.TenantRoleAdmin {
			return nil
		}
	}
	fallbackID := fallbackTenantAdminID(members, c.store.ListAccounts())
	if fallbackID == "" {
		return nil
	}
	_, err := c.store.UpsertTenantMembership(tenantID, fallbackID, domain.TenantRoleAdmin, createID)
	return err
}

func fallbackTenantAdminID(members []TenantMembershipAccount, accounts []domain.Account) string {
	memberIDs := make([]string, 0, len(members))
	for _, member := range members {
		memberIDs = append(memberIDs, member.AccountID)
	}
	sort.Strings(memberIDs)
	if len(memberIDs) > 0 {
		return memberIDs[0]
	}
	superAdminIDs := make([]string, 0)
	for _, account := range accounts {
		if account.Role == domain.AccountRoleSuperAdmin && account.Status == domain.AccountStatusActive {
			superAdminIDs = append(superAdminIDs, account.ID)
		}
	}
	sort.Strings(superAdminIDs)
	if len(superAdminIDs) == 0 {
		return ""
	}
	return superAdminIDs[0]
}

func (c *ControlPlane) tenantMembers(tenantID string) []TenantMembershipAccount {
	accounts := c.store.ListAccounts()
	items := make([]TenantMembershipAccount, 0)
	for _, account := range accounts {
		membership, ok := c.store.GetTenantMembership(account.ID, tenantID)
		if !ok {
			continue
		}
		items = append(items, TenantMembershipAccount{
			AccountID:  account.ID,
			Account:    account.Account,
			TenantID:   membership.TenantID,
			TenantName: membership.TenantName,
			Role:       membership.Role,
			JoinedAt:   membership.JoinedAt,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].AccountID < items[j].AccountID
	})
	return items
}

func (c *ControlPlane) membershipAccount(accountID string, membership domain.TenantMembership) TenantMembershipAccount {
	account, _ := c.accountByID(accountID)
	return TenantMembershipAccount{
		AccountID:  accountID,
		Account:    account.Account,
		TenantID:   membership.TenantID,
		TenantName: membership.TenantName,
		Role:       membership.Role,
		JoinedAt:   membership.JoinedAt,
	}
}

func (c *ControlPlane) accountByID(accountID string) (domain.Account, bool) {
	for _, account := range c.store.ListAccounts() {
		if account.ID == accountID {
			return account, true
		}
	}
	return domain.Account{}, false
}
