import { request } from './client';
import type {
  Node,
  NodeLink,
  NodeTransport,
  BootstrapToken,
  UnconsumedBootstrapToken,
  ConnectedNodeResult,
  NodeAccessPath,
  NodeOnboardingTask,
  NodeHealth,
  NodeHealthHistory,
  Overview,
} from '@/lib/types';

export function getNodes(accessToken: string) {
  return request<Node[]>('/nodes', {accessToken});
}

export function createNode(
  accessToken: string,
  payload: {name: string; mode: string; scopeKey: string; parentNodeId: string; publicHost: string; publicPort: number}
) {
  return request<Node>('/nodes', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function updateNode(
  accessToken: string,
  nodeID: string,
  payload: {
    name: string;
    mode: string;
    scopeKey: string;
    parentNodeId: string;
    publicHost: string;
    publicPort: number;
    enabled: boolean;
    status: string;
  }
) {
  return request<Node>(`/nodes/${nodeID}`, {
    method: 'PATCH',
    accessToken,
    body: payload
  });
}

export function deleteNode(accessToken: string, nodeID: string) {
  return request<{status: string}>(`/nodes/${nodeID}`, {
    method: 'DELETE',
    accessToken
  });
}

export function connectNode(
  accessToken: string,
  payload: {
    address: string;
    password: string;
    newPassword: string;
    name: string;
    mode: string;
    scopeKey: string;
    parentNodeId: string;
    publicHost: string;
    publicPort: number;
    controlPlaneUrl: string;
  }
) {
  return request<ConnectedNodeResult>('/nodes/connect', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function getNodeLinks(accessToken: string) {
  return request<NodeLink[]>('/node-links', {accessToken});
}

export function createNodeLink(accessToken: string, payload: {sourceNodeId: string; targetNodeId: string; linkType: string; trustState: string}) {
  return request<NodeLink>('/node-links', {method: 'POST', accessToken, body: payload});
}

export function getNodeTransports(accessToken: string) {
  return request<NodeTransport[]>('/node-transports', {accessToken});
}

export function createBootstrapToken(accessToken: string, payload: {targetType: string; targetId: string; nodeName: string}) {
  return request<BootstrapToken>('/nodes/bootstrap-token', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function getUnconsumedBootstrapTokens(accessToken: string) {
  return request<UnconsumedBootstrapToken[]>('/nodes/bootstrap-tokens/unconsumed', {accessToken});
}

export function approveNode(accessToken: string, nodeID: string) {
  return request<{node: Node; accessToken: string; trustMaterial: string; expiresAt: string}>(`/nodes/approve/${nodeID}`, {
    method: 'POST',
    accessToken
  });
}

export function getPendingNodes(accessToken: string) {
  return request<Node[]>('/nodes/pending', {accessToken});
}

export function rejectNode(accessToken: string, nodeId: string, reason?: string) {
  return request<{status: string}>(`/nodes/${nodeId}/reject`, {
    method: 'POST',
    accessToken,
    body: {reason: reason || ''}
  });
}

export function getNodeAccessPaths(accessToken: string) {
  return request<NodeAccessPath[]>('/node-access-paths', {accessToken});
}

export function createNodeAccessPath(
  accessToken: string,
  payload: {
    name: string;
    mode: string;
    targetNodeId: string;
    entryNodeId: string;
    relayNodeIds: string[];
    targetHost: string;
    targetPort: number;
  }
) {
  return request<NodeAccessPath>('/node-access-paths', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function updateNodeAccessPath(
  accessToken: string,
  pathID: string,
  payload: {
    name: string;
    mode: string;
    targetNodeId: string;
    entryNodeId: string;
    relayNodeIds: string[];
    targetHost: string;
    targetPort: number;
    enabled: boolean;
  }
) {
  return request<NodeAccessPath>(`/node-access-paths/${pathID}`, {
    method: 'PATCH',
    accessToken,
    body: payload
  });
}

export function deleteNodeAccessPath(accessToken: string, pathID: string) {
  return request<{status: string}>(`/node-access-paths/${pathID}`, {
    method: 'DELETE',
    accessToken
  });
}

export function getNodeOnboardingTasks(accessToken: string) {
  return request<NodeOnboardingTask[]>('/node-onboarding-tasks', {accessToken});
}

export function createNodeOnboardingTask(
  accessToken: string,
  payload: {
    mode: string;
    pathId: string;
    targetNodeId: string;
    targetHost: string;
    targetPort: number;
  }
) {
  return request<NodeOnboardingTask>('/node-onboarding-tasks', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function updateNodeOnboardingTaskStatus(
  accessToken: string,
  taskID: string,
  payload: {
    status: string;
    statusMessage: string;
  }
) {
  return request<NodeOnboardingTask>(`/node-onboarding-tasks/${taskID}`, {
    method: 'PATCH',
    accessToken,
    body: payload
  });
}

export function getOverview(accessToken: string) {
  return request<Overview>('/overview', {accessToken});
}

export function getNodeHealth(accessToken: string) {
  return request<NodeHealth[]>('/nodes/health', {accessToken});
}

export function getNodeHealthHistory(accessToken: string, nodeId: string, window?: string) {
  const params = new URLSearchParams({nodeId});
  if (window) params.set('window', window);
  return request<NodeHealthHistory[]>(`/nodes/health/history?${params.toString()}`, {accessToken});
}
