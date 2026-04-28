package service

import "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"

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
	return c.store.Authenticate(account, password)
}

func (c *ControlPlane) AuthenticateAccessToken(accessToken string) (domain.Account, bool) {
	return c.store.AuthenticateAccessToken(accessToken)
}

func (c *ControlPlane) RefreshSession(refreshToken string) (domain.LoginResult, bool) {
	return c.store.RefreshSession(refreshToken)
}

func (c *ControlPlane) Logout(accessToken string) bool {
	return c.store.Logout(accessToken)
}
