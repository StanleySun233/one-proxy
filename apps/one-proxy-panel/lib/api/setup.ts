import { request } from './client';
import type { SetupStatus, TestConnectionRequest, TestConnectionResult, GenerateKeyResult, InitRequest, InitResult } from '@/lib/types';

export function getSetupStatus() {
  return request<SetupStatus>('/setup/status');
}

export function testSetupConnection(payload: TestConnectionRequest) {
  return request<TestConnectionResult>('/setup/test-connection', {
    method: 'POST',
    body: payload,
  });
}

export function generateSetupKey() {
  return request<GenerateKeyResult>('/setup/generate-key');
}

export function submitSetupInit(payload: InitRequest) {
  return request<InitResult>('/setup/init', {
    method: 'POST',
    body: payload,
  });
}
