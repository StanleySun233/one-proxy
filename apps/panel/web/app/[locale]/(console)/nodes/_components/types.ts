'use client';

import {Node} from '@/lib/types';

export type BootstrapFormValues = {
  targetId: string;
  nodeName: string;
  nodeMode: string;
  scopeKey: string;
  parentNodeId: string;
  parentReachableUrl: string;
  publicHost: string;
  publicPort: string;
};

export type RegistryNodeRow = Node & {
  derivedHealthStatus: string;
  derivedHealthLabel: string;
  heartbeatAt: string;
  policyRevisionId: string;
};

export type RegistryNodeFormState = {
  name: string;
  mode: string;
  scopeKey: string;
  parentNodeId: string;
  publicHost: string;
  publicPort: string;
  enabled: boolean;
  status: string;
};
