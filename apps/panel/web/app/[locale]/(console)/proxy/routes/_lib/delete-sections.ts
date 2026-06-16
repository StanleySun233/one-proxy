import {DeleteImpactSection} from '@/components/delete-confirmation-modal';
import {RouteRule, RouteRuleGroup, RouteRuleGroupDeleteImpact} from '@/lib/types';

type RoutesT = (key: string) => string;

export function routeRuleDeleteSections(rule: RouteRule | null, routesT: RoutesT): DeleteImpactSection[] {
  return rule ? [
    {
      id: 'routeRule',
      label: routesT('deleteImpactRouteRule'),
      items: [{
        id: rule.id,
        name: rule.matchValue || String(rule.priority),
        detail: `${rule.matchType} / ${rule.actionType}`
      }]
    }
  ] : [];
}

export function routeRuleGroupDeleteSections(group: RouteRuleGroup | null, impact: RouteRuleGroupDeleteImpact | null, routesT: RoutesT): DeleteImpactSection[] {
  if (impact) {
    return [
      {id: 'routeGroup', label: routesT('deleteImpactRouteGroup'), items: impact.delete.group},
      {id: 'routeRules', label: routesT('deleteImpactRouteRules'), items: impact.delete.routeRules},
      {id: 'tenantBindings', label: routesT('deleteImpactTenantBindings'), items: impact.delete.tenantBindings}
    ];
  }
  return group ? [
    {id: 'routeGroup', label: routesT('deleteImpactRouteGroup'), count: 1}
  ] : [];
}
