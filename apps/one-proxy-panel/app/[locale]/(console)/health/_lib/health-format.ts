import {FieldEnumMap} from '@/lib/types';

export function joinMap(value: Record<string, string>) {
  return Object.entries(value || {})
    .map(([key, item]) => `${key}:${item}`)
    .join(', ');
}

export function healthBadgeClassName(status: string, enums: FieldEnumMap | undefined): string {
  const entry = enums?.node_status?.[status];
  if (entry?.meta?.className) {
    return `badge ${entry.meta.className}`;
  }
  return 'badge is-neutral';
}
