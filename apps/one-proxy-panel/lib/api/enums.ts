import { request } from './client';
import type { FieldEnumMap } from '@/lib/types';

export function fetchEnums(field?: string) {
  const path = field ? `/enums?field=${encodeURIComponent(field)}` : '/enums';
  return request<FieldEnumMap>(path);
}
