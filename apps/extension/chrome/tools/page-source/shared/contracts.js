export const defaultExtensionState = {
  enabled: false,
  themeMode: 'vivid',
  controlPlaneUrl: '',
  session: {
    account: '',
    accessToken: '',
    refreshToken: '',
    expiresAt: '',
    mustRotatePassword: false
  },
  remote: {
    schemaVersion: 'v2.1.0',
    policyRevision: '',
    fetchedAt: '',
    nodes: [],
    accessPaths: [],
    routes: [],
    routeEvaluation: {
      defaultClientMode: 'direct',
      defaultNodeMode: 'deny',
      ruleOrder: 'priority_asc_then_id_asc',
      noMatchNodeDenyReason: 'route_not_found',
      supportedMatchTypes: ['domain', 'domain_suffix', 'ip', 'ip_cidr', 'protocol', 'default'],
      supportedActions: ['chain', 'direct', 'deny']
    }
  },
  selection: {
    activeAccessPathId: ''
  },
  localOverrides: {
    directHosts: [],
    proxyHosts: []
  },
  localHelper: {
    enabled: false,
    scheme: 'SOCKS5',
    host: '127.0.0.1',
    port: 1080
  }
};
