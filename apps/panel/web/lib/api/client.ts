import type { APIResponse, Account } from '@/lib/types';
import type { TenantMembership } from '@/lib/types/auth';

export const CONTROL_PLANE_PROXY_BASE = '/api';
export const SESSION_STORAGE_KEY = 'one-proxy-panel-session';
export const AUTH_INVALID_EVENT = 'one-proxy-auth-invalid';

export class ControlPlaneAPIError extends Error {
  code: string;
  status: number;

  constructor(message: string, code: string, status: number) {
    super(message);
    this.name = 'ControlPlaneAPIError';
    this.code = code;
    this.status = status;
  }
}

export type Session = {
  account: Account;
  accessToken: string;
  refreshToken: string;
  expiresAt: string;
  mustRotatePassword: boolean;
  tenantMemberships: TenantMembership[];
  activeTenantId: string | null;
};

type RequestOptions = {
  accessToken?: string;
  tenantId?: string | null;
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  body?: unknown;
  headers?: Record<string, string>;
};

function notifyUnauthorized() {
  if (typeof window === 'undefined') {
    return;
  }

  window.dispatchEvent(new Event(AUTH_INVALID_EVENT));
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers();
  const tenantId = options.tenantId || null;
  const body = path === '/auth/refresh' ? undefined : options.body;

  for (const [key, value] of Object.entries(options.headers || {})) {
    headers.set(key, value);
  }
  if (options.accessToken) {
    headers.set('X-One-Proxy-Access-Token', options.accessToken);
  }
  if (tenantId) {
    headers.set('X-One-Proxy-Tenant-ID', tenantId);
  }
  if (body !== undefined) {
    headers.set('Content-Type', 'application/json');
  }

  let response: Response;
  try {
    response = await fetch(`${CONTROL_PLANE_PROXY_BASE}${path}`, {
      method: options.method || 'GET',
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
      cache: 'no-store'
    });
  } catch {
    throw new ControlPlaneAPIError('network_unreachable', 'network_unreachable', 0);
  }

  const raw = await response.text();
  let envelope: APIResponse<T> | null = null;

  if (raw) {
    try {
      envelope = JSON.parse(raw) as APIResponse<T>;
    } catch {
      envelope = null;
    }
  }

  if (!response.ok || !envelope || envelope.code !== 0) {
    const code = envelope?.message || `http_${response.status}`;
    if (response.status === 401 && options.accessToken) {
      notifyUnauthorized();
    }
    throw new ControlPlaneAPIError(code, code, response.status);
  }

  return envelope.data;
}

export { notifyUnauthorized, request };
