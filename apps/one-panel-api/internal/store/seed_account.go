package store

import (
	"fmt"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/auth"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (s *SeedStore) ListAccounts() []domain.Account {
	return []domain.Account{
		{
			ID:                 "acct-admin",
			Account:            "admin",
			Role:               "super_admin",
			Status:             "active",
			MustRotatePassword: true,
		},
	}
}

func (s *SeedStore) CreateAccount(input domain.CreateAccountInput) (domain.Account, error) {
	return domain.Account{
		ID:                 s.nextID("account"),
		Account:            input.Account,
		Role:               input.Role,
		Status:             "active",
		MustRotatePassword: false,
	}, nil
}

func (s *SeedStore) DeleteAccount(accountID string) error {
	if accountID == "acct-admin" {
		return fmt.Errorf("cannot_delete_admin")
	}
	return nil
}

func (s *SeedStore) UpdateAccount(accountID string, input domain.UpdateAccountInput) (domain.Account, error) {
	role := input.Role
	if role == "" {
		role = "super_admin"
	}
	status := input.Status
	if status == "" {
		status = "active"
	}
	return domain.Account{
		ID:                 accountID,
		Account:            "admin",
		Role:               role,
		Status:             status,
		MustRotatePassword: false,
	}, nil
}

func (s *SeedStore) Authenticate(account string, password string) (domain.LoginResult, bool) {
	if account != "admin" || password != s.adminPassword {
		return domain.LoginResult{}, false
	}
	return domain.LoginResult{
		Account: domain.Account{
			ID:                 "acct-admin",
			Account:            "admin",
			Role:               "super_admin",
			Status:             "active",
			MustRotatePassword: true,
		},
		AccessToken:        "seed-access-token",
		RefreshToken:       "seed-refresh-token",
		ExpiresAt:          "2026-04-25T14:00:00Z",
		MustRotatePassword: true,
	}, true
}

func (s *SeedStore) RefreshSession(refreshToken string) (domain.LoginResult, bool) {
	if refreshToken == "" {
		return domain.LoginResult{}, false
	}
	accessToken, _ := auth.RandomToken()
	nextRefresh, _ := auth.RandomToken()
	return domain.LoginResult{
		Account: domain.Account{
			ID:                 "acct-admin",
			Account:            "admin",
			Role:               "super_admin",
			Status:             "active",
			MustRotatePassword: true,
		},
		AccessToken:        accessToken,
		RefreshToken:       nextRefresh,
		ExpiresAt:          "2026-04-25T16:00:00Z",
		MustRotatePassword: true,
	}, true
}

func (s *SeedStore) AuthenticateAccessToken(accessToken string) (domain.Account, bool) {
	if accessToken == "" {
		return domain.Account{}, false
	}
	return domain.Account{
		ID:                 "acct-admin",
		Account:            "admin",
		Role:               "super_admin",
		Status:             "active",
		MustRotatePassword: true,
	}, true
}

func (s *SeedStore) Logout(accessToken string) bool {
	return accessToken != ""
}
