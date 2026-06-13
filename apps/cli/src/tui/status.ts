import type { RouteResult } from '../daemon/router.ts';
import { readConfig, readDaemonMetadata, readTokens, type Account } from '../storage.ts';
import type { TuiPathNode, TuiStatusSnapshot } from './footer.ts';

export type TuiStatusInput = {
  route?: RouteResult | null;
  pingMs?: number | null;
  uploadBytes?: number | null;
  downloadBytes?: number | null;
};

export async function collectTuiStatusSnapshot(input: TuiStatusInput = {}): Promise<TuiStatusSnapshot> {
  const [config, tokens, daemon] = await Promise.all([readConfig(), readTokens(), readDaemonMetadata()]);
  return {
    account: accountLabel(tokens?.account),
    tenant: config.activeTenantId || daemon?.tenantId || 'none',
    pingMs: input.pingMs ?? daemonPingMs(daemon?.lastHeartbeatAt),
    uploadBytes: input.uploadBytes ?? null,
    downloadBytes: input.downloadBytes ?? null,
    path: pathSnapshot(input.route)
  };
}

function accountLabel(account: Account | undefined): string {
  return account?.email || account?.account || account?.id || 'not logged in';
}

function daemonPingMs(lastHeartbeatAt?: string): number | null {
  if (!lastHeartbeatAt) {
    return null;
  }
  const timestamp = Date.parse(lastHeartbeatAt);
  return Number.isFinite(timestamp) ? Math.max(0, Date.now() - timestamp) : null;
}

function pathSnapshot(route?: RouteResult | null): TuiStatusSnapshot['path'] {
  if (!route) {
    return {
      mode: 'unknown',
      transport: 'unknown',
      fallbackReason: '',
      nodes: []
    };
  }
  const transport = route.topology?.protocol || route.mode;
  return {
    mode: route.mode,
    transport,
    fallbackReason: '',
    nodes: pathNodes(route, transport)
  };
}

function pathNodes(route: RouteResult, transport: string): TuiPathNode[] {
  const nodes: TuiPathNode[] = [
    {
      id: 'user',
      name: 'User machine',
      kind: 'user',
      transport: 'client'
    }
  ];
  if (route.topology) {
    nodes.push({
      id: route.topology.entryNodeId,
      name: route.topology.entryNodeId,
      kind: 'node',
      transport
    });
  }
  nodes.push({
    id: route.host || route.target,
    name: route.host || route.target,
    kind: 'web',
    transport: 'target'
  });
  return nodes;
}
