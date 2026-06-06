import * as net from 'node:net';
import { loopbackHost } from './lifecycle';

export type PortSelection = {
  candidatePorts: number[];
  selectedPair: [number, number];
  excludedCommonPorts: number[];
};

export class InvalidPortPairError extends Error {
  code = 'INVALID_PORT_PAIR';
}

export const excludedCommonPorts = [
  20, 21, 22, 25, 53, 80, 110, 143, 443, 3306, 5432, 6379, 8080
];

const excludedPortSet = new Set(excludedCommonPorts);
const defaultStartPort = 10000;
const defaultEndPort = 60999;

export async function selectProxyPorts(configuredHttp = 0, configuredHttps = 0): Promise<PortSelection> {
  if (configuredHttp || configuredHttps) {
    if (!configuredHttp || !configuredHttps || configuredHttps !== configuredHttp + 1) {
      throw new InvalidPortPairError('Configured HTTP and HTTPS proxy ports must be consecutive');
    }
    const [httpAvailable, httpsAvailable] = await Promise.all([
      isUsablePort(configuredHttp),
      isUsablePort(configuredHttps)
    ]);
    if (!httpAvailable || !httpsAvailable) {
      throw new InvalidPortPairError('Configured HTTP and HTTPS proxy ports are not available');
    }
    return {
      candidatePorts: [configuredHttp, configuredHttps],
      selectedPair: [configuredHttp, configuredHttps],
      excludedCommonPorts
    };
  }

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
  for (let port = start; port <= end; port += 1) {
    if (await isUsablePort(port)) {
      candidatePorts.push(port);
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
