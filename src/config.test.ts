import { describe, it, expect } from 'vitest';
import { validateConfig } from './config.js';

const validTarget = {
  host: 'smtp.gmail.com',
  port: 465,
  secure: true,
  auth: { user: 'user@gmail.com', pass: 'password' },
};

const validSource = {
  name: 'Work Email',
  host: 'imap.example.com',
  port: 993,
  secure: true,
  auth: { user: 'user@example.com', pass: 'password' },
};

describe('validateConfig', () => {
  it('should accept a valid minimal config', () => {
    const config = validateConfig({
      target: validTarget,
      sources: [validSource],
    });

    expect(config.target.host).toBe('smtp.gmail.com');
    expect(config.sources).toHaveLength(1);
    expect(config.sources[0].name).toBe('Work Email');
  });

  it('should apply default folders to INBOX', () => {
    const config = validateConfig({
      target: validTarget,
      sources: [validSource],
    });

    expect(config.sources[0].folders).toEqual(['INBOX']);
  });

  it('should apply default deleteAfterForward to false', () => {
    const config = validateConfig({
      target: validTarget,
      sources: [validSource],
    });

    expect(config.sources[0].deleteAfterForward).toBe(false);
  });

  it('should preserve custom folders', () => {
    const config = validateConfig({
      target: validTarget,
      sources: [{ ...validSource, folders: ['INBOX', 'Sent'] }],
    });

    expect(config.sources[0].folders).toEqual(['INBOX', 'Sent']);
  });

  it('should preserve deleteAfterForward: true', () => {
    const config = validateConfig({
      target: validTarget,
      sources: [{ ...validSource, deleteAfterForward: true }],
    });

    expect(config.sources[0].deleteAfterForward).toBe(true);
  });

  it('should accept multiple sources', () => {
    const config = validateConfig({
      target: validTarget,
      sources: [validSource, { ...validSource, name: 'Personal Email' }],
    });

    expect(config.sources).toHaveLength(2);
  });

  it('should accept healthCheck config', () => {
    const config = validateConfig({
      target: validTarget,
      sources: [validSource],
      healthCheck: { port: 8080 },
    });

    expect(config.healthCheck?.port).toBe(8080);
  });

  it('should reject non-object config', () => {
    expect(() => validateConfig(null)).toThrow('Config must be a JSON object');
    expect(() => validateConfig('string')).toThrow(
      'Config must be a JSON object',
    );
  });

  it('should reject missing target', () => {
    expect(() => validateConfig({ sources: [validSource] })).toThrow(
      'config.target must be an object',
    );
  });

  it('should reject missing sources', () => {
    expect(() => validateConfig({ target: validTarget })).toThrow(
      'config.sources must be a non-empty array',
    );
  });

  it('should reject empty sources array', () => {
    expect(() => validateConfig({ target: validTarget, sources: [] })).toThrow(
      'config.sources must be a non-empty array',
    );
  });

  it('should reject target with missing host', () => {
    expect(() =>
      validateConfig({
        target: { ...validTarget, host: '' },
        sources: [validSource],
      }),
    ).toThrow('config.target.host must be a non-empty string');
  });

  it('should reject target with missing auth', () => {
    const { auth: _, ...targetNoAuth } = validTarget;
    expect(() =>
      validateConfig({
        target: targetNoAuth,
        sources: [validSource],
      }),
    ).toThrow('config.target.auth must be an object');
  });

  it('should reject source with missing name', () => {
    const { name: _, ...sourceNoName } = validSource;
    expect(() =>
      validateConfig({
        target: validTarget,
        sources: [sourceNoName],
      }),
    ).toThrow('config.sources[0].name must be a non-empty string');
  });

  it('should reject source with empty folders', () => {
    expect(() =>
      validateConfig({
        target: validTarget,
        sources: [{ ...validSource, folders: [] }],
      }),
    ).toThrow('config.sources[0].folders must be a non-empty array');
  });

  it('should reject invalid healthCheck port', () => {
    expect(() =>
      validateConfig({
        target: validTarget,
        sources: [validSource],
        healthCheck: { port: 'abc' },
      }),
    ).toThrow('config.healthCheck.port must be an integer');
  });
});
