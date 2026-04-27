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
  transportType: string;
  address: string;
  status: string;
};

export type ChainProbeResult = {
  chainId: string;
  status: string;
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
  mode: string;
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
