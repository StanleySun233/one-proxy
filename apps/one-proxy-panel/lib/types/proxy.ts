import type {ActionType, MatchType, NodeMode, ProbeResultStatus, TransportStatus, TransportType} from './common';

export type Chain = {
  id: string;
  name: string;
  destinationScope: string;
  enabled: boolean;
  hops: string[];
};

export type ChainProbeHop = {
  nodeId: string;
  nodeName: string;
  transportType: TransportType;
  address: string;
  status: TransportStatus;
};

export type ChainProbeResult = {
  chainId: string;
  status: ProbeResultStatus;
  message: string;
  resolvedHops: ChainProbeHop[];
  blockingNodeId: string;
  blockingReason: string;
  targetHost: string;
  targetPort: number;
  probedAt: string;
};

export type ChainValidationResult = {
  valid: boolean;
  errors: string[];
  warnings: string[];
  hopConnectivity: { from: string; to: string; reachable: boolean }[];
  scopeOwnership: { scope: string; ownerNodeId: string; valid: boolean };
};

export type CompiledChainHop = {
  nodeId: string;
  nodeName: string;
  mode: NodeMode;
};

export type CompiledChainConfig = {
  chainId: string;
  name: string;
  hops: CompiledChainHop[];
  destinationScope: string;
  routingPath: string;
};

export type ChainPreviewResult = {
  compiledConfig: CompiledChainConfig;
};

export type RouteRule = {
  id: string;
  priority: number;
  matchType: MatchType;
  matchValue: string;
  actionType: ActionType;
  chainId?: string;
  destinationScope?: string;
  enabled: boolean;
};

export type MatchValueValidation = {
  valid: boolean;
  format: string;
  message: string;
};

export type ChainValidation = {
  valid: boolean;
  chainEnabled: boolean;
  chainHops: string[];
};

export type ScopeValidation = {
  valid: boolean;
  scopeExists: boolean;
  scopeOwnerNodeId: string;
  matchesChainFinalHop: boolean;
};

export type RouteRuleValidationResult = {
  valid: boolean;
  errors: string[];
  warnings: string[];
  matchValueValidation: MatchValueValidation;
  chainValidation: ChainValidation;
  scopeValidation: ScopeValidation;
};

export type Scope = {
  id: string;
  name: string;
  description: string;
  createdAt: string;
  updatedAt: string;
};
