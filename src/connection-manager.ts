import type { Config } from './config.js';
import { Forwarder, type ForwarderStatus } from './forwarder.js';
import { createLogger } from './logger.js';

const logger = createLogger('manager');

export class ConnectionManager {
  private readonly forwarders: Forwarder[] = [];
  private readonly statuses = new Map<string, ForwarderStatus>();

  constructor(config: Config) {
    for (const source of config.sources) {
      const forwarder = new Forwarder(source, config.target, {
        onStatusChange: (status) => {
          this.statuses.set(status.name, status);
        },
      });
      this.forwarders.push(forwarder);
    }
  }

  async startAll(): Promise<void> {
    logger.info(`Starting ${this.forwarders.length} forwarder(s)...`);
    await Promise.all(this.forwarders.map(async (f) => f.start()));
  }

  async stopAll(): Promise<void> {
    logger.info('Stopping all forwarders...');
    await Promise.all(this.forwarders.map(async (f) => f.stop()));
    logger.info('All forwarders stopped');
  }

  getStatuses(): ForwarderStatus[] {
    return this.forwarders.map((f) => f.getStatus());
  }

  getOverallStatus(): 'ok' | 'degraded' | 'error' {
    const statuses = this.getStatuses();
    const connectedCount = statuses.filter((s) => s.connected).length;

    if (connectedCount === statuses.length) return 'ok';
    if (connectedCount > 0) return 'degraded';
    return 'error';
  }
}
