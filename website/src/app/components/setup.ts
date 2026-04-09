import {Component} from '@angular/core';
import {ScrollAnimate} from '../directives/scroll-animate';

@Component({
  selector: 'app-setup',
  imports: [ScrollAnimate],
  template: `
    <section id="setup" class="setup">
      <div class="container">
        <h2 appScrollAnimate>Getting Started</h2>

        <div class="steps">
          <div class="step" appScrollAnimate>
            <div class="step-number">1</div>
            <div class="step-content">
              <h3>Install</h3>
              <p>Download a binary for your platform (Linux, macOS, Windows) from <a href="https://github.com/sinedied/imapforward/releases" target="_blank" rel="noopener noreferrer">GitHub Releases</a>, or pull the Docker image:</p>
              <div class="code-block">
                <pre><code>docker pull ghcr.io/sinedied/imapforward:latest</code></pre>
              </div>
            </div>
          </div>

          <div class="step" appScrollAnimate>
            <div class="step-number">2</div>
            <div class="step-content">
              <h3>Configure</h3>
              <p>
                Create a <code>config.json</code> file with your target and source IMAP accounts.
                Use the <a href="#config">configuration generator</a> below to build it interactively.
              </p>
            </div>
          </div>

          <div class="step" appScrollAnimate>
            <div class="step-number">3</div>
            <div class="step-content">
              <h3>Run</h3>
              <p>Start forwarding with a single command:</p>
              <div class="code-block">
                <pre><code>imapforward</code></pre>
              </div>
              <p>Or with Docker:</p>
              <div class="code-block">
                <pre><code>docker run -d --name imapforward \\
  -v ./config.json:/app/config.json:ro \\
  ghcr.io/sinedied/imapforward:latest</code></pre>
              </div>
            </div>
          </div>
        </div>

        <div class="details-grid">
          <div class="detail-card" appScrollAnimate>
            <h3>CLI Options</h3>
            <table>
              <tbody>
                <tr><td><code>-config</code></td><td>Path to config file</td></tr>
                <tr><td><code>-log-level</code></td><td>Log level (debug, info, warn, error)</td></tr>
                <tr><td><code>-auth</code></td><td>Run OAuth2 flow for Gmail API token</td></tr>
                <tr><td><code>-version</code></td><td>Show version</td></tr>
                <tr><td><code>-help</code></td><td>Show help</td></tr>
              </tbody>
            </table>
          </div>

          <div class="detail-card" appScrollAnimate>
            <h3>Environment Variables</h3>
            <table>
              <tbody>
                <tr><td><code>IMAPFORWARD_CONFIG</code></td><td>Config file path (alternative to -config)</td></tr>
                <tr><td><code>LOG_LEVEL</code></td><td>Log level (alternative to -log-level)</td></tr>
              </tbody>
            </table>
          </div>

          <div class="detail-card" appScrollAnimate>
            <h3>Docker Compose</h3>
            <div class="code-block">
              <pre><code>services:
  imapforward:
    image: ghcr.io/sinedied/imapforward:latest
    restart: unless-stopped
    volumes:
      - ./config.json:/app/config.json:ro</code></pre>
            </div>
          </div>

          <div class="detail-card" appScrollAnimate>
            <h3>Health Check</h3>
            <p>
              Built-in HTTP health check at <code>/health</code> on port <strong>8080</strong> (configurable).
              Returns JSON with overall status and per-source connection info.
            </p>
            <div class="code-block">
              <pre><code>curl http://localhost:8080/health</code></pre>
            </div>
          </div>
        </div>
      </div>
    </section>
  `,
  styles: [`
    .setup {
      padding: 6rem 1.5rem;
      background: var(--bg-secondary);
    }

    .container {
      max-width: 900px;
      margin: 0 auto;
    }

    h2 {
      font-size: clamp(1.75rem, 4vw, 2.5rem);
      text-align: center;
      margin-bottom: 3.5rem;
      letter-spacing: -0.02em;
    }

    .steps {
      display: flex;
      flex-direction: column;
      gap: 2rem;
      margin-bottom: 4rem;
    }

    .step {
      display: flex;
      gap: 1.25rem;
      align-items: flex-start;
    }

    .step-number {
      flex-shrink: 0;
      width: 2.5rem;
      height: 2.5rem;
      display: flex;
      align-items: center;
      justify-content: center;
      border-radius: 50%;
      background: var(--accent);
      color: #fff;
      font-weight: 700;
      font-size: 1.1rem;
    }

    .step-content {
      flex: 1;
      min-width: 0;

      h3 {
        font-size: 1.15rem;
        margin-bottom: 0.4rem;
      }

      p {
        color: var(--text-secondary);
        margin-bottom: 0.75rem;
        font-size: 0.95rem;
      }

      code:not(pre code) {
        background: var(--bg-surface);
        padding: 0.15em 0.4em;
        border-radius: 4px;
        font-size: 0.85em;
        color: var(--accent);
      }
    }

    .code-block {
      background: var(--bg-surface);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      overflow-x: auto;
      margin-bottom: 0.75rem;

      pre {
        padding: 0.75rem 1rem;
        margin: 0;
        font-size: 0.85rem;
        line-height: 1.5;
        white-space: pre;
      }

      code {
        color: var(--text-primary);
      }
    }

    .details-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
      gap: 1.25rem;
    }

    .detail-card {
      background: var(--bg-surface);
      border: 1px solid var(--border);
      border-radius: var(--radius-lg);
      padding: 1.5rem;

      h3 {
        font-size: 1rem;
        margin-bottom: 0.75rem;
        color: var(--accent);
      }

      p {
        color: var(--text-secondary);
        font-size: 0.9rem;
        margin-bottom: 0.75rem;
      }

      code:not(pre code) {
        background: var(--bg-elevated);
        padding: 0.15em 0.4em;
        border-radius: 4px;
        font-size: 0.85em;
      }

      table {
        width: 100%;
        border-collapse: collapse;

        td {
          padding: 0.4rem 0.5rem;
          font-size: 0.85rem;
          border-bottom: 1px solid var(--border);

          &:first-child {
            white-space: nowrap;
          }

          &:last-child {
            color: var(--text-secondary);
          }

          code {
            background: var(--bg-elevated);
            padding: 0.1em 0.35em;
            border-radius: 3px;
            font-size: 0.9em;
          }
        }

        tr:last-child td {
          border-bottom: none;
        }
      }
    }

    @media (max-width: 700px) {
      .details-grid {
        grid-template-columns: 1fr;
      }
    }
  `],
})
export class Setup {}
