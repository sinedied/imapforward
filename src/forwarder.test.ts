import { Buffer } from 'node:buffer';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { SourceConfig, TargetConfig } from './config.js';

const mockClient = {
  connect: vi.fn(),
  logout: vi.fn(),
  getMailboxLock: vi.fn(() => ({ release: vi.fn() })),
  fetch: vi.fn(function* () {
    // No messages
  }),
  idle: vi.fn(
    async () =>
      new Promise(() => {
        // Never resolves (simulates waiting)
      }),
  ),
  mailboxCreate: vi.fn(),
  append: vi.fn(),
  messageFlagsAdd: vi.fn(),
  messageDelete: vi.fn(),
  usable: true,
  on: vi.fn(),
  off: vi.fn(),
  list: vi.fn(async () => []),
};

// Mock imapflow with a constructible class
vi.mock('imapflow', () => {
  return {
    ImapFlow: class {
      connect = mockClient.connect;
      logout = mockClient.logout;
      getMailboxLock = mockClient.getMailboxLock;
      fetch = mockClient.fetch;
      idle = mockClient.idle;
      mailboxCreate = mockClient.mailboxCreate;
      append = mockClient.append;
      messageFlagsAdd = mockClient.messageFlagsAdd;
      messageDelete = mockClient.messageDelete;
      usable = mockClient.usable;
      on = mockClient.on;
      off = mockClient.off;
      list = mockClient.list;
    },
  };
});

const { Forwarder } = await import('./forwarder.js');

const testSource: SourceConfig = {
  name: 'Test Source',
  host: 'imap.test.com',
  port: 993,
  secure: true,
  auth: { user: 'test@test.com', pass: 'pass' },
  folders: ['INBOX'],
  deleteAfterForward: false,
};

const testTarget: TargetConfig = {
  host: 'imap.gmail.com',
  port: 993,
  secure: true,
  auth: { user: 'target@gmail.com', pass: 'pass' },
  folder: 'INBOX',
};

describe('forwarder', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockClient.usable = true;
    mockClient.connect.mockResolvedValue(undefined);
    mockClient.logout.mockResolvedValue(undefined);
    mockClient.mailboxCreate.mockResolvedValue({
      path: 'Free',
      created: true,
    });
    mockClient.append.mockResolvedValue({ destination: 'INBOX', uid: 999 });
    mockClient.messageFlagsAdd.mockResolvedValue(undefined);
    mockClient.messageDelete.mockResolvedValue(undefined);
    mockClient.list.mockResolvedValue([]);
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

  it('should append to source.targetFolder when provided', async () => {
    const sourceWithTargetFolder: SourceConfig = {
      ...testSource,
      targetFolder: 'Free',
    };

    const forwarder = new Forwarder(sourceWithTargetFolder, testTarget);

    // Simulate a connected source client
    (forwarder as any).client = mockClient;

    const message = {
      uid: 123,
      source: Buffer.from('Subject: Test\r\n\r\nHello'),
      flags: new Set(),
    };

    await (forwarder as any).forwardMessage(message);

    expect(mockClient.mailboxCreate).toHaveBeenCalledWith('Free');
    expect(mockClient.append).toHaveBeenCalledWith('Free', message.source);
  });

  it('should append to target.folder when source.targetFolder is not provided', async () => {
    const forwarder = new Forwarder(testSource, testTarget);

    // Simulate a connected source client
    (forwarder as any).client = mockClient;

    const message = {
      uid: 124,
      source: Buffer.from('Subject: Test\r\n\r\nHello'),
      flags: new Set(),
    };

    await (forwarder as any).forwardMessage(message);

    expect(mockClient.mailboxCreate).not.toHaveBeenCalled();
    expect(mockClient.append).toHaveBeenCalledWith('INBOX', message.source);
  });

  it('should append when target folder already exists', async () => {
    const sourceWithTargetFolder: SourceConfig = {
      ...testSource,
      targetFolder: 'Free',
    };
    mockClient.mailboxCreate.mockResolvedValue({
      path: 'Free',
      created: false,
    });

    const forwarder = new Forwarder(sourceWithTargetFolder, testTarget);
    (forwarder as any).client = mockClient;

    const message = {
      uid: 125,
      source: Buffer.from('Subject: Test\r\n\r\nHello'),
      flags: new Set(),
    };

    await (forwarder as any).forwardMessage(message);

    expect(mockClient.mailboxCreate).toHaveBeenCalledWith('Free');
    expect(mockClient.append).toHaveBeenCalledWith('Free', message.source);
  });

  it('should create a target folder only once per forwarder', async () => {
    const sourceWithTargetFolder: SourceConfig = {
      ...testSource,
      targetFolder: 'Free',
    };

    const forwarder = new Forwarder(sourceWithTargetFolder, testTarget);
    (forwarder as any).client = mockClient;

    const firstMessage = {
      uid: 126,
      source: Buffer.from('Subject: Test\r\n\r\nHello'),
      flags: new Set(),
    };
    const secondMessage = {
      uid: 127,
      source: Buffer.from('Subject: Test 2\r\n\r\nHello again'),
      flags: new Set(),
    };

    await (forwarder as any).forwardMessage(firstMessage);
    await (forwarder as any).forwardMessage(secondMessage);

    expect(mockClient.mailboxCreate).toHaveBeenCalledTimes(1);
    expect(mockClient.append).toHaveBeenNthCalledWith(
      1,
      'Free',
      firstMessage.source,
    );
    expect(mockClient.append).toHaveBeenNthCalledWith(
      2,
      'Free',
      secondMessage.source,
    );
  });

  it('should not append when target folder creation fails', async () => {
    const sourceWithTargetFolder: SourceConfig = {
      ...testSource,
      targetFolder: 'Free',
    };
    mockClient.mailboxCreate.mockRejectedValue(
      new Error('Mailbox creation failed'),
    );

    const forwarder = new Forwarder(sourceWithTargetFolder, testTarget);
    (forwarder as any).client = mockClient;

    const message = {
      uid: 128,
      source: Buffer.from('Subject: Test\r\n\r\nHello'),
      flags: new Set(),
    };

    await (forwarder as any).forwardMessage(message);

    expect(mockClient.mailboxCreate).toHaveBeenCalledWith('Free');
    expect(mockClient.append).not.toHaveBeenCalled();
    expect(mockClient.messageFlagsAdd).not.toHaveBeenCalled();
  });
});
