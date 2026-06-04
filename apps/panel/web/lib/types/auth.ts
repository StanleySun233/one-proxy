import type {AccountRole, AccountStatus} from './common';

export type Account = {
  id: string;
  account: string;
  role: AccountRole;
  status: AccountStatus;
  mustRotatePassword: boolean;
};

export type TenantMembershipRole = 'tenant_admin' | 'user';

export type TenantMembership = {
  tenantId: string;
  tenantName: string;
  role: TenantMembershipRole;
  joinedAt: string;
};

export type ActiveTenant = TenantMembership | null;

export type LoginResult = {
  account: Account;
  accessToken: string;
  refreshToken: string;
  expiresAt: string;
  mustRotatePassword: boolean;
  tenantMemberships: TenantMembership[];
  activeTenantId: string | null;
};
