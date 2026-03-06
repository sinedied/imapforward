import {Component, signal, HostListener} from '@angular/core';
import {Hero} from './components/hero';
import {Setup} from './components/setup';
import {ConfigTool} from './components/config-tool';

@Component({
  selector: 'app-root',
  imports: [Hero, Setup, ConfigTool],
  template: `
    <nav [class.scrolled]="scrolled()">
      <div class="nav-inner">
        <a href="#" class="nav-logo">
          <span class="nav-logo-icon"></span>
          imapforward
        </a>
        <div class="nav-links">
          <a href="#setup">Setup</a>
          <a href="#config">Config</a>
          <a href="https://github.com/sinedied/imapforward" target="_blank" rel="noopener"
            class="nav-github" title="GitHub">
            <svg viewBox="0 0 16 16" fill="currentColor" width="20" height="20">
              <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38
                0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13
                -.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66
                .07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15
                -.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0
                1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56
                .82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0
                1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/>
            </svg>
          </a>
        </div>
      </div>
    </nav>

    <main>
      <app-hero />
      <app-setup />
      <app-config-tool />
    </main>

    <footer>
      <div class="footer-inner">
        <span class="footer-logo">imapforward</span>
        <span class="footer-links">
          <a href="https://github.com/sinedied/imapforward" target="_blank" rel="noopener">GitHub</a>
          <span class="sep">&middot;</span>
          <a href="https://www.npmjs.com/package/imapforward" target="_blank" rel="noopener">npm</a>
          <span class="sep">&middot;</span>
          <a href="https://github.com/sinedied/imapforward/blob/main/LICENSE" target="_blank" rel="noopener">MIT License</a>
        </span>
      </div>
    </footer>
  `,
  styles: [`
    :host {
      display: block;
    }

    /* --- Nav --- */
    nav {
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      z-index: 100;
      padding: 0.75rem 1.5rem;
      transition: background 0.3s, box-shadow 0.3s, backdrop-filter 0.3s;

      &.scrolled {
        background: color-mix(in srgb, var(--bg-primary) 80%, transparent);
        backdrop-filter: blur(12px);
        -webkit-backdrop-filter: blur(12px);
        box-shadow: 0 1px 0 var(--border);
      }
    }

    .nav-inner {
      max-width: 1100px;
      margin: 0 auto;
      display: flex;
      align-items: center;
      justify-content: space-between;
    }

    .nav-logo-icon {
      display: inline-block;
      width: 22px;
      height: 22px;
      background: var(--accent);
      mask: url('../assets/logo.svg') no-repeat center / contain;
      -webkit-mask: url('../assets/logo.svg') no-repeat center / contain;
    }

    .nav-logo {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      font-weight: 700;
      font-size: 1rem;
      color: var(--text-primary);
      letter-spacing: -0.02em;

      &:hover {
        color: var(--accent);
      }

      svg {
        color: var(--accent);
      }
    }

    .nav-links {
      display: flex;
      align-items: center;
      gap: 1.5rem;

      a {
        font-size: 0.85rem;
        color: var(--text-secondary);
        font-weight: 500;
        transition: color 0.2s;

        &:hover {
          color: var(--text-primary);
        }
      }
    }

    .nav-github {
      display: flex;
      align-items: center;
      opacity: 0.7;
      transition: opacity 0.2s;

      &:hover {
        opacity: 1;
      }
    }

    /* --- Footer --- */
    footer {
      padding: 2rem 1.5rem;
      border-top: 1px solid var(--border);
      background: var(--bg-secondary);
    }

    .footer-inner {
      max-width: 1100px;
      margin: 0 auto;
      display: flex;
      justify-content: space-between;
      align-items: center;
      flex-wrap: wrap;
      gap: 0.75rem;
    }

    .footer-logo {
      font-weight: 600;
      font-size: 0.9rem;
      color: var(--text-muted);
      letter-spacing: -0.02em;
    }

    .footer-links {
      font-size: 0.8rem;
      color: var(--text-muted);
      display: flex;
      align-items: center;
      gap: 0.5rem;

      a {
        color: var(--text-secondary);

        &:hover {
          color: var(--accent);
        }
      }

      .sep {
        opacity: 0.4;
      }
    }
  `],
})
export class App {
  protected readonly scrolled = signal(false);

  @HostListener('window:scroll')
  onScroll(): void {
    this.scrolled.set(window.scrollY > 20);
  }
}

