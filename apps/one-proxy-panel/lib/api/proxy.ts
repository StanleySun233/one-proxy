import { request } from './client';
import type { Chain, ChainProbeResult, ChainValidationResult, ChainPreviewResult, RouteRule, RouteRuleValidationResult, Scope } from '@/lib/types/proxy';
import type { NodeAccessPath, NodeAccessPathPayload, NodeLink } from '@/lib/types/nodes';

export function getChains(accessToken: string, tenantId: string | null) {
  return request<Chain[]>('/proxy', {accessToken, tenantId});
}

export function createChain(accessToken: string, tenantId: string | null, payload: {name: string; destinationScope: string; hops: string[]}) {
  return request<Chain>('/proxy', {
    method: 'POST',
    accessToken,
    tenantId,
    body: payload
  });
}

export function updateChain(accessToken: string, tenantId: string | null, chainID: string, payload: {name: string; destinationScope: string; hops: string[]; enabled: boolean}) {
  return request<Chain>(`/proxy/${chainID}`, {
    method: 'PATCH',
    accessToken,
    tenantId,
    body: payload
  });
}

export function probeChain(accessToken: string, tenantId: string | null, chainID: string) {
  return request<ChainProbeResult>(`/proxy/${chainID}/probe`, {
    method: 'POST',
    accessToken,
    tenantId
  });
}

export function validateChain(accessToken: string, tenantId: string | null, payload: {name: string; destinationScope: string; hops: string[]}) {
  return request<ChainValidationResult>('/proxy/validate', {
    method: 'POST',
    accessToken,
    tenantId,
    body: payload
  });
}

export function previewChain(accessToken: string, tenantId: string | null, payload: {name: string; destinationScope: string; hops: string[]}) {
  return request<ChainPreviewResult>('/proxy/preview', {
    method: 'POST',
    accessToken,
    tenantId,
    body: payload
  });
}

export function getNodeLinks(accessToken: string, tenantId: string | null) {
  return request<NodeLink[]>('/proxy/node-links', {accessToken, tenantId});
}

export function createNodeLink(accessToken: string, tenantId: string | null, payload: {sourceNodeId: string; targetNodeId: string; linkType: string; trustState: string}) {
  return request<NodeLink>('/proxy/node-links', {method: 'POST', accessToken, tenantId, body: payload});
}

export function updateNodeLink(accessToken: string, tenantId: string | null, linkID: string, payload: {sourceNodeId: string; targetNodeId: string; linkType: string; trustState: string}) {
  return request<NodeLink>(`/proxy/node-links/${linkID}`, {method: 'PATCH', accessToken, tenantId, body: payload});
}

export function deleteNodeLink(accessToken: string, tenantId: string | null, linkID: string) {
  return request<{status: string}>(`/proxy/node-links/${linkID}`, {method: 'DELETE', accessToken, tenantId});
}

export function getNodeAccessPaths(accessToken: string, tenantId: string | null) {
  return request<NodeAccessPath[]>('/proxy/access-paths', {accessToken, tenantId});
}

export function createNodeAccessPath(accessToken: string, tenantId: string | null, payload: NodeAccessPathPayload) {
  return request<NodeAccessPath>('/proxy/access-paths', {method: 'POST', accessToken, tenantId, body: payload});
}

export function updateNodeAccessPath(accessToken: string, tenantId: string | null, pathID: string, payload: NodeAccessPathPayload & {enabled: boolean}) {
  return request<NodeAccessPath>(`/proxy/access-paths/${pathID}`, {method: 'PATCH', accessToken, tenantId, body: payload});
}

export function deleteNodeAccessPath(accessToken: string, tenantId: string | null, pathID: string) {
  return request<{status: string}>(`/proxy/access-paths/${pathID}`, {method: 'DELETE', accessToken, tenantId});
}

export function getRouteRules(accessToken: string, tenantId: string | null) {
  return request<RouteRule[]>('/proxy/routes', {accessToken, tenantId});
}

export function createRouteRule(
  accessToken: string,
  tenantId: string | null,
  payload: {
    priority: number;
    matchType: string;
    matchValue: string;
    actionType: string;
    chainId: string;
    destinationScope: string;
  }
) {
  return request<RouteRule>('/proxy/routes', {
    method: 'POST',
    accessToken,
    tenantId,
    body: payload
  });
}

export function updateRouteRule(
  accessToken: string,
  tenantId: string | null,
  ruleId: string,
  payload: {
    priority: number;
    matchType: string;
    matchValue: string;
    actionType: string;
    chainId: string;
    destinationScope: string;
    enabled: boolean;
  }
) {
  return request<RouteRule>(`/proxy/routes/${ruleId}`, {
    method: 'PATCH',
    accessToken,
    tenantId,
    body: payload
  });
}

export function deleteRouteRule(accessToken: string, tenantId: string | null, ruleId: string) {
  return request<{status: string}>(`/proxy/routes/${ruleId}`, {
    method: 'DELETE',
    accessToken,
    tenantId
  });
}

export function validateRouteRule(accessToken: string, tenantId: string | null, payload: {
  priority: number;
  matchType: string;
  matchValue: string;
  actionType: string;
  chainId: string;
  destinationScope: string;
}) {
  return request<RouteRuleValidationResult>('/proxy/routes/validate', {
    method: 'POST',
    accessToken,
    tenantId,
    body: payload
  });
}

export function getScopes(accessToken: string, tenantId: string | null) {
  return request<Scope[]>('/proxy/scopes', {accessToken, tenantId});
}

export function createScope(accessToken: string, tenantId: string | null, payload: {name: string; description: string}) {
  return request<Scope>('/proxy/scopes', {
    accessToken,
    tenantId,
    method: 'POST',
    body: payload
  });
}

export function updateScope(accessToken: string, tenantId: string | null, scopeID: string, payload: {name: string; description: string}) {
  return request<Scope>(`/proxy/scopes/${scopeID}`, {
    accessToken,
    tenantId,
    method: 'PATCH',
    body: payload
  });
}

export function deleteScope(accessToken: string, tenantId: string | null, scopeID: string) {
  return request<{status: string}>(`/proxy/scopes/${scopeID}`, {
    accessToken,
    tenantId,
    method: 'DELETE'
  });
}
