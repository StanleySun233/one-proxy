package domain

const (
	RemoteProtocolSSH = "ssh"
	RemoteProtocolRDP = "rdp"

	RemoteCredentialScopePersonal = "personal"
	RemoteCredentialScopeTenant   = "tenant"
)

type RemoteCredential struct {
	ID               string `json:"id"`
	TenantID         string `json:"tenantId,omitempty"`
	AccountID        string `json:"accountId"`
	Name             string `json:"name"`
	Protocol         string `json:"protocol"`
	Scope            string `json:"scope"`
	Username         string `json:"username"`
	SecretType       string `json:"secretType"`
	EncryptedPayload string `json:"encryptedPayload"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
	LastUsedAt       string `json:"lastUsedAt,omitempty"`
}

type CreateRemoteCredentialInput struct {
	Name             string `json:"name"`
	Protocol         string `json:"protocol"`
	Scope            string `json:"scope"`
	Username         string `json:"username"`
	SecretType       string `json:"secretType"`
	EncryptedPayload string `json:"encryptedPayload"`
}

type UpdateRemoteCredentialInput struct {
	Name             string `json:"name"`
	Username         string `json:"username"`
	SecretType       string `json:"secretType"`
	EncryptedPayload string `json:"encryptedPayload"`
}

type RemoteSessionInput struct {
	AccessPathID string `json:"accessPathId"`
	CredentialID string `json:"credentialId"`
	Protocol     string `json:"protocol"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	PrivateKey   string `json:"privateKey"`
	Passphrase   string `json:"passphrase"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	DPI          int    `json:"dpi"`
}

type RemoteSession struct {
	ID        string `json:"id"`
	Token     string `json:"token"`
	Protocol  string `json:"protocol"`
	ExpiresAt string `json:"expiresAt"`
	TunnelURL string `json:"tunnelUrl"`
}
