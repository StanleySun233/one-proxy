import type {NodeAccessPath, RemoteProtocol, RemoteSecret} from '@/lib/types';

export type RemoteStatus = 'idle' | 'connecting' | 'connected' | 'disconnected' | 'failed';

export type GuacamoleRuntime = {
  Client: new (tunnel: unknown) => any;
  Keyboard: new (element?: HTMLElement | HTMLDocument) => any;
  Mouse: new (element: HTMLElement) => any;
  WebSocketTunnel: new (url: string) => unknown;
};

export const statusClass: Record<RemoteStatus, string> = {
  idle: 'is-untested',
  connecting: 'is-exists',
  connected: 'is-success',
  disconnected: 'is-untested',
  failed: 'is-failed'
};

export function remoteTCPPaths(paths: NodeAccessPath[]) {
  return paths.filter((path) => path.enabled && path.mode === 'tcp' && path.protocol === 'tcp' && path.serviceType === 'tcp_access');
}

export function validSecret(protocol: RemoteProtocol, secret: RemoteSecret) {
  if (protocol === 'rdp') {
    return !!secret.password;
  }
  return !!secret.password || !!secret.privateKey;
}
