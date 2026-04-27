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
