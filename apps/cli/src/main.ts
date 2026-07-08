#!/usr/bin/env node
import * as fs from 'node:fs/promises';
import { accessPathList, accessPathUse, autoSyncRemoteState, login, logout, sync, tenantList, tenantUse } from './control-plane.ts';
import { envOff, envOn, onepOff, onepOn, runCommand } from './session-env.ts';
import { doctor, overrideCommand, routeCommand, statusCommand, testCommand, writeError } from './commands.ts';
import { serveDaemon } from './daemon/lifecycle.ts';
import { runSsh } from './ssh.ts';
import { profileCommand } from './profile.ts';
import { initCommand } from './init.ts';
import { shellCommand } from './shell.ts';
import { monitorCommand } from './monitor.ts';
import { maybeHandleCliUpdate } from './update-check.ts';

export type CliContext = {
  json: boolean;
};

type CommandHandler = (args: string[], context: CliContext) => Promise<number | void>;

async function packageVersion(): Promise<string> {
  const packageJson = JSON.parse(await fs.readFile(new URL('../package.json', import.meta.url), 'utf8')) as { version: string };
  return packageJson.version;
}

function stripGlobalFlags(args: string[]): { args: string[]; context: CliContext } {
  const remaining: string[] = [];
  let json = false;
  for (const arg of args) {
    if (arg === '--json') {
      json = true;
    } else {
      remaining.push(arg);
    }
  }
  return { args: remaining, context: { json } };
}

function usage(): string {
  return [
    'Usage: onep <command> [options]',
    '',
    'Commands:',
    '  init',
    '  version',
    '  login',
    '  logout',
    '  profile add <name> --control-plane <url>|use <name>|list|current',
    '  tenant list|use <name-or-id>',
    '  access-path list|use <name-or-id>',
    '  sync',
    '  status [--json]',
    '  on',
    '  off',
    '  env [on|off]',
    '  shell',
    '  run <command...>',
    '  monitor <command...>',
    '  override list|direct add <host>|proxy add <host>|remove <host>|clear',
    '  route <url-or-host> [--json]',
    '  test <url-or-host> [--json]',
    '  ssh <host|user@host> [-p <port>]',
    '  doctor [--json]'
  ].join('\n');
}

function requireArg(value: string | undefined, message: string): string {
  if (!value) {
    throw Object.assign(new Error(message), { code: 'SYNTAX_ERROR', exitCode: 2 });
  }
  return value;
}

const handlers: Record<string, CommandHandler> = {
  version: async () => {
    process.stdout.write(`${await packageVersion()}\n`);
  },
  login,
  init: initCommand,
  logout,
  sync,
  status: statusCommand,
  on: async (args) => onepOn(args),
  off: async (args) => onepOff(args),
  route: routeCommand,
  test: testCommand,
  doctor,
  profile: profileCommand,
  ssh: runSsh,
  override: overrideCommand,
  shell: shellCommand,
  monitor: monitorCommand,
  run: runCommand,
  daemon: async (args) => {
    if (args[0] === 'serve') {
      await serveDaemon();
      return 0;
    }
    throw Object.assign(new Error('daemon requires serve'), { code: 'SYNTAX_ERROR', exitCode: 2 });
  },
  env: async (args) => {
    const mode = args[0] && !args[0].startsWith('-') ? args[0] : 'on';
    const options = mode === args[0] ? args.slice(1) : args;
    if (mode === 'on') {
      return envOn(options);
    }
    if (mode === 'off') {
      return envOff(options);
    }
    throw Object.assign(new Error(`Unknown env mode: ${mode}`), { code: 'SYNTAX_ERROR', exitCode: 2 });
  },
  tenant: async (args, context) => {
    const action = args[0];
    if (action === 'list') {
      return tenantList(args.slice(1), context);
    }
    if (action === 'use') {
      return tenantUse([requireArg(args[1], 'tenant use requires a tenant name or id')], context);
    }
    throw Object.assign(new Error('tenant requires list or use'), { code: 'SYNTAX_ERROR', exitCode: 2 });
  },
  'access-path': async (args, context) => {
    const action = args[0];
    if (action === 'list') {
      return accessPathList(args.slice(1), context);
    }
    if (action === 'use') {
      return accessPathUse([requireArg(args[1], 'access-path use requires an access path name or id')], context);
    }
    throw Object.assign(new Error('access-path requires list or use'), { code: 'SYNTAX_ERROR', exitCode: 2 });
  }
};

export async function main(argv = process.argv.slice(2)): Promise<number> {
  const parsed = stripGlobalFlags(argv);
  const command = parsed.args[0];
  if (!command || command === 'help' || command === '--help' || command === '-h') {
    process.stdout.write(`${usage()}\n`);
    return command ? 0 : 2;
  }
  const handler = handlers[command];
  if (!handler) {
    writeError({ code: 'COMMAND_NOT_FOUND', message: `Unknown command: ${command}` }, parsed.context);
    return 2;
  }
  if (await maybeHandleCliUpdate(command, parsed.context, await packageVersion())) {
    return 0;
  }
  if (shouldAutoSync(command)) {
    await autoSyncRemoteState();
  }
  const code = await handler(parsed.args.slice(1), parsed.context);
  return typeof code === 'number' ? code : 0;
}

function shouldAutoSync(command: string): boolean {
  if (process.env.ONEPROXY_DAEMON_CHILD === '1') {
    return false;
  }
  return !new Set(['daemon', 'init', 'login', 'logout', 'off', 'on', 'sync', 'version']).has(command);
}

main()
  .then((code) => {
    process.exitCode = code;
  })
  .catch((error) => {
    writeError({
      code: error.code || 'INTERNAL_ERROR',
      message: error.message || String(error)
    }, { json: process.argv.includes('--json') });
    process.exitCode = error.exitCode || 1;
  });
