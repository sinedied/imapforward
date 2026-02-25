import { createServer, type Server } from 'node:http';
import type { ConnectionManager } from './connection-manager.js';
import { createLogger } from './logger.js';

const logger = createLogger('health');

export function createHealthServer(
  manager: ConnectionManager,
  port: number,
): Server {
  const server = createServer((request, response) => {
    if (request.url === '/health' && request.method === 'GET') {
      const status = manager.getOverallStatus();
      const statuses = manager.getStatuses();
      const statusCode = status === 'error' ? 503 : 200;

      response.writeHead(statusCode, { 'Content-Type': 'application/json' });
      response.end(
        JSON.stringify({
          status,
          sources: statuses,
        }),
      );
    } else {
      response.writeHead(404, { 'Content-Type': 'text/plain' });
      response.end('Not Found');
    }
  });

  server.listen(port, () => {
    logger.info(`Health check server listening on port ${port}`);
  });

  return server;
}
