export type APIResponse<T> = {
  code: number;
  message: string;
  data: T;
};

export type Account = {
  id: string;
  account: string;
  role: string;
  status: string;
  mustRotatePassword: boolean;
};

export type LoginResult = {
  account: Account;
  accessToken: string;
  refreshToken: string;
  expiresAt: string;
  mustRotatePassword: boolean;
};

export type Overview = {
  nodes: {
    healthy: number;
    degraded: number;
  };
  policies: {
    activeRevision: string;
    publishedAt: string;
  };
  certificates: {
    renewSoon: number;
  };
};

export type Node = {
  id: string;
  name: string;
  mode: string;
  scopeKey: string;
  parentNodeId: string;
  enabled: boolean;
  status: string;
  publicHost?: string;
  publicPort?: number;
};

export type NodeLink = {
  id: string;
  sourceNodeId: string;
  targetNodeId: string;
  linkType: string;
  trustState: string;
};

export type NodeTransport = {
  id: string;
  nodeId: string;
  transportType: string;
  direction: string;
  address: string;
  status: string;
  parentNodeId: string;
  connectedAt: string;
  lastHeartbeatAt: string;
  latencyMs: number;
  details: Record<string, string>;
};

export type NodeAccessPath = {
  id: string;
  name: string;
  mode: string;
  targetNodeId: string;
  entryNodeId: string;
  relayNodeIds: string[];
  targetHost: string;
  targetPort: number;
  enabled: boolean;
};

export type NodeOnboardingTask = {
  id: string;
  mode: string;
  pathId: string;
  targetNodeId: string;
  targetHost: string;
  targetPort: number;
  status: string;
  statusMessage: string;
  requestedByAccountId?: string;
  createdAt: string;
  updatedAt: string;
};

export type Chain = {
  id: string;
  name: string;
  destinationScope: string;
  enabled: boolean;
  hops: (string | number)[];
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

export type RouteRule = {
  id: string;
  priority: number;
  matchType: string;
  matchValue: string;
  actionType: string;
  chainId?: string;
  destinationScope?: string;
  enabled: boolean;
};

export type PolicyRevision = {
  id: string;
  version: string;
  status: string;
  createdAt: string;
  assignedNodes: number;
};

export type Certificate = {
  id: string;
  ownerType: string;
  ownerId: string;
  certType: string;
  provider: string;
  status: string;
  notBefore: string;
  notAfter: string;
};

export type NodeHealth = {
  nodeId: string;
  heartbeatAt: string;
  policyRevisionId: string;
  listenerStatus: Record<string, string>;
  certStatus: Record<string, string>;
};

export type NodeHealthHistory = {
  heartbeatAt: string;
  listenerStatus: Record<string, string>;
  certStatus: Record<string, string>;
};

export type BootstrapToken = {
  id: string;
  token: string;
  targetType: string;
  targetId: string;
  expiresAt: string;
};

export type NodeEnrollmentApproval = {
  id: string;
  tokenId: string;
  targetNodeName: string;
  targetNodeType: string;
  parentNodeId: number | null;
  status: string;
  operatorNote?: string;
  approvedBy?: string;
  approvedAt?: string;
  rejectedBy?: string;
  rejectedAt?: string;
  createdAt: string;
  expiresAt: string;
};

export type ConnectedNodeResult = {
  node: Node;
  connectionStatus: string;
  localIps: string[];
  nodeListenAddr: string;
  nodeHttpsListenAddr: string;
  controlPlaneBound: boolean;
  mustRotatePassword: boolean;
};
