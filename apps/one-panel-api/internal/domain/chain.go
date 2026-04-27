package domain

type Chain struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	DestinationScope string   `json:"destinationScope"`
	Enabled          bool     `json:"enabled"`
	Hops             []string `json:"hops"`
}

type ChainWithDetails struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	DestinationScope string           `json:"destinationScope"`
	Enabled          bool             `json:"enabled"`
	Hops             []string         `json:"hops"`
	HopDetails       []ChainHopDetail `json:"hopDetails"`
}

type ChainHopDetail struct {
	NodeID   string `json:"nodeId"`
	NodeName string `json:"nodeName"`
	Mode     string `json:"mode"`
}

type CreateChainInput struct {
	Name             string   `json:"name"`
	DestinationScope string   `json:"destinationScope"`
	Hops             []string `json:"hops"`
}

type UpdateChainInput struct {
	Name             string   `json:"name"`
	DestinationScope string   `json:"destinationScope"`
	Hops             []string `json:"hops"`
	Enabled          bool     `json:"enabled"`
}

type ValidateChainInput struct {
	Name             string   `json:"name"`
	Hops             []string `json:"hops"`
	DestinationScope string   `json:"destinationScope"`
}

type HopConnectivity struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Reachable  bool   `json:"reachable"`
}

type ScopeOwnership struct {
	Scope       string `json:"scope"`
	OwnerNodeID string `json:"ownerNodeId"`
	Valid       bool   `json:"valid"`
}

type ChainValidationResult struct {
	Valid           bool              `json:"valid"`
	Errors          []string          `json:"errors"`
	Warnings        []string          `json:"warnings"`
	HopConnectivity []HopConnectivity `json:"hopConnectivity"`
	ScopeOwnership  ScopeOwnership    `json:"scopeOwnership"`
}

type PreviewChainInput struct {
	Name             string   `json:"name"`
	Hops             []string `json:"hops"`
	DestinationScope string   `json:"destinationScope"`
}

type CompiledChainConfig struct {
	ChainID          string           `json:"chainId"`
	Name             string           `json:"name"`
	Hops             []ChainHopDetail `json:"hops"`
	DestinationScope string           `json:"destinationScope"`
	RoutingPath      string           `json:"routingPath"`
}

type ChainPreviewResult struct {
	CompiledConfig CompiledChainConfig `json:"compiledConfig"`
}

type ChainProbeHop struct {
	NodeID        string `json:"nodeId"`
	NodeName      string `json:"nodeName"`
	TransportType string `json:"transportType"`
	Address       string `json:"address"`
	Status        string `json:"status"`
}

type ChainProbeResult struct {
	ChainID         string          `json:"chainId"`
	Status          string          `json:"status"`
	Message         string          `json:"message"`
	ResolvedHops    []ChainProbeHop `json:"resolvedHops"`
	BlockingNodeID  string          `json:"blockingNodeId"`
	BlockingReason  string          `json:"blockingReason"`
	TargetHost      string          `json:"targetHost"`
	TargetPort      int             `json:"targetPort"`
	ProbedAt        string          `json:"probedAt"`
}

type SaveChainProbeResultInput struct {
	ChainID         string          `json:"chainId"`
	Status          string          `json:"status"`
	Message         string          `json:"message"`
	ResolvedHops    []ChainProbeHop `json:"resolvedHops"`
	BlockingNodeID  string          `json:"blockingNodeId"`
	BlockingReason  string          `json:"blockingReason"`
	TargetHost      string          `json:"targetHost"`
	TargetPort      int             `json:"targetPort"`
	ProbedAt        string          `json:"probedAt"`
}
