export type ResourceBindingType = 'node' | 'node_link' | 'scope' | 'chain' | 'route_rule' | 'access_path';

export type ResourceBindingPermission = 'use' | 'manage';

export type ResourceBinding = {
  tenantId: string;
  tenantName: string;
  resourceType: ResourceBindingType;
  resourceId: string;
  permission: ResourceBindingPermission;
  createdAt: string;
};

export type ResourceBindingPayload = {
  permission?: ResourceBindingPermission;
};

export type ResourcePermissionMetadata = {
  permission?: ResourceBindingPermission;
};
