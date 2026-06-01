const messageNamespaces = [
  'meta',
  'shell',
  'auth',
  'setup',
  'nav',
  'overview',
  'pages',
  'routes',
  'certificates',
  'scopes',
  'accounts',
  'common',
  'chains',
  'health',
  'nodesConsole',
  'nodes'
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
