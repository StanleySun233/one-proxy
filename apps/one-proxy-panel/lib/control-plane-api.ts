import {
  Account,
  APIResponse,
  BootstrapToken,
  Certificate,
  Chain,
  ChainProbeResult,
  ConnectedNodeResult,
  Group,
  GroupDetail,
  LoginResult,
  Node,
  NodeAccessPath,
  NodeEnrollmentApproval,
  NodeHealth,
  NodeHealthHistory,
  NodeLink,
  NodeTransport,
  NodeOnboardingTask,
  Overview,
  PolicyRevision,
  RouteRule
} from '@/lib/control-plane-types';

const CONTROL_PLANE_PROXY_BASE = '/api/v1';
const SESSION_STORAGE_KEY = 'one-proxy-panel-session';
const AUTH_INVALID_EVENT = 'one-proxy-auth-invalid';

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
  method?: 'GET' | 'POST' | 'PATCH' | 'DELETE';
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

export function login(account: string, password: string) {
  return request<LoginResult>('/auth/login', {
    method: 'POST',
    body: {account, password}
  });
}

export function logout(accessToken: string) {
  return request<{status: string}>('/auth/logout', {
    method: 'POST',
    accessToken
  });
}

export function getOverview(accessToken: string) {
  return request<Overview>('/overview', {accessToken});
}

export function getAccounts(accessToken: string) {
  return request<Account[]>('/accounts', {accessToken});
}

export function createAccount(accessToken: string, payload: {account: string; password: string; role: string}) {
 return request<Account>('/accounts', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function updateAccount(
  accessToken: string,
  accountID: string,
  payload: {password?: string; role?: string; status?: string}
) {
  return request<Account>(`/accounts/${accountID}`, {
    method: 'PATCH',
    accessToken,
    body: payload
  });
}

export function deleteAccount(accessToken: string, accountID: string) {
  return request<{status: string}>(`/accounts/${accountID}`, {
    method: 'DELETE',
    accessToken
  });
}

export function getNodes(accessToken: string) {
  return request<Node[]>('/nodes', {accessToken});
}

export function createNode(
  accessToken: string,
  payload: {name: string; mode: string; scopeKey: string; parentNodeId: string; publicHost: string; publicPort: number}
) {
  return request<Node>('/nodes', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function updateNode(
  accessToken: string,
  nodeID: string,
  payload: {
    name: string;
    mode: string;
    scopeKey: string;
    parentNodeId: string;
    publicHost: string;
    publicPort: number;
    enabled: boolean;
    status: string;
  }
) {
  return request<Node>(`/nodes/${nodeID}`, {
    method: 'PATCH',
    accessToken,
    body: payload
  });
}

export function deleteNode(accessToken: string, nodeID: string) {
  return request<{status: string}>(`/nodes/${nodeID}`, {
    method: 'DELETE',
    accessToken
  });
}

export function connectNode(
  accessToken: string,
  payload: {
    address: string;
    password: string;
    newPassword: string;
    name: string;
    mode: string;
    scopeKey: string;
    parentNodeId: string;
    publicHost: string;
    publicPort: number;
    controlPlaneUrl: string;
  }
) {
  return request<ConnectedNodeResult>('/nodes/connect', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function getNodeLinks(accessToken: string) {
  return request<NodeLink[]>('/node-links', {accessToken});
}

export function getNodeTransports(accessToken: string) {
  return request<NodeTransport[]>('/node-transports', {accessToken});
}

export function createBootstrapToken(accessToken: string, payload: {targetType: string; targetId: string}) {
  return request<BootstrapToken>('/nodes/bootstrap-token', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function approveNode(accessToken: string, nodeID: string) {
  return request<{node: Node; accessToken: string; trustMaterial: string; expiresAt: string}>(`/nodes/approve/${nodeID}`, {
    method: 'POST',
    accessToken
  });
}

export function getNodeEnrollmentApprovals(accessToken: string) {
  return request<NodeEnrollmentApproval[]>('/nodes/approvals', {accessToken});
}

export function approveEnrollment(accessToken: string, approvalId: string, operatorNote?: string) {
  return request<NodeEnrollmentApproval>(`/nodes/approvals/${approvalId}/approve`, {
    method: 'POST',
    accessToken,
    body: {operatorNote}
  });
}

export function rejectEnrollment(accessToken: string, approvalId: string, operatorNote?: string) {
  return request<NodeEnrollmentApproval>(`/nodes/approvals/${approvalId}/reject`, {
    method: 'POST',
    accessToken,
    body: {operatorNote}
  });
}

export function getNodeAccessPaths(accessToken: string) {
  return request<NodeAccessPath[]>('/node-access-paths', {accessToken});
}

export function createNodeAccessPath(
  accessToken: string,
  payload: {
    name: string;
    mode: string;
    targetNodeId: string;
    entryNodeId: string;
    relayNodeIds: string[];
    targetHost: string;
    targetPort: number;
  }
) {
  return request<NodeAccessPath>('/node-access-paths', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function updateNodeAccessPath(
  accessToken: string,
  pathID: string,
  payload: {
    name: string;
    mode: string;
    targetNodeId: string;
    entryNodeId: string;
    relayNodeIds: string[];
    targetHost: string;
    targetPort: number;
    enabled: boolean;
  }
) {
  return request<NodeAccessPath>(`/node-access-paths/${pathID}`, {
    method: 'PATCH',
    accessToken,
    body: payload
  });
}

export function deleteNodeAccessPath(accessToken: string, pathID: string) {
  return request<{status: string}>(`/node-access-paths/${pathID}`, {
    method: 'DELETE',
    accessToken
  });
}

export function getNodeOnboardingTasks(accessToken: string) {
  return request<NodeOnboardingTask[]>('/node-onboarding-tasks', {accessToken});
}

export function createNodeOnboardingTask(
  accessToken: string,
  payload: {
    mode: string;
    pathId: string;
    targetNodeId: string;
    targetHost: string;
    targetPort: number;
  }
) {
  return request<NodeOnboardingTask>('/node-onboarding-tasks', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function updateNodeOnboardingTaskStatus(
  accessToken: string,
  taskID: string,
  payload: {
    status: string;
    statusMessage: string;
  }
) {
  return request<NodeOnboardingTask>(`/node-onboarding-tasks/${taskID}`, {
    method: 'PATCH',
    accessToken,
    body: payload
  });
}

export function getChains(accessToken: string) {
  return request<Chain[]>('/chains', {accessToken});
}

export function createChain(accessToken: string, payload: {name: string; destinationScope: string; hops: number[]}) {
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

export function getRouteRules(accessToken: string) {
  return request<RouteRule[]>('/route-rules', {accessToken});
}

export function createRouteRule(
  accessToken: string,
  payload: {
    priority: number;
    matchType: string;
    matchValue: string;
    actionType: string;
    chainId: string;
    destinationScope: string;
  }
) {
  return request<RouteRule>('/route-rules', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function getPolicyRevisions(accessToken: string) {
  return request<PolicyRevision[]>('/policies/revisions', {accessToken});
}

export function publishPolicy(accessToken: string) {
  return request<PolicyRevision>('/policies/publish', {
    method: 'POST',
    accessToken
  });
}

export function getCertificates(accessToken: string) {
  return request<Certificate[]>('/certificates', {accessToken});
}

export function getNodeHealth(accessToken: string) {
  return request<NodeHealth[]>('/nodes/health', {accessToken});
}

export function getNodeHealthHistory(accessToken: string, nodeId: string, window?: string) {
  const params = new URLSearchParams({nodeId});
  if (window) params.set('window', window);
  return request<NodeHealthHistory[]>(`/nodes/health/history?${params.toString()}`, {accessToken});
}

export function listGroups(accessToken: string) {
  return request<Group[]>('/groups', {accessToken});
}

export function createGroup(
  accessToken: string,
  payload: {name: string; description: string; enabled: boolean}
) {
  return request<Group>('/groups', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function getGroup(accessToken: string, groupID: string) {
  return request<GroupDetail>(`/groups/${groupID}`, {accessToken});
}

export function updateGroup(
  accessToken: string,
  groupID: string,
  payload: {name?: string; description?: string; enabled?: boolean}
) {
  return request<Group>(`/groups/${groupID}`, {
    method: 'PUT',
    accessToken,
    body: payload
  });
}

export function deleteGroup(accessToken: string, groupID: string) {
  return request<{status: string}>(`/groups/${groupID}`, {
    method: 'DELETE',
    accessToken
  });
}

export function setGroupAccounts(accessToken: string, groupID: string, accountIds: string[]) {
  return request<{status: string}>(`/groups/${groupID}/accounts`, {
    method: 'PUT',
    accessToken,
    body: {accountIds}
  });
}

export function setGroupScopes(accessToken: string, groupID: string, scopeKeys: string[]) {
  return request<{status: string}>(`/groups/${groupID}/scopes`, {
    method: 'PUT',
    accessToken,
    body: {scopeKeys}
  });
}

export {AUTH_INVALID_EVENT, SESSION_STORAGE_KEY};
