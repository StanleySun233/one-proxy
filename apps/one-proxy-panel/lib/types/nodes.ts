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
  reviewedBy?: string;
  reviewedAt?: string;
  rejectReason?: string;
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

export type UnconsumedBootstrapToken = {
  id: string;
  targetType: string;
  targetId: string;
  nodeName: string;
  expiresAt: string;
  createdAt: string;
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
