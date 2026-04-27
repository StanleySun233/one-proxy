import { request } from './client';
import type { RouteRule, RouteRuleValidationResult } from '@/lib/types';

export function getRouteRules(accessToken: string) {
  return request<RouteRule[]>('/route-rules', {accessToken});
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
  return request<RouteRule>('/route-rules', {
    method: 'POST',
    accessToken,
    body: payload
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
  return request<RouteRuleValidationResult>('/route-rules/validate', {
    method: 'POST',
    accessToken,
    body: payload
  });
}
