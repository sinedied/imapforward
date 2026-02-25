import {describe, it, expect, vi, beforeEach} from 'vitest';
import type {SourceConfig, TargetConfig} from './config.js';

// Mock imapflow
vi.mock('imapflow', () => {
  const mockClient = {
    connect: vi.fn(),
    logout: vi.fn(),
    getMailboxLock: vi.fn(() => ({release: vi.fn()})),
    fetch: vi.fn(function* () {
      // No messages
    }),
    idle: vi.fn(
      async () =>
        new Promise(() => {
          // Never resolves (simulates waiting)
        }),
    ),
    messageFlagsAdd: vi.fn(),
    messageDelete: vi.fn(),
    usable: true,
    on: vi.fn(),
  };

  return {ImapFlow: vi.fn(() => mockClient)};
});

// Mock nodemailer
vi.mock('nodemailer', () => ({
  createTransport: vi.fn(() => ({
    sendMail: vi.fn(),
    options: {auth: {user: 'target@gmail.com'}},
  })),
}));

const {Forwarder} = await import('./forwarder.js');

const testSource: SourceConfig = {
  name: 'Test Source',
  host: 'imap.test.com',
  port: 993,
  secure: true,
  auth: {user: 'test@test.com', pass: 'pass'},
  folders: ['INBOX'],
  deleteAfterForward: false,
};

const testTarget: TargetConfig = {
  host: 'smtp.gmail.com',
  port: 465,
  secure: true,
  auth: {user: 'target@gmail.com', pass: 'pass'},
};

describe('forwarder', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should create a forwarder instance', () => {
    const forwarder = new Forwarder(testSource, testTarget);
    expect(forwarder).toBeDefined();
  });

  it('should return disconnected status before start', () => {
    const forwarder = new Forwarder(testSource, testTarget);
    const status = forwarder.getStatus();
    expect(status.name).toBe('Test Source');
    expect(status.connected).toBe(false);
    expect(status.lastSync).toBeUndefined();
  });

  it('should stop cleanly without starting', async () => {
    const forwarder = new Forwarder(testSource, testTarget);
    await forwarder.stop();
    const status = forwarder.getStatus();
    expect(status.connected).toBe(false);
  });
});
