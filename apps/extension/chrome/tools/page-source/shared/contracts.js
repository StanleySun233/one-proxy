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
    policyRevision: '',
    fetchedAt: '',
    groups: []
  },
  selection: {
    activeGroupId: ''
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
