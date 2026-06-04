import type {TenantMembershipRole} from './auth';

export type Tenant = {
  id: string;
  name: string;
  createdAt: string;
  updatedAt: string;
};

export type TenantMembershipAccount = {
  accountId: string;
  account: string;
  tenantId: string;
  tenantName: string;
  role: TenantMembershipRole;
  joinedAt: string;
};

export type TenantCreatedResult = {
  tenant: Tenant;
  membership: TenantMembershipAccount;
};
