import {FieldEnumMap, Node, NodeHealth} from '@/lib/types';

export const staleThresholdMs = 2 * 60 * 1000;

export function describeNodeName(nodeID: string, nodesByID: Map<string, Node>) {
  if (!nodeID) {
    return '';
  }
  return nodesByID.get(nodeID)?.name || nodeID;
}

export function deriveNodeHealthState(item?: NodeHealth, enums?: FieldEnumMap) {
  if (!item) {
    return {status: 'unreported', label: 'unreported'};
  }
  const heartbeatTime = Date.parse(item.heartbeatAt);
  const isStale = Number.isFinite(heartbeatTime) ? Date.now() - heartbeatTime > staleThresholdMs : true;
  const listenerValues = Object.values(item.listenerStatus || {});
  const certValues = Object.values(item.certStatus || {});
  const allValues = [...listenerValues, ...certValues];
  const isGoodValue = (value: string) =>
    enums?.listener_status?.[value]?.meta?.className === 'is-good' ||
    enums?.cert_status?.[value]?.meta?.className === 'is-good';
  const hasDegradedSignal = enums
    ? allValues.some(v => !isGoodValue(v))
    : allValues.some((value) => value !== 'up' && value !== 'healthy' && value !== 'renewed');
  if (isStale) {
    return {status: 'stale', label: 'stale'};
  }
  if (hasDegradedSignal) {
    return {status: 'degraded', label: 'degraded'};
  }
  return {status: 'healthy', label: 'healthy'};
}

export function healthBadgeClassName(status: string, enums?: FieldEnumMap) {
  if (!enums) {
    if (status === 'healthy') return 'badge is-good';
    if (status === 'stale') return 'badge is-warn';
    if (status === 'unreported') return 'badge is-neutral';
    return 'badge is-danger';
  }
  const cls = enums.node_status?.[status]?.meta?.className;
  if (cls) return `badge ${cls}`;
  if (status === 'stale') return 'badge is-warn';
  if (status === 'unreported') return 'badge is-neutral';
  return 'badge is-danger';
}

export function statusBadgeClassName(status: string, enums?: FieldEnumMap) {
  if (!enums) {
    if (status === 'healthy') return 'badge is-good';
    if (status === 'degraded') return 'badge is-danger';
    if (status === 'pending') return 'badge is-warn';
    return 'badge is-neutral';
  }
  const cls = enums.node_status?.[status]?.meta?.className;
  return `badge ${cls || 'is-neutral'}`;
}

export function transportBadgeClassName(status: string, enums?: FieldEnumMap) {
  if (!enums) {
    if (status === 'connected' || status === 'ready') return 'badge is-good';
    if (status === 'degraded' || status === 'failed') return 'badge is-danger';
    if (status === 'pending') return 'badge is-warn';
    return 'badge is-neutral';
  }
  const cls = enums.transport_status?.[status]?.meta?.className;
  return `badge ${cls || 'is-neutral'}`;
}
