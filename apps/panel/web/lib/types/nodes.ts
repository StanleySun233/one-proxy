import type {
  AccessAuthMode,
  AccessProtocol,
  AccessServiceType,
  BootstrapTargetType,
  CertStatus,
  ListenerStatus,
  LinkType,
  NodeMode,
  NodeStatus,
  PathMode,
  TLSMode,
  TransportStatus,
  TransportType,
  TrustState
} from './common';
import type {ResourcePermissionMetadata} from './grants';
import type {DeleteImpactItem} from './proxy';

export type Node = ResourcePermissionMetadata & {
  id: string;
  name: string;
  mode: NodeMode;
  scopeKey: string;
  parentNodeId: string;
  enabled: boolean;
  status: NodeStatus;
  publicHost?: string;
  publicPort?: number;
  reviewedBy?: string;
  reviewedAt?: string;
  rejectReason?: string;
};

export type NodeDeleteImpact = {
  nodeId: string;
  delete: {
    node: number;
    chains: number;
    chainHops: number;
    routeRules: number;
    accessPaths: number;
    onboardingTasks: number;
    chainProbeResults: number;
    runtimeTransports: number;
    nodeLinks: number;
    policyAssignments: number;
    healthSnapshots: number;
    slaMinutes: number;
    apiTokens: number;
    trustMaterials: number;
    bootstrapTokens: number;
    tenantBindings: number;
  };
  update: {
    childNodesDetached: number;
  };
};

export type NodeLink = ResourcePermissionMetadata & {
  id: string;
  sourceNodeId: string;
  targetNodeId: string;
  linkType: LinkType;
  trustState: TrustState;
};

export type NodeAccessPath = ResourcePermissionMetadata & {
  id: string;
  chainId: string;
  name: string;
  mode: PathMode;
  protocol: AccessProtocol;
  serviceType: AccessServiceType;
  targetNodeId: string;
  entryNodeId: string;
  relayNodeIds: string[];
  listenHost: string;
  listenPort: number;
  targetProtocol: string;
  targetHost: string;
  targetPort: number;
  targetSni: string;
  tlsMode: TLSMode;
  authMode: AccessAuthMode;
  options: Record<string, string>;
  enabled: boolean;
};

export type NodeAccessPathDeleteImpact = {
  pathId: string;
  delete: {
    accessPath: DeleteImpactItem[];
    onboardingTasks: DeleteImpactItem[];
    tenantBindings: DeleteImpactItem[];
  };
};

export type NodeAccessPathPayload = Omit<NodeAccessPath, 'id' | 'enabled'> & {
  enabled?: boolean;
};

export type NodeTransport = {
  id: string;
  nodeId: string;
  transportType: TransportType;
  direction: string;
  address: string;
  status: TransportStatus;
  parentNodeId: string;
  connectedAt: string;
  lastHeartbeatAt: string;
  latencyMs: number;
  details: Record<string, string>;
};

export type NodeHealth = {
  nodeId: string;
  heartbeatAt: string;
  policyRevisionId: string;
  listenerStatus: Record<string, ListenerStatus | NodeStatus>;
  certStatus: Record<string, CertStatus>;
};

export type NodeHealthHistory = {
  heartbeatAt: string;
  listenerStatus: Record<string, ListenerStatus | NodeStatus>;
  certStatus: Record<string, CertStatus>;
};

export type NodeSLAMinute = {
  scenarioId: string;
  scenarioName: string;
  nodeId: string;
  nodeName: string;
  windowStart: string;
  expectedHeartbeats: number;
  receivedHeartbeats: number;
  success: number;
  createdAt: string;
  updatedAt: string;
};

export type BootstrapToken = {
  id: string;
  token: string;
  targetType: BootstrapTargetType;
  targetId: string;
  nodeName: string;
  nodeMode: NodeMode;
  scopeKey: string;
  parentNodeId: string;
  publicHost: string;
  publicPort: number;
  expiresAt: string;
};

export type UnconsumedBootstrapToken = {
  id: string;
  targetType: BootstrapTargetType;
  targetId: string;
  nodeName: string;
  nodeMode: NodeMode;
  scopeKey: string;
  parentNodeId: string;
  publicHost: string;
  publicPort: number;
  expiresAt: string;
  createdAt: string;
};

export type NodeParentURLProbeResult = {
  reachable: boolean;
  url: string;
  healthUrl: string;
  statusCode: number;
  mode: string;
  controlPlaneBound?: boolean;
  message: string;
};
