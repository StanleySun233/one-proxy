import { request } from './client';
import type { Chain, ChainProbeResult, ChainValidationResult, ChainPreviewResult } from '@/lib/types';

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
