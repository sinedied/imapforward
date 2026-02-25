export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

const levels: Record<LogLevel, number> = {
  debug: 0,
  info: 1,
  warn: 2,
  error: 3,
};

let currentLevel: LogLevel = 'info';

export function setLogLevel(level: LogLevel) {
  currentLevel = level;
}

export function getLogLevel(): LogLevel {
  return currentLevel;
}

function shouldLog(level: LogLevel): boolean {
  return levels[level] >= levels[currentLevel];
}

function timestamp(): string {
  return new Date().toISOString();
}

function formatMessage(
  level: LogLevel,
  context: string | undefined,
  message: string,
): string {
  const prefix = context ? `[${context}]` : '';
  return `${timestamp()} ${level.toUpperCase().padEnd(5)} ${prefix} ${message}`.trimEnd();
}

export type Logger = {
  debug(message: string): void;
  info(message: string): void;
  warn(message: string): void;
  error(message: string, error?: unknown): void;
};

export function createLogger(context?: string): Logger {
  return {
    debug(message: string) {
      if (shouldLog('debug')) {
        console.debug(formatMessage('debug', context, message));
      }
    },
    info(message: string) {
      if (shouldLog('info')) {
        console.info(formatMessage('info', context, message));
      }
    },
    warn(message: string) {
      if (shouldLog('warn')) {
        console.warn(formatMessage('warn', context, message));
      }
    },
    error(message: string, error?: unknown) {
      if (shouldLog('error')) {
        const errorDetails = error instanceof Error ? `: ${error.message}` : '';
        console.error(formatMessage('error', context, message + errorDetails));
      }
    },
  };
}
