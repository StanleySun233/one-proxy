package store

import (
	"fmt"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/auth"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

type SeedStore struct {
	adminPassword string
}

func NewSeedStore() *SeedStore {
	return &SeedStore{adminPassword: "admin"}
}

func (s *SeedStore) BootstrapAdminPassword() string {
	return s.adminPassword
}

func (s *SeedStore) GetOverview() domain.Overview {
	return domain.Overview{
		Nodes: domain.OverviewNodes{
			Healthy:  0,
			Degraded: 0,
		},
		Policies: domain.OverviewPolicies{},
		Certificates: domain.OverviewCertificates{
			RenewSoon: 0,
		},
	}
}

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
		ID:                 newID("acct"),
		Account:            input.Account,
		Role:               input.Role,
		Status:             "active",
		MustRotatePassword: false,
	}, nil
}

func (s *SeedStore) ListNodeLinks() []domain.NodeLink {
	return []domain.NodeLink{}
}

func (s *SeedStore) CreateNodeLink(input domain.CreateNodeLinkInput) (domain.NodeLink, error) {
	return domain.NodeLink{
		ID:           newID("link"),
		SourceNodeID: input.SourceNodeID,
		TargetNodeID: input.TargetNodeID,
		LinkType:     input.LinkType,
		TrustState:   input.TrustState,
	}, nil
}

func (s *SeedStore) ListNodeAccessPaths() []domain.NodeAccessPath {
	return []domain.NodeAccessPath{}
}

func (s *SeedStore) CreateNodeAccessPath(input domain.CreateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	return domain.NodeAccessPath{
		ID:           newID("path"),
		Name:         input.Name,
		Mode:         input.Mode,
		TargetNodeID: input.TargetNodeID,
		EntryNodeID:  input.EntryNodeID,
		RelayNodeIDs: normalizeStringSlice(input.RelayNodeIDs),
		TargetHost:   input.TargetHost,
		TargetPort:   input.TargetPort,
		Enabled:      true,
	}, nil
}

func (s *SeedStore) UpdateNodeAccessPath(pathID string, input domain.UpdateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	return domain.NodeAccessPath{
		ID:           pathID,
		Name:         input.Name,
		Mode:         input.Mode,
		TargetNodeID: input.TargetNodeID,
		EntryNodeID:  input.EntryNodeID,
		RelayNodeIDs: normalizeStringSlice(input.RelayNodeIDs),
		TargetHost:   input.TargetHost,
		TargetPort:   input.TargetPort,
		Enabled:      input.Enabled,
	}, nil
}

func (s *SeedStore) DeleteNodeAccessPath(pathID string) error {
	_ = pathID
	return nil
}

func (s *SeedStore) ListNodeOnboardingTasks() []domain.NodeOnboardingTask {
	return []domain.NodeOnboardingTask{}
}

func (s *SeedStore) CreateNodeOnboardingTask(accountID string, input domain.CreateNodeOnboardingTaskInput) (domain.NodeOnboardingTask, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	return domain.NodeOnboardingTask{
		ID:                   newID("task"),
		Mode:                 input.Mode,
		PathID:               input.PathID,
		TargetNodeID:         input.TargetNodeID,
		TargetHost:           input.TargetHost,
		TargetPort:           input.TargetPort,
		Status:               "planned",
		StatusMessage:        "task created",
		RequestedByAccountID: accountID,
		CreatedAt:            now,
		UpdatedAt:            now,
	}, nil
}

func (s *SeedStore) UpdateNodeOnboardingTaskStatus(taskID string, status string, statusMessage string) (domain.NodeOnboardingTask, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	return domain.NodeOnboardingTask{
		ID:            taskID,
		Status:        status,
		StatusMessage: statusMessage,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func (s *SeedStore) ListCertificates() []domain.Certificate {
	return []domain.Certificate{}
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

func (s *SeedStore) ListNodes() []domain.Node {
	return []domain.Node{}
}

func (s *SeedStore) ListNodeTransports() []domain.NodeTransport {
	return []domain.NodeTransport{}
}

func (s *SeedStore) UpsertNodeTransport(input domain.UpsertNodeTransportInput) (domain.NodeTransport, error) {
	return domain.NodeTransport{
		ID:              newID("transport"),
		NodeID:          input.NodeID,
		TransportType:   input.TransportType,
		Direction:       input.Direction,
		Address:         input.Address,
		Status:          input.Status,
		ParentNodeID:    input.ParentNodeID,
		ConnectedAt:     input.ConnectedAt,
		LastHeartbeatAt: input.LastHeartbeatAt,
		LatencyMs:       input.LatencyMs,
		Details:         input.Details,
	}, nil
}

func (s *SeedStore) CreateNode(input domain.CreateNodeInput) (domain.Node, error) {
	return domain.Node{
		ID:           fmt.Sprintf("node-%d", time.Now().UnixNano()),
		Name:         input.Name,
		Mode:         input.Mode,
		ScopeKey:     input.ScopeKey,
		ParentNodeID: input.ParentNodeID,
		Enabled:      true,
		Status:       "healthy",
		PublicHost:   input.PublicHost,
		PublicPort:   input.PublicPort,
	}, nil
}

func (s *SeedStore) ProvisionNodeAccess(nodeID string) (domain.ApproveNodeEnrollmentResult, error) {
	return domain.ApproveNodeEnrollmentResult{
		Node: domain.Node{
			ID:       nodeID,
			Name:     "seed-node",
			Mode:     "relay",
			ScopeKey: "seed-scope",
			Enabled:  true,
			Status:   "healthy",
		},
		AccessToken:   "seed-node-token",
		TrustMaterial: "seed-shared-secret",
		ExpiresAt:     time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339),
	}, nil
}

func (s *SeedStore) UpdateNode(nodeID string, input domain.UpdateNodeInput) (domain.Node, error) {
	return domain.Node{
		ID:           nodeID,
		Name:         input.Name,
		Mode:         input.Mode,
		ScopeKey:     input.ScopeKey,
		ParentNodeID: input.ParentNodeID,
		Enabled:      input.Enabled,
		Status:       input.Status,
		PublicHost:   input.PublicHost,
		PublicPort:   input.PublicPort,
	}, nil
}

func (s *SeedStore) DeleteNode(nodeID string) error {
	_ = nodeID
	return nil
}

func (s *SeedStore) ListChains() []domain.Chain {
	return []domain.Chain{}
}

func (s *SeedStore) GetChainProbeResult(chainID string) (domain.ChainProbeResult, bool) {
	_ = chainID
	return domain.ChainProbeResult{}, false
}

func (s *SeedStore) SaveChainProbeResult(input domain.SaveChainProbeResultInput) (domain.ChainProbeResult, error) {
	return domain.ChainProbeResult{
		ChainID:        input.ChainID,
		Status:         input.Status,
		Message:        input.Message,
		ResolvedHops:   input.ResolvedHops,
		BlockingNodeID: input.BlockingNodeID,
		BlockingReason: input.BlockingReason,
		TargetHost:     input.TargetHost,
		TargetPort:     input.TargetPort,
		ProbedAt:       input.ProbedAt,
	}, nil
}

func (s *SeedStore) CreateChain(input domain.CreateChainInput) (domain.Chain, error) {
	return domain.Chain{
		ID:               fmt.Sprintf("chain-%d", time.Now().UnixNano()),
		Name:             input.Name,
		DestinationScope: input.DestinationScope,
		Enabled:          true,
		Hops:             input.Hops,
	}, nil
}

func (s *SeedStore) UpdateChain(chainID string, input domain.UpdateChainInput) (domain.Chain, error) {
	return domain.Chain{
		ID:               chainID,
		Name:             input.Name,
		DestinationScope: input.DestinationScope,
		Enabled:          input.Enabled,
		Hops:             input.Hops,
	}, nil
}

func (s *SeedStore) DeleteChain(chainID string) error {
	_ = chainID
	return nil
}

func (s *SeedStore) ListRouteRules() []domain.RouteRule {
	return []domain.RouteRule{}
}

func (s *SeedStore) CreateRouteRule(input domain.CreateRouteRuleInput) (domain.RouteRule, error) {
	return domain.RouteRule{
		ID:               fmt.Sprintf("rule-%d", time.Now().UnixNano()),
		Priority:         input.Priority,
		MatchType:        input.MatchType,
		MatchValue:       input.MatchValue,
		ActionType:       input.ActionType,
		ChainID:          input.ChainID,
		DestinationScope: input.DestinationScope,
		Enabled:          true,
	}, nil
}

func (s *SeedStore) UpdateRouteRule(ruleID string, input domain.UpdateRouteRuleInput) (domain.RouteRule, error) {
	return domain.RouteRule{
		ID:               ruleID,
		Priority:         input.Priority,
		MatchType:        input.MatchType,
		MatchValue:       input.MatchValue,
		ActionType:       input.ActionType,
		ChainID:          input.ChainID,
		DestinationScope: input.DestinationScope,
		Enabled:          input.Enabled,
	}, nil
}

func (s *SeedStore) DeleteRouteRule(ruleID string) error {
	_ = ruleID
	return nil
}

func (s *SeedStore) ListNodeHealth() []domain.NodeHealth {
	return []domain.NodeHealth{}
}

func (s *SeedStore) CreateBootstrapToken(input domain.CreateBootstrapTokenInput) (domain.BootstrapToken, error) {
	token, _ := auth.RandomToken()
	return domain.BootstrapToken{
		ID:         newID("bootstrap"),
		Token:      token,
		TargetType: input.TargetType,
		TargetID:   input.TargetID,
		ExpiresAt:  time.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339),
	}, nil
}

func (s *SeedStore) EnrollNode(input domain.EnrollNodeInput) (domain.EnrollNodeResult, error) {
	enrollmentSecret, _ := auth.RandomToken()
	return domain.EnrollNodeResult{
		Node: domain.Node{
			ID:           newID("node"),
			Name:         input.Name,
			Mode:         input.Mode,
			ScopeKey:     input.ScopeKey,
			ParentNodeID: input.ParentNodeID,
			Enabled:      true,
			Status:       "pending",
			PublicHost:   input.PublicHost,
			PublicPort:   input.PublicPort,
		},
		EnrollmentSecret: enrollmentSecret,
		ApprovalState:    "pending",
	}, nil
}

func (s *SeedStore) ApproveNodeEnrollment(nodeID string) (domain.ApproveNodeEnrollmentResult, error) {
	accessToken, _ := auth.RandomToken()
	trustMaterial, _ := auth.RandomToken()
	return domain.ApproveNodeEnrollmentResult{
		Node: domain.Node{
			ID:       nodeID,
			Name:     nodeID,
			Mode:     "relay",
			ScopeKey: "seed-scope",
			Enabled:  true,
			Status:   "degraded",
		},
		AccessToken:   accessToken,
		TrustMaterial: trustMaterial,
		ExpiresAt:     time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339),
	}, nil
}

func (s *SeedStore) ExchangeNodeEnrollment(input domain.ExchangeNodeEnrollmentInput) (domain.ApproveNodeEnrollmentResult, error) {
	return s.ApproveNodeEnrollment(input.NodeID)
}

func (s *SeedStore) ListPolicyRevisions() []domain.PolicyRevision {
	return []domain.PolicyRevision{}
}

func (s *SeedStore) PublishPolicy(accountID string) (domain.PolicyRevision, error) {
	_ = accountID
	return domain.PolicyRevision{
		ID:            newID("policy"),
		Version:       fmt.Sprintf("rev-%d", time.Now().Unix()),
		Status:        "published",
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		AssignedNodes: 0,
	}, nil
}

func (s *SeedStore) AuthenticateNodeToken(accessToken string) (string, bool) {
	_ = accessToken
	return "", false
}

func (s *SeedStore) GetNodeAgentPolicy(nodeID string) (domain.NodeAgentPolicy, bool) {
	_ = nodeID
	return domain.NodeAgentPolicy{}, false
}

func (s *SeedStore) UpsertNodeHeartbeat(input domain.NodeHeartbeatInput) (domain.NodeHealth, error) {
	return domain.NodeHealth{
		NodeID:           input.NodeID,
		HeartbeatAt:      time.Now().UTC().Format(time.RFC3339),
		PolicyRevisionID: input.PolicyRevisionID,
		ListenerStatus:   input.ListenerStatus,
		CertStatus:       input.CertStatus,
	}, nil
}

func (s *SeedStore) RenewNodeCertificate(input domain.NodeCertRenewInput) (domain.NodeCertRenewResult, error) {
	return domain.NodeCertRenewResult{
		NodeID:   input.NodeID,
		CertType: input.CertType,
		Status:   "renewed",
		NotAfter: time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339),
	}, nil
}

func (s *SeedStore) CleanupExpiredSessions() (int64, error) {
	return 0, nil
}

func (s *SeedStore) CleanupExpiredBootstrapTokens() (int64, error) {
	return 0, nil
}

func (s *SeedStore) CleanupExpiredNodeTokens() (int64, error) {
	return 0, nil
}

func (s *SeedStore) RefreshCertificateStatus(window time.Duration) error {
	_ = window
	return nil
}

func (s *SeedStore) RefreshNodeStatus(staleAfter time.Duration) error {
	_ = staleAfter
	return nil
}
