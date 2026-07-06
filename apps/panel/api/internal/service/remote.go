package service

import (
	"net/http"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

const remoteSessionTTL = 5 * time.Minute

type remoteSessionRecord struct {
	SessionID     string
	Token         string
	Protocol      string
	Username      string
	Password      string
	PrivateKey    string
	Passphrase    string
	Width         int
	Height        int
	DPI           int
	TargetHost    string
	TargetPort    int
	TCPAccessHost string
	TCPAccessPort int
	ChainNodeIDs  []string
	ProxyToken    string
	ExpiresAt     time.Time
}

func (c *ControlPlane) RemoteCredentials(account domain.Account, tenantCtx domain.TenantAuthContext, protocol string) []domain.RemoteCredential {
	if !validRemoteProtocol(protocol) {
		return []domain.RemoteCredential{}
	}
	return c.store.ListRemoteCredentials(account, tenantCtx, protocol)
}

func (c *ControlPlane) CreateRemoteCredential(account domain.Account, tenantCtx domain.TenantAuthContext, input domain.CreateRemoteCredentialInput) (domain.RemoteCredential, error) {
	input = normalizeCreateRemoteCredentialInput(input)
	if err := validateCreateRemoteCredentialInput(tenantCtx, input); err != nil {
		return domain.RemoteCredential{}, err
	}
	return c.store.CreateRemoteCredential(account, tenantCtx, input)
}

func (c *ControlPlane) UpdateRemoteCredential(account domain.Account, tenantCtx domain.TenantAuthContext, credentialID string, input domain.UpdateRemoteCredentialInput) (domain.RemoteCredential, error) {
	if credentialID == "" {
		return domain.RemoteCredential{}, invalidInput("missing_remote_credential_id")
	}
	input = normalizeUpdateRemoteCredentialInput(input)
	if input.Name == "" || input.Username == "" || input.SecretType == "" || input.EncryptedPayload == "" {
		return domain.RemoteCredential{}, invalidInput("invalid_remote_credential_payload")
	}
	current, ok := c.store.RemoteCredential(account, tenantCtx, credentialID)
	if !ok {
		return domain.RemoteCredential{}, newError(http.StatusNotFound, "remote_credential_not_found")
	}
	if current.Scope == domain.RemoteCredentialScopeTenant && !canManageTenantRemoteCredentials(tenantCtx) {
		return domain.RemoteCredential{}, newError(http.StatusForbidden, "tenant_role_forbidden")
	}
	return c.store.UpdateRemoteCredential(account, tenantCtx, credentialID, input)
}

func (c *ControlPlane) DeleteRemoteCredential(account domain.Account, tenantCtx domain.TenantAuthContext, credentialID string) error {
	if credentialID == "" {
		return invalidInput("missing_remote_credential_id")
	}
	current, ok := c.store.RemoteCredential(account, tenantCtx, credentialID)
	if !ok {
		return newError(http.StatusNotFound, "remote_credential_not_found")
	}
	if current.Scope == domain.RemoteCredentialScopeTenant && !canManageTenantRemoteCredentials(tenantCtx) {
		return newError(http.StatusForbidden, "tenant_role_forbidden")
	}
	return c.store.DeleteRemoteCredential(account, tenantCtx, credentialID)
}

func (c *ControlPlane) CreateRemoteSession(account domain.Account, tenantCtx domain.TenantAuthContext, input domain.RemoteSessionInput) (domain.RemoteSession, error) {
	input = normalizeRemoteSessionInput(input)
	if tenantCtx.ActiveTenant.TenantID == "" {
		return domain.RemoteSession{}, invalidInput("tenant_required")
	}
	if err := validateRemoteSessionInput(input); err != nil {
		return domain.RemoteSession{}, err
	}
	path, ok := c.accessPathForTenant(tenantCtx, input.AccessPathID)
	if !ok || !path.Enabled {
		return domain.RemoteSession{}, newError(http.StatusNotFound, "access_path_not_found")
	}
	if path.Mode != domain.PathModeTCP || path.Protocol != domain.AccessProtocolTCP || path.ServiceType != domain.AccessServiceTCPAccess {
		return domain.RemoteSession{}, invalidInput("remote_access_path_must_be_tcp")
	}
	if path.TargetHost == "" || path.TargetPort < 1 || path.TargetPort > 65535 || path.EntryNodeID == "" || path.TargetNodeID == "" {
		return domain.RemoteSession{}, invalidInput("invalid_remote_access_path")
	}
	if input.CredentialID != "" {
		credential, ok := c.store.RemoteCredential(account, tenantCtx, input.CredentialID)
		if !ok {
			return domain.RemoteSession{}, newError(http.StatusNotFound, "remote_credential_not_found")
		}
		if credential.Protocol != input.Protocol {
			return domain.RemoteSession{}, invalidInput("remote_credential_protocol_mismatch")
		}
		if err := c.store.TouchRemoteCredential(input.CredentialID); err != nil {
			return domain.RemoteSession{}, internalFailure("remote_credential_touch_failed")
		}
	}
	tcpAccessHost, tcpAccessPort, err := c.remoteTCPAccessEndpoint(tenantCtx, path)
	if err != nil {
		return domain.RemoteSession{}, err
	}
	proxyToken, _, ok := c.IssueProxyToken(account, tenantCtx)
	if !ok {
		return domain.RemoteSession{}, internalFailure("remote_proxy_token_issue_failed")
	}
	sessionID, err := randomURLToken()
	if err != nil {
		return domain.RemoteSession{}, internalFailure("remote_session_issue_failed")
	}
	sessionToken, err := randomURLToken()
	if err != nil {
		return domain.RemoteSession{}, internalFailure("remote_session_issue_failed")
	}
	expiresAt := time.Now().UTC().Add(remoteSessionTTL)
	record := remoteSessionRecord{
		SessionID:     sessionID,
		Token:         sessionToken,
		Protocol:      input.Protocol,
		Username:      input.Username,
		Password:      input.Password,
		PrivateKey:    input.PrivateKey,
		Passphrase:    input.Passphrase,
		Width:         input.Width,
		Height:        input.Height,
		DPI:           input.DPI,
		TargetHost:    path.TargetHost,
		TargetPort:    path.TargetPort,
		TCPAccessHost: tcpAccessHost,
		TCPAccessPort: tcpAccessPort,
		ChainNodeIDs:  remoteChainNodeIDs(path),
		ProxyToken:    proxyToken,
		ExpiresAt:     expiresAt,
	}
	c.remoteMu.Lock()
	c.cleanupRemoteSessionsLocked(time.Now().UTC())
	c.remoteSessions[sessionID] = record
	c.remoteMu.Unlock()
	return domain.RemoteSession{
		ID:        sessionID,
		Token:     sessionToken,
		Protocol:  input.Protocol,
		ExpiresAt: expiresAt.Format(time.RFC3339),
		TunnelURL: "/api/remote/sessions/" + sessionID + "/tunnel",
	}, nil
}

func (c *ControlPlane) consumeRemoteSession(sessionID string, token string) (remoteSessionRecord, bool) {
	c.remoteMu.Lock()
	defer c.remoteMu.Unlock()
	c.cleanupRemoteSessionsLocked(time.Now().UTC())
	record, ok := c.remoteSessions[sessionID]
	if !ok || record.Token != token {
		return remoteSessionRecord{}, false
	}
	delete(c.remoteSessions, sessionID)
	return record, true
}

func (c *ControlPlane) cleanupRemoteSessionsLocked(now time.Time) {
	for sessionID, record := range c.remoteSessions {
		if now.After(record.ExpiresAt) {
			delete(c.remoteSessions, sessionID)
		}
	}
}

func normalizeCreateRemoteCredentialInput(input domain.CreateRemoteCredentialInput) domain.CreateRemoteCredentialInput {
	input.Name = strings.TrimSpace(input.Name)
	input.Protocol = strings.TrimSpace(input.Protocol)
	input.Scope = strings.TrimSpace(input.Scope)
	input.Username = strings.TrimSpace(input.Username)
	input.SecretType = strings.TrimSpace(input.SecretType)
	return input
}

func normalizeUpdateRemoteCredentialInput(input domain.UpdateRemoteCredentialInput) domain.UpdateRemoteCredentialInput {
	input.Name = strings.TrimSpace(input.Name)
	input.Username = strings.TrimSpace(input.Username)
	input.SecretType = strings.TrimSpace(input.SecretType)
	return input
}

func normalizeRemoteSessionInput(input domain.RemoteSessionInput) domain.RemoteSessionInput {
	input.AccessPathID = strings.TrimSpace(input.AccessPathID)
	input.CredentialID = strings.TrimSpace(input.CredentialID)
	input.Protocol = strings.TrimSpace(input.Protocol)
	input.Username = strings.TrimSpace(input.Username)
	if input.Width <= 0 {
		input.Width = 1024
	}
	if input.Height <= 0 {
		input.Height = 768
	}
	if input.DPI <= 0 {
		input.DPI = 96
	}
	return input
}

func validateCreateRemoteCredentialInput(tenantCtx domain.TenantAuthContext, input domain.CreateRemoteCredentialInput) error {
	if !validRemoteProtocol(input.Protocol) || input.Name == "" || input.Scope == "" || input.Username == "" || input.SecretType == "" || input.EncryptedPayload == "" {
		return invalidInput("invalid_remote_credential_payload")
	}
	switch input.Scope {
	case domain.RemoteCredentialScopePersonal:
		return nil
	case domain.RemoteCredentialScopeTenant:
		if tenantCtx.ActiveTenant.TenantID == "" {
			return invalidInput("tenant_required")
		}
		if !canManageTenantRemoteCredentials(tenantCtx) {
			return newError(http.StatusForbidden, "tenant_role_forbidden")
		}
		return nil
	default:
		return invalidInput("invalid_remote_credential_scope")
	}
}

func validateRemoteSessionInput(input domain.RemoteSessionInput) error {
	if !validRemoteProtocol(input.Protocol) || input.AccessPathID == "" || input.Username == "" {
		return invalidInput("invalid_remote_session_payload")
	}
	switch input.Protocol {
	case domain.RemoteProtocolSSH:
		if input.Password == "" && input.PrivateKey == "" {
			return invalidInput("invalid_remote_session_secret")
		}
	case domain.RemoteProtocolRDP:
		if input.Password == "" {
			return invalidInput("invalid_remote_session_secret")
		}
	}
	return nil
}

func validRemoteProtocol(protocol string) bool {
	return protocol == domain.RemoteProtocolSSH || protocol == domain.RemoteProtocolRDP
}

func canManageTenantRemoteCredentials(tenantCtx domain.TenantAuthContext) bool {
	return tenantCtx.SuperAdmin || tenantCtx.ActiveTenant.Role == domain.TenantRoleAdmin
}

func (c *ControlPlane) remoteTCPAccessEndpoint(tenantCtx domain.TenantAuthContext, path domain.NodeAccessPath) (string, int, error) {
	if path.ListenPort < 1 || path.ListenPort > 65535 {
		return "", 0, invalidInput("relay_entry_unavailable")
	}
	host := strings.TrimSpace(path.ListenHost)
	if host == "" || host == "0.0.0.0" || host == "::" {
		nodes := c.store.ListNodesForTenant(tenantCtx)
		entryNode, ok := nodeByID(nodes, path.EntryNodeID)
		if !ok || entryNode.PublicHost == "" {
			return "", 0, invalidInput("relay_entry_unavailable")
		}
		host = entryNode.PublicHost
	}
	return host, path.ListenPort, nil
}

func remoteChainNodeIDs(path domain.NodeAccessPath) []string {
	nodeIDs := append([]string(nil), path.RelayNodeIDs...)
	if path.TargetNodeID != "" && path.TargetNodeID != path.EntryNodeID {
		nodeIDs = append(nodeIDs, path.TargetNodeID)
	}
	return uniqueStrings(nodeIDs)
}
