export type APIResponse<T> = {
  code: number;
  message: string;
  data: T;
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

export type FieldEnumEntry = {
  name: string;
  meta?: {
    color?: string;
    className?: string;
  };
};

export type FieldEnumMap = Record<string, Record<string, FieldEnumEntry>>;

export type NodeMode = 'edge' | 'relay';
export type NodeStatus = 'healthy' | 'degraded' | 'pending' | 'inactive';
export type AccountRole = 'super_admin';
export type AccountStatus = 'active' | 'disabled';
export type PathMode = 'direct' | 'relay_chain' | 'upstream_pull';
export type TaskStatus = 'planned' | 'pending' | 'connected' | 'failed' | 'cancelled';
export type ActionType = 'chain' | 'direct';
export type LinkType = 'parent_child' | 'relay' | 'managed';
export type TrustState = 'trusted' | 'active';
export type TransportType = 'public_http' | 'public_https' | 'reverse_ws_parent' | 'child_ws' | 'reverse_ws';
export type TransportStatus = 'connected' | 'available' | 'degraded' | 'failed' | 'pending';
export type CertStatus = 'healthy' | 'degraded' | 'renew-soon' | 'expired' | 'renewed';
export type CertType = 'public' | 'internal';
export type BootstrapTargetType = 'node';
export type TrustMaterialStatus = 'active' | 'rotated' | 'pending' | 'consumed';
export type ProbeResultStatus = 'connected' | 'failed';
export type PolicyStatus = 'published';
export type ListenerStatus = 'up' | 'degraded';
export type ApprovalState = 'pending' | 'approved' | 'rejected';
export type MatchType = 'domain' | 'domain_suffix' | 'ip_cidr' | 'ip_range' | 'port' | 'url_regex' | 'default';
export type DerivedHealthStatus = 'healthy' | 'degraded' | 'stale' | 'unreported';

export type SetupStatus = {
  configured: boolean;
};

export type TestConnectionResult = {
  success: boolean;
  message: string;
  exists?: boolean;
};

export type GenerateKeyResult = {
  key: string;
};

export type InitResult = {
  success: boolean;
  message: string;
};

export type TestConnectionRequest = {
  host: string;
  port: number;
  user: string;
  password: string;
  database: string;
};

export type InitRequest = {
  host: string;
  port: number;
  user: string;
  password: string;
  database: string;
  jwtSigningKey: string;
  adminPassword: string;
  needInitialize: boolean;
};

export type PolicyRevision = {
  id: string;
  version: string;
  status: PolicyStatus;
  createdAt: string;
  assignedNodes: number;
};

export type Certificate = {
  id: string;
  ownerType: string;
  ownerId: string;
  certType: CertType;
  provider: string;
  status: CertStatus;
  notBefore: string;
  notAfter: string;
};
