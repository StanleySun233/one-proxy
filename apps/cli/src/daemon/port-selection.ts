import * as net from 'node:net';

export type PortSelection = {
  candidatePorts: number[];
  selectedPair: [number, number];
  excludedCommonPorts: number[];
};

export const excludedCommonPorts = [
  20, 21, 22, 25, 53, 80, 110, 143, 443, 3306, 5432, 6379, 8080
];

const excludedPortSet = new Set(excludedCommonPorts);
const defaultStartPort = 10000;
const defaultEndPort = 60999;
const loopbackHost = '127.0.0.1';
const scanBatchSize = 256;

export async function selectProxyPorts(): Promise<PortSelection> {
  const candidatePorts = await scanAvailableCandidatePorts();
  const pairs = consecutivePairs(candidatePorts);
  if (pairs.length === 0) {
    throw new Error('No available consecutive loopback proxy port pair');
  }
  return {
    candidatePorts,
    selectedPair: pairs[Math.floor(Math.random() * pairs.length)],
    excludedCommonPorts
  };
}

export async function scanAvailableCandidatePorts(start = defaultStartPort, end = defaultEndPort) {
  const candidatePorts: number[] = [];
  for (let batchStart = start; batchStart <= end; batchStart += scanBatchSize) {
    const batchEnd = Math.min(batchStart + scanBatchSize - 1, end);
    const ports = Array.from({ length: batchEnd - batchStart + 1 }, (_, index) => batchStart + index);
    const results = await Promise.all(ports.map(async (port) => ({
      port,
      usable: await isUsablePort(port)
    })));
    for (const result of results) {
      if (result.usable) {
        candidatePorts.push(result.port);
      }
    }
  }
  return candidatePorts;
}

export async function isUsablePort(port: number) {
  if (!Number.isInteger(port) || port < 1 || port > 65535 || excludedPortSet.has(port)) {
    return false;
  }
  const server = net.createServer();
  return await new Promise<boolean>((resolve) => {
    server.once('error', () => resolve(false));
    server.listen(port, loopbackHost, () => {
      server.close(() => resolve(true));
    });
  });
}

function consecutivePairs(candidatePorts: number[]) {
  const pairs: Array<[number, number]> = [];
  const candidates = new Set(candidatePorts);
  for (const port of candidatePorts) {
    if (candidates.has(port + 1)) {
      pairs.push([port, port + 1]);
    }
  }
  return pairs;
}
