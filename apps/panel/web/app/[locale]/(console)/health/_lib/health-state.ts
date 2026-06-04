import {FieldEnumMap, Node, NodeHealth, NodeHealthHistory} from '@/lib/types';

import {joinMap} from './health-format';

const staleThresholdMs = 2 * 60 * 1000;

export type HealthRow = NodeHealth & {
  name: string;
  mode: string;
  scopeKey: string;
  parentNodeId: string;
  derivedStatus: string;
  derivedLabel: string;
  listenerSummary: string;
  certSummary: string;
};

export function buildHealthRows(nodes: Node[], health: NodeHealth[], enums: FieldEnumMap | undefined): HealthRow[] {
  const healthByNodeID = new Map(health.map((item) => [item.nodeId, item]));
  return nodes.map((node) => {
    const item = healthByNodeID.get(node.id);
    if (!item) {
      return {
        nodeId: node.id,
        heartbeatAt: '',
        policyRevisionId: '',
        listenerStatus: {},
        certStatus: {},
        name: node.name,
        mode: node.mode,
        scopeKey: node.scopeKey,
        parentNodeId: node.parentNodeId,
        derivedStatus: 'unreported',
        derivedLabel: 'unreported',
        listenerSummary: '',
        certSummary: ''
      };
    }
    const derived = deriveHealthState(item, enums);
    return {
      ...item,
      name: node.name,
      mode: node.mode,
      scopeKey: node.scopeKey,
      parentNodeId: node.parentNodeId,
      derivedStatus: derived.status,
      derivedLabel: derived.label,
      listenerSummary: joinMap(item.listenerStatus),
      certSummary: joinMap(item.certStatus)
    };
  });
}

export function deriveHealthState(
  item: {heartbeatAt: string; listenerStatus: Record<string, string>; certStatus: Record<string, string>},
  enums: FieldEnumMap | undefined
) {
  const heartbeatTime = Date.parse(item.heartbeatAt);
  const isStale = Number.isFinite(heartbeatTime) ? Date.now() - heartbeatTime > staleThresholdMs : true;
  const listenerValues = Object.values(item.listenerStatus || {});
  const certValues = Object.values(item.certStatus || {});
  const hasDegradedSignal = [...listenerValues, ...certValues].some(
    (value) => !isGoodEnumValue('listener_status', value, enums) && !isGoodEnumValue('cert_status', value, enums)
  );
  if (isStale) {
    return {status: 'stale', label: enums?.node_status?.stale?.name || 'stale'};
  }
  if (hasDegradedSignal) {
    return {status: 'degraded', label: enums?.node_status?.degraded?.name || 'degraded'};
  }
  return {status: 'healthy', label: enums?.node_status?.healthy?.name || 'healthy'};
}

export function deriveTrendState(item: NodeHealthHistory, enums: FieldEnumMap | undefined): string {
  const listenerValues = Object.values(item.listenerStatus || {});
  const certValues = Object.values(item.certStatus || {});
  const hasDegradedSignal = [...listenerValues, ...certValues].some(
    (value) => !isGoodEnumValue('listener_status', value, enums) && !isGoodEnumValue('cert_status', value, enums)
  );
  return hasDegradedSignal ? 'degraded' : 'healthy';
}

function isGoodEnumValue(statusField: string, value: string, enums: FieldEnumMap | undefined): boolean {
  const entry = enums?.[statusField]?.[value];
  if (entry) {
    return entry.meta?.className === 'is-good';
  }
  return value === 'up' || value === 'healthy' || value === 'renewed';
}
