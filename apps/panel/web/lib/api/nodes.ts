import { request } from './client';
import type {
  Node,
  NodeDeleteImpact,
  NodeTransport,
  BootstrapToken,
  UnconsumedBootstrapToken,
  NodeHealth,
  NodeHealthHistory,
  NodeSLAMinute,
  Overview,
} from '@/lib/types';

export function getNodes(accessToken: string, tenantId: string | null) {
  return request<Node[]>('/nodes', {accessToken, tenantId});
}

export function updateNode(
  accessToken: string,
  tenantId: string | null,
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
    tenantId,
    body: payload
  });
}

export function getNodeDeleteImpact(accessToken: string, tenantId: string | null, nodeID: string) {
  return request<NodeDeleteImpact>(`/nodes/${nodeID}/delete-impact`, {accessToken, tenantId});
}

export function deleteNode(accessToken: string, tenantId: string | null, nodeID: string) {
  return request<{status: string}>(`/nodes/${nodeID}`, {
    method: 'DELETE',
    accessToken,
    tenantId
  });
}

export function getNodeTransports(accessToken: string, tenantId: string | null) {
  return request<NodeTransport[]>('/nodes/transports', {accessToken, tenantId});
}

export function createBootstrapToken(
  accessToken: string,
  tenantId: string | null,
  payload: {targetType: string; targetId: string; nodeName: string; nodeMode: string; scopeKey: string; parentNodeId: string; publicHost: string; publicPort: number}
) {
  return request<BootstrapToken>('/nodes/bootstrap/token', {
    method: 'POST',
    accessToken,
    tenantId,
    body: payload
  });
}

export function getUnconsumedBootstrapTokens(accessToken: string, tenantId: string | null) {
  return request<UnconsumedBootstrapToken[]>('/nodes/bootstrap/tokens/unconsumed', {accessToken, tenantId});
}

export function deleteBootstrapToken(accessToken: string, tenantId: string | null, tokenID: string) {
  return request<{status: string}>(`/nodes/bootstrap/tokens/${tokenID}`, {
    method: 'DELETE',
    accessToken,
    tenantId
  });
}

export function approveNode(accessToken: string, tenantId: string | null, nodeID: string) {
  return request<{node: Node; accessToken: string; trustMaterial: string; expiresAt: string}>(`/nodes/${nodeID}/approve`, {
    method: 'POST',
    accessToken,
    tenantId
  });
}

export function getPendingNodes(accessToken: string, tenantId: string | null) {
  return request<Node[]>('/nodes/pending', {accessToken, tenantId});
}

export function rejectNode(accessToken: string, tenantId: string | null, nodeId: string, reason?: string) {
  return request<{status: string}>(`/nodes/${nodeId}/reject`, {
    method: 'POST',
    accessToken,
    tenantId,
    body: {reason: reason || ''}
  });
}

export function getOverview(accessToken: string, tenantId: string | null) {
  return request<Overview>('/overview', {accessToken, tenantId});
}

export function getNodeHealth(accessToken: string, tenantId: string | null) {
  return request<NodeHealth[]>('/nodes/health', {accessToken, tenantId});
}

export function getNodeHealthHistory(accessToken: string, tenantId: string | null, nodeId: string, window?: string) {
  const params = new URLSearchParams({nodeId});
  if (window) params.set('window', window);
  return request<NodeHealthHistory[]>(`/nodes/health/history?${params.toString()}`, {accessToken, tenantId});
}

export function getNodeSLA(accessToken: string, tenantId: string | null, window?: string) {
  const params = new URLSearchParams();
  if (window) params.set('window', window);
  const query = params.toString();
  return request<NodeSLAMinute[]>(`/nodes/sla${query ? `?${query}` : ''}`, {accessToken, tenantId});
}
