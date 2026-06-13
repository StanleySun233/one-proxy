import assert from 'node:assert/strict';
import { EventEmitter } from 'node:events';
import test from 'node:test';

import {
  detectTuiCapability,
  tuiUnavailableWarning,
  footerRowsForTerminalHeight
} from '../src/tui/capability.ts';
import {
  formatFooter,
  formatPathText,
  latencyStyleName,
  stripAnsi,
  visibleWidth
} from '../src/tui/footer.ts';
import { runTuiRuntime, runTuiSession } from '../src/tui/runtime.ts';

function status(overrides = {}) {
  return {
    account: 'stanley@example.com',
    tenant: 'demo',
    pingMs: 32,
    uploadBytes: 12_400_000,
    downloadBytes: 91_800_000,
    path: {
      mode: 'proxy',
      transport: 'connect',
      fallbackReason: '',
      nodes: [
        { id: 'user_1', name: 'user', kind: 'user', transport: 'local' },
        { id: 'entry_a', name: 'entry-a', kind: 'node', transport: 'connect' },
        { id: 'target_1', name: 'target', kind: 'web', transport: 'https' }
      ]
    },
    ...overrides
  };
}

class FakeInput extends EventEmitter {
  isTTY = true;
  rawMode = false;

  setRawMode(value) {
    this.rawMode = value;
  }

  resume() {}

  pause() {}
}

class FakeOutput extends EventEmitter {
  isTTY = true;

  constructor(columns, rows) {
    super();
    this.columns = columns;
    this.rows = rows;
    this.chunks = [];
  }

  write(chunk) {
    this.chunks.push(String(chunk));
    return true;
  }
}

class FakePty extends EventEmitter {
  constructor() {
    super();
    this.writes = [];
    this.resizes = [];
  }

  write(chunk) {
    this.writes.push(chunk);
  }

  resize(columns, rows) {
    this.resizes.push({ columns, rows });
  }

  onData(handler) {
    this.on('data', handler);
    return { dispose: () => this.off('data', handler) };
  }

  onExit(handler) {
    this.on('exit', handler);
    return { dispose: () => this.off('exit', handler) };
  }

  exit(exitCode, signal = null) {
    this.emit('exit', { exitCode, signal });
  }
}

test('capability detection allows interactive linux and rejects unsupported terminals', () => {
  const base = {
    requested: true,
    json: false,
    platform: 'linux',
    env: { TERM: 'xterm-256color' },
    stdin: { isTTY: true },
    stdout: { isTTY: true, columns: 100, rows: 20 },
    stderr: { isTTY: true },
    ptyAvailable: true
  };

  assert.deepEqual(detectTuiCapability(base), { available: true, reason: null, color: true });
  assert.equal(detectTuiCapability({ ...base, stdout: { ...base.stdout, isTTY: false } }).available, false);
  assert.equal(detectTuiCapability({ ...base, env: { TERM: 'dumb' } }).reason, 'dumb_terminal');
  assert.equal(detectTuiCapability({ ...base, platform: 'win32' }).reason, 'unsupported_platform');
  assert.equal(detectTuiCapability({ ...base, stdout: { ...base.stdout, rows: 9 } }).reason, 'terminal_too_small');
  assert.equal(detectTuiCapability({ ...base, ptyAvailable: false }).reason, 'pty_unavailable');
  assert.equal(detectTuiCapability({ ...base, json: true }).reason, 'json_output');
  assert.equal(detectTuiCapability({ ...base, env: { TERM: 'xterm-256color', NO_COLOR: '1' } }).color, false);
});

test('footer rows follow terminal height thresholds', () => {
  assert.equal(footerRowsForTerminalHeight(30), 3);
  assert.equal(footerRowsForTerminalHeight(18), 3);
  assert.equal(footerRowsForTerminalHeight(17), 2);
  assert.equal(footerRowsForTerminalHeight(14), 2);
  assert.equal(footerRowsForTerminalHeight(13), 1);
  assert.equal(footerRowsForTerminalHeight(10), 1);
  assert.equal(footerRowsForTerminalHeight(9), 0);
});

test('footer formatter right-aligns traffic and keeps labels out of status lines', () => {
  const footer = formatFooter(status(), { columns: 64, footerRows: 3, color: false });

  assert.equal(footer.lines.length, 3);
  assert.match(footer.lines[0], /^stanley@example\.com  demo  32ms$/);
  assert.doesNotMatch(footer.lines[0], /Account:|Tenant:|Ping:/);
  assert.match(footer.lines[1], /^ +Total ↑ 12\.4 MB \| ↓ 91\.8 MB$/);
  assert.equal(visibleWidth(footer.lines[1]), 64);
  assert.equal(footer.lines[2], 'user-entry-a-target');
  assert.doesNotMatch(footer.lines[2], /Path:/);
});

test('footer formatter truncates by visible width when ANSI color is enabled', () => {
  const footer = formatFooter(status({
    account: 'very-long-account-name@example.com',
    tenant: 'very-long-tenant-name',
    pingMs: 301
  }), { columns: 24, footerRows: 3, color: true });

  assert.equal(footer.lines.every((line) => visibleWidth(line) <= 24), true);
  assert.equal(footer.lines.some((line) => /\x1b\[[0-9;]*m/.test(line)), true);
  assert.equal(stripAnsi(footer.lines[0]).includes('Account:'), false);
});

test('latency color thresholds map to contract styles', () => {
  assert.equal(latencyStyleName(null), 'gray');
  assert.equal(latencyStyleName(99), 'mint');
  assert.equal(latencyStyleName(100), 'butter');
  assert.equal(latencyStyleName(299), 'butter');
  assert.equal(latencyStyleName(300), 'coral');
});

test('path text uses node names, falls back to ids, and omits empty nodes', () => {
  assert.equal(formatPathText({
    nodes: [
      { id: 'user_1', name: 'user', kind: 'user', transport: 'local' },
      { id: 'entry_1', name: '', kind: 'node', transport: 'connect' },
      { id: '', name: '', kind: 'node', transport: 'connect' },
      { id: 'target_1', name: 'target', kind: 'web', transport: 'https' }
    ]
  }), 'user-entry_1-target');
});

test('runtime sizes fake PTY, resizes on terminal changes, and propagates exit code', async () => {
  const stdin = new FakeInput();
  const stdout = new FakeOutput(100, 20);
  const stderr = new FakeOutput(100, 20);
  const pty = new FakePty();
  let spawnRequest = null;
  const ptyAdapter = {
    spawn(command, args, options) {
      spawnRequest = { command, args, options };
      return pty;
    }
  };

  const session = runTuiSession({
    command: 'ssh',
    args: ['example.com'],
    env: { ONEPROXY_ACTIVE: '1' },
    status: status(),
    stdin,
    stdout,
    stderr,
    ptyAdapter
  });

  assert.equal(spawnRequest.command, 'ssh');
  assert.deepEqual(spawnRequest.args, ['example.com']);
  assert.equal(spawnRequest.options.cols, 100);
  assert.equal(spawnRequest.options.rows, 17);

  stdout.columns = 80;
  stdout.rows = 15;
  stdout.emit('resize');
  assert.deepEqual(pty.resizes.at(-1), { columns: 80, rows: 13 });

  pty.exit(7);
  assert.equal(await session, 7);
});

test('runtime returns 1 when fake PTY exits by signal', async () => {
  const stdin = new FakeInput();
  const stdout = new FakeOutput(100, 20);
  const stderr = new FakeOutput(100, 20);
  const pty = new FakePty();
  const session = runTuiSession({
    command: 'shell',
    args: [],
    env: {},
    status: status(),
    stdin,
    stdout,
    stderr,
    ptyAdapter: {
      spawn() {
        return pty;
      }
    }
  });

  pty.exit(0, 'SIGTERM');
  assert.equal(await session, 1);
});

test('runtime warns when default TUI cannot start', async () => {
  const previous = {
    stdin: process.stdin.isTTY,
    stdout: process.stdout.isTTY,
    stdoutRows: process.stdout.rows,
    stderr: process.stderr.isTTY,
    term: process.env.TERM
  };
  const originalWrite = process.stderr.write;
  let stderr = '';
  Object.defineProperty(process.stdin, 'isTTY', { value: true, configurable: true });
  Object.defineProperty(process.stdout, 'isTTY', { value: true, configurable: true });
  Object.defineProperty(process.stdout, 'rows', { value: 20, configurable: true });
  Object.defineProperty(process.stderr, 'isTTY', { value: true, configurable: true });
  process.env.TERM = 'dumb';
  process.stderr.write = ((chunk, encoding, callback) => {
    stderr += String(chunk);
    if (typeof encoding === 'function') {
      encoding();
    } else if (callback) {
      callback();
    }
    return true;
  });
  try {
    const result = await runTuiRuntime({
      command: { executable: 'node', args: [] },
      snapshot: status(),
      requested: true,
      interactive: true,
      json: false
    });

    assert.deepEqual(result, { ran: false });
    assert.equal(stderr, `${tuiUnavailableWarning}\n`);
  } finally {
    Object.defineProperty(process.stdin, 'isTTY', { value: previous.stdin, configurable: true });
    Object.defineProperty(process.stdout, 'isTTY', { value: previous.stdout, configurable: true });
    Object.defineProperty(process.stdout, 'rows', { value: previous.stdoutRows, configurable: true });
    Object.defineProperty(process.stderr, 'isTTY', { value: previous.stderr, configurable: true });
    if (previous.term === undefined) {
      delete process.env.TERM;
    } else {
      process.env.TERM = previous.term;
    }
    process.stderr.write = originalWrite;
  }
});
