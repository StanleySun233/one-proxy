import assert from 'node:assert/strict';
import { mkdtemp, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';

import {
  readProfilesIndex,
  readConfig,
  storageFile,
  writeConfig,
  writeState,
  writeTokens
} from '../src/storage.ts';
import { routeRulesFromBootstrap } from '../src/control-plane.ts';
import { resolveRoute } from '../src/daemon/router.ts';
import { parseSshCommandArgs, parseSshTarget } from '../src/ssh.ts';
import { parseShellCommandArgs } from '../src/shell.ts';
import { parseEnvCommandArgs, parseRunCommandArgs, proxyEnv } from '../src/session-env.ts';
import { runIsolationInternals, runProxyOnlyBestEffortCommand } from '../src/run-isolation.ts';
import { detectShellFamily, detectShellPath } from '../src/shell-detect.ts';
import { detectTuiCapability, tuiUnavailableWarning } from '../src/tui/capability.ts';
import { buildTuiStatusSnapshot, collectTuiStatusSnapshot } from '../src/tui/status.ts';

const repoRoot = path.resolve(import.meta.dirname, '../../..');
const mainEntrypoint = path.join(repoRoot, 'apps/cli/src/main.ts');

async function withHome(fn) {
  const previous = process.env.ONEPROXY_HOME;
  const home = await mkdtemp(path.join(tmpdir(), 'oneproxy-cli-test-'));
  process.env.ONEPROXY_HOME = home;
  try {
    return await fn(home);
  } finally {
    if (previous === undefined) {
      delete process.env.ONEPROXY_HOME;
    } else {
      process.env.ONEPROXY_HOME = previous;
    }
    await rm(home, { recursive: true, force: true });
  }
}

function runCli(args, home, extraEnv = {}) {
  return spawnSync(process.execPath, [mainEntrypoint, ...args], {
    cwd: repoRoot,
    env: {
      ...process.env,
      ONEPROXY_HOME: home,
      ...extraEnv
    },
    encoding: 'utf8'
  });
}

test('storage normalizes defaults and override hosts', async () => {
  await withHome(async () => {
    assert.deepEqual(await readConfig(), {
      schemaVersion: 1,
      profileName: 'default',
      overrides: { direct: [], proxy: [] }
    });

    await writeConfig({
      schemaVersion: 99,
      controlPlaneUrl: 'https://control.example.com',
      activeTenantId: 'tenant_1',
      activeGroupId: 'group_1',
      overrides: {
        direct: [' Example.COM ', 'example.com', 'LOCALHOST'],
        proxy: ['Proxy.Example', '', 'proxy.example']
      }
    });

    const config = await readConfig();
    assert.equal(config.schemaVersion, 1);
    assert.deepEqual(config.overrides.direct, ['example.com', 'localhost']);
    assert.deepEqual(config.overrides.proxy, ['proxy.example']);

    const stored = JSON.parse(await readFile(storageFile('config'), 'utf8'));
    assert.deepEqual(stored.overrides.direct, ['example.com', 'localhost']);
  });
});

test('profile commands manage active panel urls', async () => {
  await withHome(async (home) => {
    const addCamel = runCli(['profile', 'add', 'camel', '--control-plane', 'https://camel.example.com'], home);
    assert.equal(addCamel.status, 0, addCamel.stderr);

    const addLab = runCli(['profile', 'add', 'lab', '--control-plane', 'https://lab.example.com'], home);
    assert.equal(addLab.status, 0, addLab.stderr);

    const useCamel = runCli(['profile', 'use', 'camel'], home);
    assert.equal(useCamel.status, 0, useCamel.stderr);

    const current = runCli(['profile', 'current', '--json'], home);
    assert.equal(current.status, 0, current.stderr);
    assert.deepEqual(JSON.parse(current.stdout), {
      activeProfile: 'camel',
      profile: {
        name: 'camel',
        controlPlaneUrl: 'https://camel.example.com'
      }
    });

    const index = await readProfilesIndex();
    assert.equal(index.activeProfile, 'camel');
    assert.deepEqual(Object.keys(index.profiles).sort(), ['camel', 'lab']);
  });
});

test('route matching applies direct override before proxy override and policy', () => {
  const route = resolveRoute({
    config: {
      schemaVersion: 1,
      activeTenantId: 'tenant_1',
      activeGroupId: 'group_1',
      overrides: {
        direct: ['example.com'],
        proxy: ['example.com', 'proxy.local']
      }
    },
    state: {
      schemaVersion: 1,
      bootstrap: {
        entryNodes: [{ id: 'entry_1', host: 'edge.example.com', port: 443, protocol: 'connect' }]
      },
      routeGroups: [
        {
          id: 'group_1',
          tenantId: 'tenant_1',
          name: 'Default',
          rules: [{ id: 'rule_1', type: 'suffix', pattern: 'example.com', mode: 'proxy' }]
        }
      ]
    },
    target: 'https://example.com/path'
  });

  assert.equal(route.mode, 'direct');
  assert.equal(route.matched.source, 'local_override_direct');
  assert.equal(route.tenant.id, 'tenant_1');
  assert.equal(route.group.id, 'group_1');
  assert.equal(route.topology, null);
});

test('route matching returns proxied topology for suffix policy', () => {
  const route = resolveRoute({
    config: {
      schemaVersion: 1,
      activeTenantId: 'tenant_1',
      activeGroupId: 'group_1',
      overrides: { direct: [], proxy: [] }
    },
    state: {
      schemaVersion: 1,
      bootstrap: {
        entryNodes: [{ id: 'entry_1', host: 'edge.example.com', port: 443, protocol: 'connect' }]
      },
      routeGroups: [
        {
          id: 'group_1',
          tenantId: 'tenant_1',
          name: 'Default',
          rules: [{ id: 'rule_1', type: 'suffix', pattern: 'example.com', mode: 'proxy' }]
        }
      ]
    },
    target: 'api.example.com',
    protocol: 'https'
  });

  assert.equal(route.mode, 'proxy');
  assert.equal(route.port, 443);
  assert.deepEqual(route.topology, {
    entryNodeId: 'entry_1',
    entryHost: 'edge.example.com',
    entryPort: 443,
    protocol: 'connect'
  });
});

test('route matching treats dot and wildcard suffix policies as root plus subdomains', () => {
  for (const pattern of ['.openai.com', '*.openai.com']) {
    for (const target of ['openai.com', 'api.openai.com']) {
      const route = resolveRoute({
        config: {
          schemaVersion: 1,
          activeTenantId: 'tenant_1',
          activeGroupId: 'group_1',
          overrides: { direct: [], proxy: [] }
        },
        state: {
          schemaVersion: 1,
          bootstrap: {
            entryNodes: [{ id: 'entry_1', host: 'edge.example.com', port: 443, protocol: 'connect' }]
          },
          routeGroups: [
            {
              id: 'group_1',
              tenantId: 'tenant_1',
              name: 'Default',
              rules: [{ id: `rule_${pattern}`, type: 'suffix', pattern, mode: 'proxy' }]
            }
          ]
        },
        target,
        protocol: 'https'
      });

      assert.equal(route.mode, 'proxy');
      assert.equal(route.matched.ruleType, 'suffix');
      assert.equal(route.matched.pattern, pattern);
    }
  }
});

test('env off prints shell restoration output', async () => {
  await withHome(async (home) => {
    const result = runCli(['env', 'off'], home, { ONEPROXY_SHELL: '/bin/bash' });
    assert.equal(result.status, 0, result.stderr);
    assert.match(result.stdout, /unset ONEPROXY_ACTIVE ONEPROXY_HTTP_PORT ONEPROXY_HTTPS_PORT/);
    assert.equal(result.stderr, '');
  });
});

test('env command relies on shell auto-detection instead of shell flags', () => {
  assert.deepEqual(parseEnvCommandArgs([]), {});
  assert.throws(() => parseEnvCommandArgs(['--shell', 'bash']), /Unknown env option: --shell/);
});

test('shell detection prefers the current parent shell over the login shell', () => {
  assert.equal(detectShellPath({ env: { SHELL: '/bin/bash' }, parentShell: 'fish' }), 'fish');
  assert.equal(detectShellFamily({ env: { SHELL: '/bin/bash' }, parentShell: 'fish' }), 'fish');
  assert.equal(detectShellPath({ env: { ONEPROXY_SHELL: 'zsh', SHELL: '/bin/bash' }, parentShell: 'fish' }), 'zsh');
  assert.equal(detectShellPath({ env: { SHELL: '/bin/zsh' }, parentShell: 'node' }), '/bin/zsh');
});

test('proxy env includes lower-case variables and bypass hosts', () => {
  const env = proxyEnv({
    host: '127.0.0.1',
    httpPort: 10080,
    httpsPort: 10081,
    ipcPort: 10082
  }, ['panel.example.com', 'edge.example.com']);

  assert.equal(env.HTTP_PROXY, 'http://127.0.0.1:10080');
  assert.equal(env.http_proxy, env.HTTP_PROXY);
  assert.equal(env.HTTPS_PROXY, 'http://127.0.0.1:10081');
  assert.equal(env.https_proxy, env.HTTPS_PROXY);
  assert.equal(env.ALL_PROXY, env.HTTP_PROXY);
  assert.equal(env.all_proxy, env.HTTP_PROXY);
  assert.equal(env.NO_PROXY, 'localhost,127.0.0.1,::1,panel.example.com,edge.example.com');
  assert.equal(env.no_proxy, env.NO_PROXY);
});

test('run isolation firewall allows only the proxy-only port before rejecting egress', () => {
  const rules = runIsolationInternals.firewallRules(12000, 'oneproxy-run-test');
  assert.deepEqual(rules.map((rule) => rule.command), ['iptables', 'iptables', 'ip6tables']);
  assert.deepEqual(rules[0].add, [
    '-I', 'OUTPUT', '1',
    '-p', 'tcp',
    '-m', 'cgroup', '--path', 'oneproxy-run-test',
    '-d', '127.0.0.1/32',
    '--dport', '12000',
    '-j', 'ACCEPT'
  ]);
  assert.deepEqual(rules[1].add, [
    '-I', 'OUTPUT', '1',
    '-m', 'cgroup', '--path', 'oneproxy-run-test',
    '-j', 'REJECT'
  ]);
  assert.deepEqual(rules[2].add, [
    '-I', 'OUTPUT', '1',
    '-m', 'cgroup', '--path', 'oneproxy-run-test',
    '-j', 'REJECT'
  ]);
});

test('run isolation fallback detection only matches isolation support errors', () => {
  assert.equal(runIsolationInternals.isProxyIsolationUnavailable(Object.assign(new Error('missing root'), { code: 'PROXY_ISOLATION_REQUIRED' })), true);
  assert.equal(runIsolationInternals.isProxyIsolationUnavailable(Object.assign(new Error('cleanup failed'), { code: 'PROXY_ISOLATION_CLEANUP_FAILED' })), false);
});

test('run isolation help gives install guidance for missing cgroup and iptables', () => {
  assert.deepEqual(runIsolationInternals.proxyIsolationHelp(Object.assign(new Error('missing iptables'), { code: 'PROXY_ISOLATION_REQUIRED', reason: 'missing_iptables' })), [
    'Install iptables/ip6tables for strict isolation:',
    '  Debian/Ubuntu: sudo apt-get install iptables',
    '  Fedora/RHEL: sudo dnf install iptables iptables-nft',
    '  Arch: sudo pacman -S iptables'
  ]);
  assert.deepEqual(runIsolationInternals.proxyIsolationHelp(Object.assign(new Error('missing cgroup'), { code: 'PROXY_ISOLATION_REQUIRED', reason: 'missing_cgroup_v2' })), [
    'Enable cgroup v2 at /sys/fs/cgroup for strict isolation.',
    'cgroup v2 is a Linux host capability, not a OneProxy package dependency.',
    'On systemd Linux, /sys/fs/cgroup should be mounted as cgroup2fs.'
  ]);
});

test('best-effort run passes proxy-only environment to the child', async () => {
  const exitCode = await runProxyOnlyBestEffortCommand({
    executable: process.execPath,
    args: ['-e', 'process.exit(process.env.HTTP_PROXY === "http://127.0.0.1:12000" && process.env.NO_PROXY === "" && process.env.ONEPROXY_PROXY_ONLY === "1" ? 0 : 9)'],
    env: {
      ...process.env,
      HTTP_PROXY: 'http://127.0.0.1:12000',
      NO_PROXY: '',
      ONEPROXY_PROXY_ONLY: '1'
    },
    proxyPort: 12000
  });

  assert.equal(exitCode, 0);
});

test('command parser handles help and unknown commands', async () => {
  await withHome(async (home) => {
    const help = runCli(['help'], home);
    assert.equal(help.status, 0, help.stderr);
    assert.match(help.stdout, /Usage: onep <command>/);

    const unknown = runCli(['missing-command'], home);
    assert.equal(unknown.status, 2);
    assert.match(unknown.stderr, /Unknown command: missing-command/);
  });
});

test('ssh --tui parsing strips the runtime flag before target parsing', () => {
  const parsed = parseSshCommandArgs(['stanley@Ssh.Example', '-p', '2222', '--tui']);
  assert.deepEqual(parsed, { args: ['stanley@Ssh.Example', '-p', '2222'], tui: true });

  assert.deepEqual(parseSshTarget(parsed.args), {
    user: 'stanley',
    host: 'ssh.example',
    port: 2222,
    original: 'stanley@Ssh.Example'
  });
});

test('shell --tui parsing enables TUI without passing the flag to the child shell', () => {
  assert.deepEqual(parseShellCommandArgs(['--tui']), { args: [], tui: true });
  assert.deepEqual(parseShellCommandArgs([]), { args: [], tui: false });
  assert.throws(() => parseShellCommandArgs(['--shell', 'zsh']), /Unknown shell option: --shell/);
});

test('run --tui parsing preserves the command argv after the runtime flag', () => {
  assert.deepEqual(parseRunCommandArgs(['--tui', 'node', 'script.mjs', '--inspect']), {
    args: ['node', 'script.mjs', '--inspect'],
    tui: true
  });
});

test('explicit TUI request warns once before fallback when PTY is unavailable', () => {
  const capability = detectTuiCapability({
    requested: true,
    interactive: true,
    json: false,
    stdinIsTty: true,
    stdoutIsTty: true,
    stderrIsTty: true,
    term: 'xterm-256color',
    platform: 'linux',
    rows: 20,
    ptyAvailable: false
  });

  assert.deepEqual(capability, { enabled: false, warn: true, reason: 'pty_unavailable' });
  assert.equal(tuiUnavailableWarning, '! TUI failed to start; falling back to standard terminal mode.');
});

test('TUI status module exports the runtime snapshot entrypoint', () => {
  assert.equal(buildTuiStatusSnapshot, collectTuiStatusSnapshot);
});

test('status --json output matches contract shape', async () => {
  await withHome(async (home) => {
    await writeConfig({
      schemaVersion: 1,
      controlPlaneUrl: 'https://control.example.com',
      activeTenantId: 'tenant_1',
      activeGroupId: 'group_1',
      overrides: { direct: ['direct.example'], proxy: ['proxy.example'] }
    });
    await writeState({
      schemaVersion: 1,
      bootstrap: { groupId: 'group_1', entryNodes: [] },
      policyRevision: 'rev_1',
      fetchedAt: '2026-06-06T06:00:00.000Z',
      routeGroups: [{ id: 'group_1', tenantId: 'tenant_1', name: 'Default', rules: [] }]
    });
    await writeTokens({
      schemaVersion: 1,
      account: { id: 'user_1', email: 'user@example.com' },
      accessTokenExpiresAt: '2026-06-06T07:00:00.000Z',
      refreshTokenExpiresAt: '2026-07-06T06:00:00.000Z',
      proxyTokenExpiresAt: '2026-06-06T07:00:00.000Z'
    });

    const result = runCli(['status', '--json'], home);
    assert.equal(result.status, 0, result.stderr);
    const status = JSON.parse(result.stdout);
    assert.deepEqual(Object.keys(status).sort(), [
      'account',
      'controlPlane',
      'daemon',
      'group',
      'localPorts',
      'overrides',
      'policyRevision',
      'portSelection',
      'tenant',
      'tokens'
    ]);
    assert.equal(status.account.email, 'user@example.com');
    assert.equal(status.localPorts.http, null);
    assert.equal(status.localPorts.https, null);
    assert.equal(status.overrides.directCount, 1);
  });
});

test('bootstrap route conversion preserves cidr match type', () => {
  const rules = routeRulesFromBootstrap({
    id: 'group_1',
    name: 'hk-public-node',
    routes: [
      { id: 'route_1', matchType: 'ip_cidr', matchValue: '172.20.116.0/24', actionType: 'chain' },
      { id: 'route_2', matchType: 'domain_suffix', matchValue: '.example.com', actionType: 'direct' },
      { id: 'route_3', matchType: 'suffix', matchValue: 'legacy.example.com', actionType: 'direct' },
      { id: 'route_4', matchType: 'default', matchValue: '*', actionType: 'proxy' }
    ]
  });

  assert.deepEqual(rules, [
    { id: 'route_1', type: 'cidr', pattern: '172.20.116.0/24', mode: 'proxy' },
    { id: 'route_2', type: 'suffix', pattern: '.example.com', mode: 'direct' },
    { id: 'route_3', type: 'suffix', pattern: 'legacy.example.com', mode: 'direct' },
    { id: 'route_4', type: 'wildcard', pattern: '*', mode: 'proxy' }
  ]);
});

test('bootstrap host pattern conversion preserves suffix semantics', () => {
  const rules = routeRulesFromBootstrap({
    id: 'group_1',
    name: 'hk-public-node',
    proxyHosts: ['*.openai.com'],
    directHosts: ['.example.com', 'exact.example.net']
  });

  assert.deepEqual(rules, [
    { id: 'direct:.example.com', type: 'suffix', pattern: '.example.com', mode: 'direct' },
    { id: 'direct:exact.example.net', type: 'domain', pattern: 'exact.example.net', mode: 'direct' },
    { id: 'proxy:*.openai.com', type: 'suffix', pattern: '*.openai.com', mode: 'proxy' }
  ]);
});
