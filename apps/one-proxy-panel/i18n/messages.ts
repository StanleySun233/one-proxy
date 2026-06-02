const messageNamespaces = [
  'meta',
  'shell',
  'auth',
  'setup',
  'nav',
  'overview',
  'pages',
  'chainsRoutes',
  'chainsScopes',
  'accounts',
  'common',
  'chains',
  'health',
  'nodesConsole',
  'nodes',
  'accessPaths'
] as const;

export async function loadMessages(locale: string) {
  const entries = await Promise.all(
    messageNamespaces.map(async (namespace) => [
      namespace,
      (await import(`../messages/${locale}/${namespace}.json`)).default
    ])
  );

  return Object.fromEntries(entries);
}
