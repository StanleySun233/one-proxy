import { request } from './client';
import type { Certificate } from '@/lib/types';

export function getCertificates(accessToken: string) {
  return request<Certificate[]>('/certificates', {accessToken});
}
