import type { APIResponse, Account } from '@/lib/types';

export const CONTROL_PLANE_PROXY_BASE = '/api/v1';
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
};

type RequestOptions = {
  accessToken?: string;
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  body?: unknown;
};

function notifyUnauthorized() {
  if (typeof window === 'undefined') {
    return;
  }

  window.localStorage.removeItem(SESSION_STORAGE_KEY);
  window.dispatchEvent(new Event(AUTH_INVALID_EVENT));
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers();

  if (options.accessToken) {
    headers.set('Authorization', `Bearer ${options.accessToken}`);
  }
  if (options.body !== undefined) {
    headers.set('Content-Type', 'application/json');
  }

  let response: Response;
  try {
    response = await fetch(`${CONTROL_PLANE_PROXY_BASE}${path}`, {
      method: options.method || 'GET',
      headers,
      body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
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
    if (response.status === 401) {
      notifyUnauthorized();
    }
    throw new ControlPlaneAPIError(code, code, response.status);
  }

  return envelope.data;
}

export { notifyUnauthorized, request };
