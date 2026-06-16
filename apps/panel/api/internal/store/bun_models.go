package store

import "github.com/uptrace/bun"

type ChainModel struct {
	bun.BaseModel `bun:"table:chains"`

	ID               string `bun:"id,pk"`
	Name             string `bun:"name"`
	DestinationScope string `bun:"destination_scope"`
	Enabled          bool   `bun:"enabled"`
	CreateID         string `bun:"create_id"`
	OwnerID          string `bun:"owner_id"`
	CreatedAt        string `bun:"created_at"`
	UpdatedAt        string `bun:"updated_at"`
}

type ChainHopModel struct {
	bun.BaseModel `bun:"table:chain_hops"`

	ChainID  string `bun:"chain_id,pk"`
	HopIndex int    `bun:"hop_index,pk"`
	NodeID   string `bun:"node_id"`
}

type RouteRuleModel struct {
	bun.BaseModel `bun:"table:route_rules"`

	ID               string `bun:"id,pk"`
	Priority         int    `bun:"priority"`
	MatchType        string `bun:"match_type"`
	MatchValue       string `bun:"match_value"`
	ActionType       string `bun:"action_type"`
	ChainID          string `bun:"chain_id,nullzero"`
	DestinationScope string `bun:"destination_scope,nullzero"`
	Enabled          bool   `bun:"enabled"`
	CreateID         string `bun:"create_id"`
	OwnerID          string `bun:"owner_id"`
	CreatedAt        string `bun:"created_at"`
	UpdatedAt        string `bun:"updated_at"`
}

type NodeAccessPathModel struct {
	bun.BaseModel `bun:"table:node_access_paths"`

	ID               string `bun:"id,pk"`
	ChainID          string `bun:"chain_id,nullzero"`
	Name             string `bun:"name"`
	Mode             string `bun:"mode"`
	Protocol         string `bun:"protocol"`
	ServiceType      string `bun:"service_type"`
	TargetNodeID     string `bun:"target_node_id,nullzero"`
	EntryNodeID      string `bun:"entry_node_id,nullzero"`
	RelayNodeIDsJSON string `bun:"relay_node_ids_json"`
	ListenHost       string `bun:"listen_host,nullzero"`
	ListenPort       int    `bun:"listen_port"`
	TargetProtocol   string `bun:"target_protocol"`
	TargetHost       string `bun:"target_host,nullzero"`
	TargetPort       int    `bun:"target_port"`
	TargetSNI        string `bun:"target_sni,nullzero"`
	TLSMode          string `bun:"tls_mode"`
	AuthMode         string `bun:"auth_mode"`
	OptionsJSON      string `bun:"options_json"`
	Enabled          bool   `bun:"enabled"`
	CreateID         string `bun:"create_id"`
	OwnerID          string `bun:"owner_id"`
	CreatedAt        string `bun:"created_at"`
	UpdatedAt        string `bun:"updated_at"`
}

type NodeOnboardingTaskModel struct {
	bun.BaseModel `bun:"table:node_onboarding_tasks"`

	ID                   string `bun:"id,pk"`
	Mode                 string `bun:"mode"`
	PathID               string `bun:"path_id,nullzero"`
	TargetNodeID         string `bun:"target_node_id,nullzero"`
	TargetHost           string `bun:"target_host,nullzero"`
	TargetPort           int    `bun:"target_port"`
	Status               string `bun:"status"`
	StatusMessage        string `bun:"status_message"`
	RequestedByAccountID string `bun:"requested_by_account_id"`
	CreatedAt            string `bun:"created_at"`
	UpdatedAt            string `bun:"updated_at"`
}

type ChainProbeResultModel struct {
	bun.BaseModel `bun:"table:chain_probe_results"`

	ChainID          string `bun:"chain_id,pk"`
	Status           string `bun:"status"`
	Message          string `bun:"message"`
	ResolvedHopsJSON string `bun:"resolved_hops_json"`
	BlockingNodeID   string `bun:"blocking_node_id,nullzero"`
	BlockingReason   string `bun:"blocking_reason,nullzero"`
	TargetHost       string `bun:"target_host,nullzero"`
	TargetPort       int    `bun:"target_port"`
	ProbedAt         string `bun:"probed_at"`
}

type TenantChainModel struct {
	bun.BaseModel `bun:"table:tenant_chains"`

	TenantID   string `bun:"tenant_id,pk"`
	ChainID    string `bun:"chain_id,pk"`
	Permission string `bun:"permission"`
	CreateID   string `bun:"create_id"`
	CreatedAt  string `bun:"created_at"`
}

type TenantRouteRuleModel struct {
	bun.BaseModel `bun:"table:tenant_route_rules"`

	TenantID    string `bun:"tenant_id,pk"`
	RouteRuleID string `bun:"route_rule_id,pk"`
	Permission  string `bun:"permission"`
	CreateID    string `bun:"create_id"`
	CreatedAt   string `bun:"created_at"`
}

type TenantAccessPathModel struct {
	bun.BaseModel `bun:"table:tenant_access_paths"`

	TenantID     string `bun:"tenant_id,pk"`
	AccessPathID string `bun:"access_path_id,pk"`
	Permission   string `bun:"permission"`
	CreateID     string `bun:"create_id"`
	CreatedAt    string `bun:"created_at"`
}
