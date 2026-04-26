'use client';

export type NodeFormValues = {
  name: string;
  mode: string;
  scopeKey: string;
  parentNodeId: string;
  publicHost: string;
  publicPort: string;
};

export type QuickConnectFormValues = {
  address: string;
  password: string;
  name: string;
  mode: string;
  scopeKey: string;
  parentNodeId: string;
  publicHost: string;
  publicPort: string;
  controlPlaneUrl: string;
};

export type BootstrapFormValues = {
  targetId: string;
};
