import { request } from './client';
import type {
  Node,
  NodeLink,
  NodeTransport,
  BootstrapToken,
  UnconsumedBootstrapToken,
  ConnectedNodeResult,
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

export function updateNodeLink(accessToken: string, linkID: string, payload: {sourceNodeId: string; targetNodeId: string; linkType: string; trustState: string}) {
  return request<NodeLink>(`/node-links/${linkID}`, {method: 'PATCH', accessToken, body: payload});
}

export function deleteNodeLink(accessToken: string, linkID: string) {
  return request<{status: string}>(`/node-links/${linkID}`, {method: 'DELETE', accessToken});
}

export function getNodeTransports(accessToken: string) {
  return request<NodeTransport[]>('/node-transports', {accessToken});
}

export function createBootstrapToken(
  accessToken: string,
  payload: {targetType: string; targetId: string; nodeName: string; nodeMode: string; scopeKey: string; parentNodeId: string; publicHost: string; publicPort: number}
) {
  return request<BootstrapToken>('/nodes/bootstrap-token', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function getUnconsumedBootstrapTokens(accessToken: string) {
  return request<UnconsumedBootstrapToken[]>('/nodes/bootstrap-tokens/unconsumed', {accessToken});
}

export function deleteBootstrapToken(accessToken: string, tokenID: string) {
  return request<{status: string}>(`/nodes/bootstrap-tokens/${tokenID}`, {
    method: 'DELETE',
    accessToken
  });
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
