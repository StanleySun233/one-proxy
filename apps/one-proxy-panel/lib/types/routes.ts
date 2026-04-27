export type RouteRule = {
  id: string;
  priority: number;
  matchType: string;
  matchValue: string;
  actionType: string;
  chainId?: string;
  destinationScope?: string;
  enabled: boolean;
};

export type MatchValueValidation = {
  valid: boolean;
  format: string;
  message: string;
};

export type ChainValidation = {
  valid: boolean;
  chainEnabled: boolean;
  chainHops: string[];
};

export type ScopeValidation = {
  valid: boolean;
  scopeExists: boolean;
  scopeOwnerNodeId: string;
  matchesChainFinalHop: boolean;
};

export type RouteRuleValidationResult = {
  valid: boolean;
  errors: string[];
  warnings: string[];
  matchValueValidation: MatchValueValidation;
  chainValidation: ChainValidation;
  scopeValidation: ScopeValidation;
};
