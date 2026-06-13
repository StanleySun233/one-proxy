#!/usr/bin/env node
import * as fs from 'node:fs/promises';
import { login, logout, sync, tenantList, tenantUse, groupList, groupUse } from "./control-plane.js";
import { envOff, envOn, runCommand } from "./session-env.js";
import { doctor, overrideCommand, routeCommand, statusCommand, testCommand, writeError } from "./commands.js";
import { serveDaemon } from "./daemon/lifecycle.js";
import { runSsh } from "./ssh.js";
import { profileCommand } from "./profile.js";
import { initCommand } from "./init.js";
import { shellCommand } from "./shell.js";
import { monitorCommand } from "./monitor.js";
async function packageVersion() {
    const packageJson = JSON.parse(await fs.readFile(new URL('../package.json', import.meta.url), 'utf8'));
    return packageJson.version;
}
function stripGlobalFlags(args) {
    const remaining = [];
    let json = false;
    for (const arg of args) {
        if (arg === '--json') {
            json = true;
        }
        else {
            remaining.push(arg);
        }
    }
    return { args: remaining, context: { json } };
}
function usage() {
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
        '  group list|use <name-or-id>',
        '  sync',
        '  status [--json]',
        '  env [on|off]',
        '  shell [--tui]',
        '  run [--tui] <command...>',
        '  monitor <command...>',
        '  override list|direct add <host>|proxy add <host>|remove <host>|clear',
        '  route <url-or-host> [--json]',
        '  test <url-or-host> [--json]',
        '  ssh <host|user@host> [-p <port>] [--tui]',
        '  doctor [--json]'
    ].join('\n');
}
function requireArg(value, message) {
    if (!value) {
        throw Object.assign(new Error(message), { code: 'SYNTAX_ERROR', exitCode: 2 });
    }
    return value;
}
const handlers = {
    version: async () => {
        process.stdout.write(`${await packageVersion()}\n`);
    },
    login,
    init: initCommand,
    logout,
    sync,
    status: statusCommand,
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
        const mode = args[0] ?? 'on';
        if (mode === 'on') {
            return envOn();
        }
        if (mode === 'off') {
            return envOff();
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
    group: async (args, context) => {
        const action = args[0];
        if (action === 'list') {
            return groupList(args.slice(1), context);
        }
        if (action === 'use') {
            return groupUse([requireArg(args[1], 'group use requires a group name or id')], context);
        }
        throw Object.assign(new Error('group requires list or use'), { code: 'SYNTAX_ERROR', exitCode: 2 });
    }
};
export async function main(argv = process.argv.slice(2)) {
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
    const code = await handler(parsed.args.slice(1), parsed.context);
    return typeof code === 'number' ? code : 0;
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
//# sourceMappingURL=main.js.map