export type AuditOutcome = 'success' | 'failure' | 'denied';
export type AuditDecision = 'allow' | 'deny';

export type AuditQuery = {
  tenantId?: string;
  actorId?: string;
  actorType?: string;
  resourceType?: string;
  resourceId?: string;
  action?: string;
  outcome?: string;
  from?: string;
  to?: string;
  limit?: number;
};

export type NetworkAuditQuery = {
  tenantId?: string;
  actorId?: string;
  tokenId?: string;
  nodeId?: string;
  targetHost?: string;
  routeId?: string;
  scopeId?: string;
  chainId?: string;
  decision?: string;
  from?: string;
  to?: string;
  limit?: number;
};

export type AuditEvent = {
  id: string;
  tenantId: string;
  occurredAt: string;
  actorType: string;
  actorId: string;
  actorName?: string;
  actorIp?: string;
  actorAgent?: string;
  action: string;
  resourceType: string;
  resourceId: string;
  resourceName?: string;
  outcome: AuditOutcome | string;
  reason?: string;
  requestId?: string;
  beforeJson?: unknown;
  afterJson?: unknown;
  metadataJson?: unknown;
};

export type NetworkSession = {
  id: string;
  tenantId: string;
  startedAt: string;
  endedAt: string;
  actorType: string;
  actorId: string;
  tokenId?: string;
  sourceIp?: string;
  entryNodeId?: string;
  exitNodeId?: string;
  targetHost: string;
  targetPort: number;
  scheme?: string;
  method?: string;
  routeId?: string;
  scopeId?: string;
  chainId?: string;
  decision: AuditDecision | string;
  denyReason?: string;
  bytesIn: number;
  bytesOut: number;
  durationMs: number;
  statusCode?: number;
  errorCode?: string;
  metadataJson?: unknown;
};

export type AuditGroup = {
  id: string;
  name: string;
  count?: number;
  sessions?: number;
  success?: number;
  failure?: number;
  denied?: number;
  allowed?: number;
  bytesIn?: number;
  bytesOut?: number;
  durationMs?: number;
};

export type AuditDecisionCount = {
  decision: string;
  count: number;
};

export type AuditBusinessSummary = {
  total: number;
  outcomeCount: Record<string, number>;
  actionCount: Record<string, number>;
  resourceType: Record<string, number>;
  actorCount: {
    actorType: string;
    actorId: string;
    actorName: string;
    count: number;
  }[];
};

export type NetworkAuditSummary = {
  total: number;
  bytesIn: number;
  bytesOut: number;
  durationAvgMs: number;
  decisionCount: Record<string, number>;
  topTargets: {
    targetHost: string;
    bytesIn: number;
    bytesOut: number;
    count: number;
  }[];
  userTraffic: {
    actorId: string;
    bytesIn: number;
    bytesOut: number;
    count: number;
  }[];
  nodeTraffic: {
    nodeId: string;
    bytesIn: number;
    bytesOut: number;
    count: number;
  }[];
  tenantTraffic: {
    tenantId: string;
    bytesIn: number;
    bytesOut: number;
    count: number;
  }[];
  recentBusinessEvents?: AuditEvent[];
};

export type AuditBusinessEventsResult = {
  items: AuditEvent[];
  summary: AuditBusinessSummary;
};

export type NetworkAuditSessionsResult = {
  items: NetworkSession[];
  summary: NetworkAuditSummary;
};

export type AuditDashboard = {
  total: number;
  bytesIn: number;
  bytesOut: number;
  durationAvgMs: number;
  decisionCount: Record<string, number>;
  topTargets: NetworkAuditSummary['topTargets'];
  userTraffic: NetworkAuditSummary['userTraffic'];
  nodeTraffic: NetworkAuditSummary['nodeTraffic'];
  tenantTraffic: NetworkAuditSummary['tenantTraffic'];
  recentBusinessEvents: AuditEvent[];
};
