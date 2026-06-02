import { request } from './client';
import type { Chain, ChainProbeResult, ChainValidationResult, ChainPreviewResult, RouteRule, RouteRuleValidationResult, Scope } from '@/lib/types/chains';

export function getChains(accessToken: string) {
  return request<Chain[]>('/chains', {accessToken});
}

export function createChain(accessToken: string, payload: {name: string; destinationScope: string; hops: string[]}) {
  return request<Chain>('/chains', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function updateChain(accessToken: string, chainID: string, payload: {name: string; destinationScope: string; hops: string[]; enabled: boolean}) {
  return request<Chain>(`/chains/${chainID}`, {
    method: 'PATCH',
    accessToken,
    body: payload
  });
}

export function probeChain(accessToken: string, chainID: string) {
  return request<ChainProbeResult>(`/chains/${chainID}/probe`, {
    method: 'POST',
    accessToken
  });
}

export function validateChain(accessToken: string, payload: {name: string; destinationScope: string; hops: string[]}) {
  return request<ChainValidationResult>('/chains/validate', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function previewChain(accessToken: string, payload: {name: string; destinationScope: string; hops: string[]}) {
  return request<ChainPreviewResult>('/chains/preview', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function getRouteRules(accessToken: string) {
  return request<RouteRule[]>('/chains/routes', {accessToken});
}

export function createRouteRule(
  accessToken: string,
  payload: {
    priority: number;
    matchType: string;
    matchValue: string;
    actionType: string;
    chainId: string;
    destinationScope: string;
  }
) {
  return request<RouteRule>('/chains/routes', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function updateRouteRule(
  accessToken: string,
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
  return request<RouteRule>(`/chains/routes/${ruleId}`, {
    method: 'PATCH',
    accessToken,
    body: payload
  });
}

export function deleteRouteRule(accessToken: string, ruleId: string) {
  return request<{status: string}>(`/chains/routes/${ruleId}`, {
    method: 'DELETE',
    accessToken
  });
}

export function validateRouteRule(accessToken: string, payload: {
  priority: number;
  matchType: string;
  matchValue: string;
  actionType: string;
  chainId: string;
  destinationScope: string;
}) {
  return request<RouteRuleValidationResult>('/chains/routes/validate', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function getScopes(accessToken: string) {
  return request<Scope[]>('/chains/scopes', {accessToken});
}

export function createScope(accessToken: string, payload: {name: string; description: string}) {
  return request<Scope>('/chains/scopes', {
    accessToken,
    method: 'POST',
    body: payload
  });
}

export function updateScope(accessToken: string, scopeID: string, payload: {name: string; description: string}) {
  return request<Scope>(`/chains/scopes/${scopeID}`, {
    accessToken,
    method: 'PATCH',
    body: payload
  });
}

export function deleteScope(accessToken: string, scopeID: string) {
  return request<{status: string}>(`/chains/scopes/${scopeID}`, {
    accessToken,
    method: 'DELETE'
  });
}
