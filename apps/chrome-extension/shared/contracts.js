export const defaultSessionState = {
  auth: {
    account: '',
    accessToken: '',
    refreshToken: '',
    expiresAt: '',
    mustRotatePassword: false
  },
  selection: {
    nodeId: '',
    profileId: '',
    routingMode: 'smart'
  },
  policy: {
    revision: '',
    fetchedAt: '',
    certificateState: 'unknown'
  },
  localOverrides: {
    directHosts: [],
    proxyHosts: [],
    internalHosts: []
  },
  ui: {
    themeMode: 'vivid'
  }
};
