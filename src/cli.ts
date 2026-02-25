#!/usr/bin/env node

import process from 'node:process';
import { parseArgs } from 'node:util';
import { readFileSync } from 'node:fs';
import type { Server } from 'node:http';
import { loadConfig } from './config.js';
import { ConnectionManager } from './connection-manager.js';
import { createHealthServer } from './health.js';
import { createLogger, setLogLevel, type LogLevel } from './logger.js';

const logger = createLogger('cli');

function getVersion(): string {
  try {
    const packageJson = JSON.parse(
      readFileSync(new URL('../package.json', import.meta.url), 'utf8'),
    ) as { version: string };
    return packageJson.version;
  } catch {
    return 'unknown';
  }
}

function printHelp(): void {
  console.log(`
imapforward - IMAP email forwarder

Usage: imapforward [options]

Options:
  -c, --config <path>   Path to config file (default: config.json)
  -l, --log-level <lvl> Log level: debug, info, warn, error (default: info)
  -v, --version         Show version
  -h, --help            Show this help
`);
}

async function main(): Promise<void> {
  const { values } = parseArgs({
    options: {
      config: { type: 'string', short: 'c', default: 'config.json' },
      'log-level': { type: 'string', short: 'l', default: 'info' },
      version: { type: 'boolean', short: 'v', default: false },
      help: { type: 'boolean', short: 'h', default: false },
    },
    strict: true,
  });

  if (values.help) {
    printHelp();
    process.exit(0);
  }

  if (values.version) {
    console.log(getVersion());
    process.exit(0);
  }

  const logLevel = (
    process.env.LOG_LEVEL ??
    values['log-level'] ??
    'info'
  ).toLowerCase() as LogLevel;
  setLogLevel(logLevel);

  const configPath =
    process.env.IMAPFORWARD_CONFIG ?? values.config ?? 'config.json';

  let config;
  try {
    config = await loadConfig(configPath);
  } catch (error) {
    logger.error('Configuration error', error);
    process.exit(1);
  }

  const manager = new ConnectionManager(config);

  const healthServer = createHealthServer(manager, config.healthCheck.port);

  // Graceful shutdown
  const shutdown = async () => {
    logger.info('Shutting down...');
    await manager.stopAll();

    await new Promise<void>((resolve) => {
      healthServer.close(() => {
        resolve();
      });
    });

    logger.info('Shutdown complete');
    process.exit(0);
  };

  process.on('SIGTERM', () => {
    void shutdown();
  });
  process.on('SIGINT', () => {
    void shutdown();
  });

  logger.info(`imapforward v${getVersion()} starting`);
  await manager.startAll();
}

await main();
