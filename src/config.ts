import {readFile} from 'node:fs/promises';
import {createLogger} from './logger.js';

const logger = createLogger('config');

export type ImapAuth = {
  user: string;
  pass: string;
};

export type SourceConfig = {
  name: string;
  host: string;
  port: number;
  secure: boolean;
  auth: ImapAuth;
  folders: string[];
  deleteAfterForward: boolean;
};

export type TargetConfig = {
  host: string;
  port: number;
  secure: boolean;
  auth: ImapAuth;
};

export type HealthCheckConfig = {
  port: number;
};

export type Config = {
  target: TargetConfig;
  sources: SourceConfig[];
  healthCheck?: HealthCheckConfig;
};

function validateAuth(auth: unknown, path: string): asserts auth is ImapAuth {
  if (!auth || typeof auth !== 'object') {
    throw new Error(`${path}.auth must be an object`);
  }

  const a = auth as Record<string, unknown>;
  if (typeof a.user !== 'string' || a.user.length === 0) {
    throw new Error(`${path}.auth.user must be a non-empty string`);
  }

  if (typeof a.pass !== 'string' || a.pass.length === 0) {
    throw new Error(`${path}.auth.pass must be a non-empty string`);
  }
}

function validateTarget(target: unknown): asserts target is TargetConfig {
  if (!target || typeof target !== 'object') {
    throw new Error('config.target must be an object');
  }

  const t = target as Record<string, unknown>;
  if (typeof t.host !== 'string' || t.host.length === 0) {
    throw new Error('config.target.host must be a non-empty string');
  }

  if (typeof t.port !== 'number' || !Number.isInteger(t.port)) {
    throw new TypeError('config.target.port must be an integer');
  }

  if (typeof t.secure !== 'boolean') {
    throw new TypeError('config.target.secure must be a boolean');
  }

  validateAuth(t.auth, 'config.target');
}

function validateSource(
  source: unknown,
  index: number,
): asserts source is SourceConfig {
  const path = `config.sources[${index}]`;
  if (!source || typeof source !== 'object') {
    throw new Error(`${path} must be an object`);
  }

  const s = source as Record<string, unknown>;
  if (typeof s.name !== 'string' || s.name.length === 0) {
    throw new Error(`${path}.name must be a non-empty string`);
  }

  if (typeof s.host !== 'string' || s.host.length === 0) {
    throw new Error(`${path}.host must be a non-empty string`);
  }

  if (typeof s.port !== 'number' || !Number.isInteger(s.port)) {
    throw new TypeError(`${path}.port must be an integer`);
  }

  if (typeof s.secure !== 'boolean') {
    throw new TypeError(`${path}.secure must be a boolean`);
  }

  validateAuth(s.auth, path);

  if (
    s.folders !== undefined &&
    (!Array.isArray(s.folders) ||
      s.folders.length === 0 ||
      !s.folders.every((f: unknown) => typeof f === 'string'))
  ) {
    throw new Error(`${path}.folders must be a non-empty array of strings`);
  }
}

export function validateConfig(raw: unknown): Config {
  if (!raw || typeof raw !== 'object') {
    throw new Error('Config must be a JSON object');
  }

  const config = raw as Record<string, unknown>;

  validateTarget(config.target);

  if (!Array.isArray(config.sources) || config.sources.length === 0) {
    throw new Error('config.sources must be a non-empty array');
  }

  for (const [index, source] of config.sources.entries()) {
    validateSource(source, index);
  }

  if (config.healthCheck !== undefined && config.healthCheck !== null) {
    if (typeof config.healthCheck !== 'object') {
      throw new TypeError('config.healthCheck must be an object');
    }

    const hc = config.healthCheck as Record<string, unknown>;
    if (typeof hc.port !== 'number' || !Number.isInteger(hc.port)) {
      throw new TypeError('config.healthCheck.port must be an integer');
    }
  }

  // Apply defaults
  const sources: SourceConfig[] = (config.sources as unknown[]).map((s) => {
    const source = s as Record<string, unknown>;
    return {
      ...(source as unknown as SourceConfig),
      folders: (source.folders as string[] | undefined) ?? ['INBOX'],
      deleteAfterForward:
        (source.deleteAfterForward as boolean | undefined) ?? false,
    };
  });

  return {
    target: config.target,
    sources,
    healthCheck: config.healthCheck as HealthCheckConfig | undefined,
  };
}

export async function loadConfig(configPath: string): Promise<Config> {
  logger.info(`Loading configuration from ${configPath}`);

  let content: string;
  try {
    content = await readFile(configPath, 'utf8');
  } catch (error) {
    throw new Error(
      `Failed to read config file "${configPath}": ${(error as Error).message}`,
    );
  }

  let raw: unknown;
  try {
    raw = JSON.parse(content);
  } catch {
    throw new Error(
      `Failed to parse config file "${configPath}": invalid JSON`,
    );
  }

  const config = validateConfig(raw);
  logger.info(
    `Configuration loaded: ${config.sources.length} source(s) configured`,
  );
  return config;
}
