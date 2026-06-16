import {Chain, RouteRule} from '@/lib/types';

export type RouteRuleFormValues = {
  groupId: string;
  priority: string;
  matchType: string;
  matchValue: string;
  actionType: string;
  chainId: string;
  destinationScope: string;
  enabled: boolean;
};

export type RouteRuleValidationPayload = {
  ruleId?: string;
  groupId: string;
  priority: number;
  matchType: string;
  matchValue: string;
  actionType: string;
  chainId: string;
  destinationScope: string;
};

export type RouteRuleSubmitPayload = Omit<RouteRuleValidationPayload, 'ruleId'> & {
  enabled: boolean;
};

export function defaultRouteRuleFormValues(): RouteRuleFormValues {
  return {
    groupId: '',
    priority: '100',
    matchType: 'domain',
    matchValue: '',
    actionType: 'chain',
    chainId: '',
    destinationScope: '',
    enabled: true
  };
}

export function routeRuleFormValues(rule: RouteRule): RouteRuleFormValues {
  return {
    groupId: rule.groupId,
    priority: String(rule.priority),
    matchType: rule.matchType,
    matchValue: rule.matchValue,
    actionType: rule.actionType,
    chainId: rule.chainId || '',
    destinationScope: rule.destinationScope || '',
    enabled: rule.enabled
  };
}

export function routeRuleValidationPayload(values: RouteRuleFormValues, ruleId?: string): RouteRuleValidationPayload {
  return {
    ruleId,
    groupId: values.groupId.trim(),
    priority: Number(values.priority) || 0,
    matchType: values.matchType,
    matchValue: values.matchValue.trim(),
    actionType: values.actionType,
    chainId: values.chainId.trim(),
    destinationScope: values.destinationScope.trim()
  };
}

export function routeRuleSubmitPayload(values: RouteRuleFormValues, chains: Chain[]): RouteRuleSubmitPayload {
  const chainDestinationScope = values.actionType === 'chain'
    ? chains.find((chain) => chain.id === values.chainId)?.destinationScope || ''
    : values.destinationScope.trim();

  return {
    priority: Number(values.priority),
    groupId: values.groupId.trim(),
    matchType: values.matchType.trim(),
    matchValue: values.matchValue.trim(),
    actionType: values.actionType,
    chainId: values.chainId.trim(),
    destinationScope: chainDestinationScope,
    enabled: values.enabled
  };
}
