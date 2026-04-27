package domain

type Account struct {
	ID                 string `json:"id"`
	Account            string `json:"account"`
	Role               string `json:"role"`
	Status             string `json:"status"`
	MustRotatePassword bool   `json:"mustRotatePassword"`
}

type CreateAccountInput struct {
	Account  string `json:"account"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type UpdateAccountInput struct {
	Password string `json:"password"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

type LoginResult struct {
	Account            Account `json:"account"`
	AccessToken        string  `json:"accessToken"`
	RefreshToken       string  `json:"refreshToken"`
	ExpiresAt          string  `json:"expiresAt"`
	MustRotatePassword bool    `json:"mustRotatePassword"`
}

type RefreshSessionInput struct {
	RefreshToken string `json:"refreshToken"`
}

type LogoutInput struct {
	RefreshToken string `json:"refreshToken"`
}

type ExtensionBootstrap struct {
	Account        Account          `json:"account"`
	PolicyRevision string           `json:"policyRevision"`
	FetchedAt      string           `json:"fetchedAt"`
	Groups         []ExtensionGroup `json:"groups"`
}

type ExtensionGroup struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	EntryNodeID   string   `json:"entryNodeId"`
	EntryNodeName string   `json:"entryNodeName"`
	ProxyScheme   string   `json:"proxyScheme"`
	ProxyHost     string   `json:"proxyHost"`
	ProxyPort     int      `json:"proxyPort"`
	ProxyHosts    []string `json:"proxyHosts"`
	ProxyCIDRs    []string `json:"proxyCidrs"`
	DirectHosts   []string `json:"directHosts"`
	DirectCIDRs   []string `json:"directCidrs"`
}
