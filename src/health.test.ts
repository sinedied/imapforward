import http from 'node:http';
import { describe, it, expect, vi, afterEach } from 'vitest';
import { createHealthServer } from './health.js';
import type { ConnectionManager } from './connection-manager.js';

async function makeRequest(
  port: number,
  path: string,
): Promise<{ status: number; body: string }> {
  return new Promise((resolve, reject) => {
    http
      .get(`http://localhost:${port}${path}`, (response) => {
        let body = '';
        response.on('data', (chunk: string) => {
          body += chunk;
        });
        response.on('end', () => {
          resolve({ status: response.statusCode ?? 0, body });
        });
      })
      .on('error', reject);
  });
}

function getPort(server: http.Server): number {
  const address = server.address();
  return typeof address === 'object' ? (address?.port ?? 0) : 0;
}

describe('health server', () => {
  let server: http.Server;

  afterEach(async () => {
    if (server) {
      await new Promise<void>((resolve) => {
        server.close(() => {
          resolve();
        });
      });
    }
  });

  it('should return 200 with ok status when all sources connected', async () => {
    const mockManager = {
      getOverallStatus: vi.fn(() => 'ok'),
      getStatuses: vi.fn(() => [
        {
          name: 'Test',
          connected: true,
          lastSync: undefined,
          error: undefined,
        },
      ]),
    } as unknown as ConnectionManager;

    server = createHealthServer(mockManager, 0);
    await new Promise((resolve) => {
      server.on('listening', resolve);
    });

    const { status, body } = await makeRequest(getPort(server), '/health');
    expect(status).toBe(200);

    const data = JSON.parse(body) as { status: string };
    expect(data.status).toBe('ok');
  });

  it('should return 503 when all sources errored', async () => {
    const mockManager = {
      getOverallStatus: vi.fn(() => 'error'),
      getStatuses: vi.fn(() => [
        {
          name: 'Test',
          connected: false,
          lastSync: undefined,
          error: 'Connection refused',
        },
      ]),
    } as unknown as ConnectionManager;

    server = createHealthServer(mockManager, 0);
    await new Promise((resolve) => {
      server.on('listening', resolve);
    });

    const { status } = await makeRequest(getPort(server), '/health');
    expect(status).toBe(503);
  });

  it('should return 404 for unknown paths', async () => {
    const mockManager = {
      getOverallStatus: vi.fn(() => 'ok'),
      getStatuses: vi.fn(() => []),
    } as unknown as ConnectionManager;

    server = createHealthServer(mockManager, 0);
    await new Promise((resolve) => {
      server.on('listening', resolve);
    });

    const { status } = await makeRequest(getPort(server), '/unknown');
    expect(status).toBe(404);
  });
});
