import {Component, signal} from '@angular/core';

@Component({
  selector: 'app-hero',
  template: `
    <section class="hero">
      <div class="hero-bg"></div>
      <div class="content">
        <h1>
          <span class="logo-icon">
            <svg viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
              <rect x="2" y="7" width="18" height="14" rx="2" stroke="currentColor" stroke-width="1.8"/>
              <path d="M2 9l9 6.5L20 9" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
              <path d="M22 15l6-5m0 0l-6-5m6 5H18" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </span>
          <span class="title-imap">imap</span><span class="title-forward">forward</span>
        </h1>
        <p class="tagline">
          Simple real-time IMAP email forwarder for syncing multiple
          email accounts into one. Built to replace deprecated gmailify for Gmail.
        </p>

        <div class="features">
          <div class="feature-card">
            <div class="feature-icon">⚡</div>
            <h3>Real-time Sync</h3>
            <p>IMAP IDLE push notifications for instant forwarding</p>
          </div>
          <div class="feature-card">
            <div class="feature-icon">🔒</div>
            <h3>Header Preservation</h3>
            <p>Raw RFC822 append preserves original email headers</p>
          </div>
          <div class="feature-card">
            <div class="feature-icon">🔄</div>
            <h3>Reliable</h3>
            <p>Auto-reconnect with exponential backoff</p>
          </div>
        </div>

        <div class="install-commands">
          <div class="install-cmd">
            <code>npm install -g imapforward</code>
            <button
              (click)="copyCmd('npm install -g imapforward')"
              [class.copied]="copiedCmd() === 'npm install -g imapforward'"
              title="Copy to clipboard">
              {{ copiedCmd() === 'npm install -g imapforward' ? '✓' : '⎘' }}
            </button>
          </div>
          <span class="or">or</span>
          <div class="install-cmd">
            <code>docker pull ghcr.io/sinedied/imapforward:latest</code>
            <button
              (click)="copyCmd('docker pull ghcr.io/sinedied/imapforward:latest')"
              [class.copied]="copiedCmd() === 'docker pull ghcr.io/sinedied/imapforward:latest'"
              title="Copy to clipboard">
              {{ copiedCmd() === 'docker pull ghcr.io/sinedied/imapforward:latest' ? '✓' : '⎘' }}
            </button>
          </div>
        </div>

        <div class="cta-row">
          <a href="#config" class="cta-btn primary">Create your config</a>
          <a href="https://github.com/sinedied/imapforward" target="_blank" rel="noopener" class="cta-btn secondary">
            <svg viewBox="0 0 16 16" fill="currentColor" width="18" height="18">
              <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38
                0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13
                -.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66
                .07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15
                -.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0
                1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56
                .82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0
                1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/>
            </svg>
            GitHub
          </a>
        </div>
      </div>
    </section>
  `,
  styles: [`
    .hero {
      position: relative;
      min-height: 100vh;
      min-height: 100dvh;
      display: flex;
      align-items: center;
      justify-content: center;
      overflow: hidden;
      padding: 6rem 1.5rem 4rem;
    }

    .hero-bg {
      position: absolute;
      inset: 0;
      background:
        radial-gradient(ellipse 80% 60% at 50% -20%, rgba(79, 138, 255, 0.15), transparent),
        radial-gradient(ellipse 60% 50% at 80% 50%, rgba(124, 77, 255, 0.08), transparent);
      pointer-events: none;
    }

    .content {
      position: relative;
      max-width: 860px;
      text-align: center;
    }

    h1 {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 0.5rem;
      font-size: clamp(2.5rem, 6vw, 4.5rem);
      font-weight: 800;
      letter-spacing: -0.03em;
      margin-bottom: 1.25rem;
    }

    .logo-icon {
      display: inline-flex;

      svg {
        width: 1em;
        height: 1em;
        color: var(--accent);
      }
    }

    .title-imap {
      color: var(--text-primary);
    }

    .title-forward {
      background: var(--accent-gradient);
      -webkit-background-clip: text;
      background-clip: text;
      -webkit-text-fill-color: transparent;
    }

    .tagline {
      font-size: clamp(1rem, 2vw, 1.25rem);
      color: var(--text-secondary);
      max-width: 600px;
      margin: 0 auto 3rem;
    }

    .features {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: 1rem;
      margin-bottom: 3rem;
      max-width: 680px;
      margin-inline: auto;
    }

    .feature-card {
      position: relative;
      border: none;
      border-radius: var(--radius-lg);
      padding: 1.25rem 1rem;
      text-align: center;
      overflow: hidden;
      color: #fff;

      &:nth-child(1) {
        background: linear-gradient(135deg, #2563eb, #0ea5e9);
      }

      &:nth-child(2) {
        background: linear-gradient(135deg, #7c3aed, #a855f7);
      }

      &:nth-child(3) {
        background: linear-gradient(135deg, #0891b2, #2dd4bf);
      }

      &::after {
        content: '';
        position: absolute;
        top: 0;
        left: 15%;
        right: 15%;
        height: 1px;
        background: linear-gradient(
          90deg,
          transparent,
          rgba(255, 255, 255, 0.5),
          transparent
        );
      }

      .feature-icon {
        font-size: 1.75rem;
        margin-bottom: 0.6rem;
        display: block;
      }

      h3 {
        font-size: 0.95rem;
        font-weight: 700;
        margin-bottom: 0.25rem;
        color: #fff;
        text-shadow: 0 1px 3px rgba(0, 0, 0, 0.25);
      }

      p {
        font-size: 0.78rem;
        color: rgba(255, 255, 255, 0.92);
        line-height: 1.4;
        text-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
      }
    }

    @media (max-width: 600px) {
      .features {
        grid-template-columns: 1fr;
        max-width: 300px;
      }
    }

    .install-commands {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 0.75rem;
      flex-wrap: wrap;
      margin-bottom: 2.5rem;
    }

    .or {
      color: var(--text-muted);
      font-size: 0.85rem;
    }

    .install-cmd {
      display: flex;
      align-items: center;
      gap: 0;
      background: var(--bg-surface);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      overflow: hidden;
      font-size: 0.85rem;

      code {
        padding: 0.6rem 1rem;
        color: var(--text-primary);
        user-select: all;
      }

      button {
        padding: 0.6rem 0.75rem;
        background: transparent;
        border: none;
        border-left: 1px solid var(--border);
        color: var(--text-secondary);
        cursor: pointer;
        font-size: 1rem;
        transition: color 0.2s, background 0.2s;

        &:hover {
          background: var(--accent-subtle);
          color: var(--accent);
        }

        &.copied {
          color: var(--success);
        }
      }
    }

    .cta-row {
      display: flex;
      gap: 1rem;
      justify-content: center;
      flex-wrap: wrap;
    }

    .cta-btn {
      display: inline-flex;
      align-items: center;
      gap: 0.5rem;
      padding: 0.75rem 1.75rem;
      border-radius: var(--radius);
      font-weight: 600;
      font-size: 0.95rem;
      transition: transform 0.15s, box-shadow 0.2s, background 0.2s;

      &:hover {
        transform: translateY(-1px);
      }

      &.primary {
        background: var(--accent);
        color: #fff;
        box-shadow: 0 4px 16px rgba(79, 138, 255, 0.3);

        &:hover {
          background: var(--accent-hover);
          color: #fff;
          box-shadow: 0 6px 24px rgba(79, 138, 255, 0.4);
        }
      }

      &.secondary {
        background: var(--bg-surface);
        color: var(--text-primary);
        border: 1px solid var(--border);

        &:hover {
          border-color: var(--text-muted);
          color: var(--text-primary);
        }
      }
    }

    @media (max-width: 600px) {
      .install-commands {
        flex-direction: column;
      }

      .install-cmd code {
        font-size: 0.75rem;
      }
    }
  `],
})
export class Hero {
  protected readonly copiedCmd = signal<string | null>(null);

  async copyCmd(cmd: string): Promise<void> {
    await navigator.clipboard.writeText(cmd);
    this.copiedCmd.set(cmd);
    setTimeout(() => this.copiedCmd.set(null), 2000);
  }
}
