import { ImapFlow, type FetchMessageObject } from 'imapflow';
import { createTransport, type Transporter } from 'nodemailer';
import type { SourceConfig, TargetConfig } from './config.js';
import { createLogger } from './logger.js';

const forwardedFlag = '$Forwarded';
const reconnectBaseDelay = 1000;
const reconnectMaxDelay = 60_000;

export type ForwarderStatus = {
  name: string;
  connected: boolean;
  lastSync: string | undefined;
  error: string | undefined;
};

export type ForwarderEvents = {
  onStatusChange?: (status: ForwarderStatus) => void;
};

export class Forwarder {
  private readonly targetUser: string;
  private readonly transporter: Transporter;
  private readonly logger;
  private client: ImapFlow | undefined;
  private running = false;
  private reconnectDelay = reconnectBaseDelay;
  private reconnectTimer: ReturnType<typeof setTimeout> | undefined;
  private lastSync: string | undefined;
  private lastError: string | undefined;
  private stopResolve: (() => void) | undefined;

  constructor(
    private readonly source: SourceConfig,
    target: TargetConfig,
    private readonly events: ForwarderEvents = {},
  ) {
    this.targetUser = target.auth.user;
    this.logger = createLogger(source.name);

    this.transporter = createTransport({
      host: target.host,
      port: target.port,
      secure: target.secure,
      auth: {
        user: target.auth.user,
        pass: target.auth.pass,
      },
    });
  }

  getStatus(): ForwarderStatus {
    return {
      name: this.source.name,
      connected: this.client?.usable ?? false,
      lastSync: this.lastSync,
      error: this.lastError,
    };
  }

  async start(): Promise<void> {
    this.running = true;
    this.lastError = undefined;
    await this.connect();
  }

  async stop(): Promise<void> {
    this.running = false;

    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = undefined;
    }

    // Signal any waiting watchFolder to stop
    this.stopResolve?.();

    if (this.client) {
      try {
        await this.client.logout();
      } catch {
        // Ignore logout errors during shutdown
      }

      this.client = undefined;
    }

    this.notifyStatus();
    this.logger.info('Stopped');
  }

  private async connect(): Promise<void> {
    if (!this.running) return;

    try {
      this.client = new ImapFlow({
        host: this.source.host,
        port: this.source.port,
        secure: this.source.secure,
        auth: {
          user: this.source.auth.user,
          pass: this.source.auth.pass,
        },
        logger: false,
      });

      this.client.on('close', () => {
        this.logger.warn('Connection closed');
        this.stopResolve?.();
        this.notifyStatus();
        void this.scheduleReconnect();
      });

      this.client.on('error', (error: Error) => {
        this.lastError = error.message;
        this.logger.error('Connection error', error);
        this.notifyStatus();
      });

      await this.client.connect();
      this.logger.info(`Connected to ${this.source.host}:${this.source.port}`);
      this.reconnectDelay = reconnectBaseDelay;
      this.lastError = undefined;
      this.notifyStatus();

      await this.listFolders();
      await this.processAllFolders();
    } catch (error) {
      this.lastError = (error as Error).message;
      this.logger.error('Failed to connect', error);
      this.notifyStatus();
      await this.scheduleReconnect();
    }
  }

  private async scheduleReconnect(): Promise<void> {
    if (!this.running) return;

    this.logger.info(
      `Reconnecting in ${Math.round(this.reconnectDelay / 1000)}s...`,
    );

    await new Promise<void>((resolve) => {
      this.reconnectTimer = setTimeout(() => {
        this.reconnectTimer = undefined;
        resolve();
      }, this.reconnectDelay);
    });

    this.reconnectDelay = Math.min(this.reconnectDelay * 2, reconnectMaxDelay);

    await this.connect();
  }

  private async listFolders(): Promise<void> {
    if (!this.client?.usable) return;

    try {
      const mailboxes = await this.client.list();
      const folderNames = mailboxes.map((m) => m.path);
      this.logger.info(`Available folders: ${folderNames.join(', ')}`);
    } catch (error) {
      this.logger.warn(`Failed to list folders: ${(error as Error).message}`);
    }
  }

  private async processAllFolders(): Promise<void> {
    // Process folders sequentially
    for (const folder of this.source.folders) {
      if (!this.running || !this.client?.usable) return;
      // eslint-disable-next-line no-await-in-loop
      await this.processFolder(folder);
    }
  }

  private async processFolder(folder: string): Promise<void> {
    if (!this.client?.usable) return;

    this.logger.info(`Processing folder: ${folder}`);

    try {
      // Initial processing: forward any existing unseen messages
      const lock = await this.client.getMailboxLock(folder);
      try {
        await this.forwardNewMessages();
      } finally {
        lock.release();
      }

      // Watch the folder for new messages using the exists event.
      // After lock release, imapflow auto-IDLE kicks in to keep
      // the connection alive and receive server notifications.
      await this.watchFolder(folder);
    } catch (error) {
      if (this.running) {
        this.logger.error(`Error processing folder "${folder}"`, error);
        this.lastError = (error as Error).message;
        this.notifyStatus();
      }
    }
  }

  private async watchFolder(folder: string): Promise<void> {
    if (!this.client?.usable || !this.running) return;

    this.logger.info(`Watching for new messages in ${folder}...`);

    let processing = false;

    const onExists = async (data: {
      path: string;
      count: number;
      prevCount: number;
    }) => {
      if (!this.running || !this.client?.usable || processing) return;
      if (data.path !== folder || data.count <= data.prevCount) return;

      processing = true;
      try {
        this.logger.info(
          `${data.count - data.prevCount} new message(s) in ${folder}`,
        );
        const lock = await this.client.getMailboxLock(folder);
        try {
          await this.forwardNewMessages();
        } finally {
          lock.release();
        }
      } catch (error) {
        if (this.running) {
          this.logger.error(
            `Error processing new messages in "${folder}"`,
            error,
          );
          this.lastError = (error as Error).message;
          this.notifyStatus();
        }
      } finally {
        processing = false;
      }
    };

    this.client.on('exists', onExists);

    // Wait until we're stopped or the connection drops
    try {
      await new Promise<void>((resolve) => {
        this.stopResolve = resolve;
      });
    } finally {
      this.client?.off('exists', onExists);
    }
  }

  private async forwardNewMessages(): Promise<void> {
    if (!this.client?.usable) return;

    const messages: FetchMessageObject[] = [];
    // Search for messages without the forwarded flag
    try {
      for await (const message of this.client.fetch(
        { seen: false },
        { source: true, uid: true, flags: true },
      )) {
        if (!message.flags?.has(forwardedFlag)) {
          messages.push(message);
        }
      }
    } catch {
      // No messages or fetch error — continue
      return;
    }

    if (messages.length > 0) {
      this.logger.info(`Found ${messages.length} new message(s) to forward`);
    }

    // Forward messages sequentially to avoid overwhelming the SMTP server
    for (const message of messages) {
      // eslint-disable-next-line no-await-in-loop
      await this.forwardMessage(message);
    }
  }

  private async forwardMessage(message: FetchMessageObject): Promise<void> {
    if (!this.client?.usable) return;

    try {
      const rawSource = message.source;
      if (!rawSource) {
        this.logger.warn(
          `Message UID ${message.uid}: no source data, skipping`,
        );
        return;
      }

      // Send the raw message preserving all original headers
      await this.transporter.sendMail({
        envelope: {
          from: '',
          to: this.targetUser,
        },
        raw: rawSource,
      });

      this.logger.info(`Message UID ${message.uid}: forwarded successfully`);

      // Mark as forwarded
      await this.client.messageFlagsAdd({ uid: message.uid }, [forwardedFlag], {
        uid: true,
      });

      // Optionally delete after forwarding
      if (this.source.deleteAfterForward) {
        await this.client.messageDelete({ uid: message.uid }, { uid: true });
        this.logger.info(`Message UID ${message.uid}: deleted from source`);
      }

      this.lastSync = new Date().toISOString();
      this.notifyStatus();
    } catch (error) {
      this.logger.error(`Message UID ${message.uid}: failed to forward`, error);
    }
  }

  private notifyStatus(): void {
    this.events.onStatusChange?.(this.getStatus());
  }
}
