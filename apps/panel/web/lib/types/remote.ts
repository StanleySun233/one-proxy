export type RemoteProtocol = 'ssh' | 'rdp';
export type RemoteCredentialScope = 'personal' | 'tenant';
export type RemoteSecretType = 'password' | 'private_key';

export type RemoteCredential = {
  id: string;
  tenantId?: string;
  accountId: string;
  name: string;
  protocol: RemoteProtocol;
  scope: RemoteCredentialScope;
  username: string;
  secretType: RemoteSecretType;
  encryptedPayload: string;
  createdAt: string;
  updatedAt: string;
  lastUsedAt?: string;
};

export type RemoteCredentialPayload = {
  name: string;
  protocol: RemoteProtocol;
  scope: RemoteCredentialScope;
  username: string;
  secretType: RemoteSecretType;
  encryptedPayload: string;
};

export type RemoteCredentialUpdatePayload = {
  name: string;
  username: string;
  secretType: RemoteSecretType;
  encryptedPayload: string;
};

export type RemoteSessionPayload = {
  accessPathId: string;
  credentialId?: string;
  protocol: RemoteProtocol;
  username: string;
  password?: string;
  privateKey?: string;
  passphrase?: string;
  width: number;
  height: number;
  dpi: number;
};

export type RemoteSession = {
  id: string;
  token: string;
  protocol: RemoteProtocol;
  expiresAt: string;
  tunnelUrl: string;
};

export type RemoteSecret = {
  password: string;
  privateKey: string;
  passphrase: string;
};
