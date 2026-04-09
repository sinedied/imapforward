import {Component, computed, signal, inject} from '@angular/core';
import {
  FormBuilder,
  FormArray,
  type FormGroup,
  type AbstractControl,
  ReactiveFormsModule,
  Validators,
} from '@angular/forms';
import {toSignal} from '@angular/core/rxjs-interop';
import {startWith} from 'rxjs';
import {ScrollAnimate} from '../directives/scroll-animate';

const IMPLICIT_TLS_PORTS = new Set([465, 993]);

@Component({
  selector: 'app-config-tool',
  imports: [ReactiveFormsModule, ScrollAnimate],
  template: `
    <section id="config" class="config-tool">
      <div class="container" appScrollAnimate>
        <h2>Configuration Generator</h2>
        <p class="subtitle">Build your <code>config.json</code> interactively — download or copy when ready.</p>

        <div class="layout">
          <!-- Form -->
          <div class="form-panel">
            <form [formGroup]="form">

              <!-- Target -->
              <fieldset>
                <legend>
                  <svg viewBox="0 0 20 20" fill="currentColor" width="16" height="16">
                    <path d="M3 4a2 2 0 00-2 2v1.161l8.441 4.221a1.25 1.25 0 001.118 0L19 7.162V6a2 2 0 00-2-2H3z"/>
                    <path d="M19 8.839l-7.556 3.778a2.75 2.75 0 01-2.888 0L1 8.839V14a2 2 0 002 2h14a2 2 0 002-2V8.839z"/>
                  </svg>
                  Target
                </legend>

                <div class="form-row">
                  <div class="field">
                    <label>Forwarding Method *</label>
                    <div class="method-toggle">
                      <label class="radio-label">
                        <input type="radio" formControlName="forwardMethod" value="imap" />
                        <span>IMAP Append</span>
                      </label>
                      <label class="radio-label">
                        <input type="radio" formControlName="forwardMethod" value="smtp" />
                        <span>SMTP Forward</span>
                      </label>
                    </div>
                    <span class="hint">
                      {{ form.get('forwardMethod')?.value === 'smtp'
                        ? 'Forwards via SMTP — enables spam filtering, adds Reply-To for replies'
                        : 'Appends raw message via IMAP — preserves all headers, bypasses spam filters' }}
                    </span>
                  </div>
                </div>
                <div formGroupName="target">
                  <div class="form-row">
                    <div class="field">
                      <label for="t-host">Host *</label>
                      <input id="t-host" formControlName="host"
                        [placeholder]="form.get('forwardMethod')?.value === 'smtp' ? 'smtp.gmail.com' : 'imap.gmail.com'" />
                    </div>
                    <div class="field field-sm">
                      <label for="t-port">Port *</label>
                      <input id="t-port" type="number" formControlName="port"
                        (input)="onPortChange(targetGroup)" />
                    </div>
                    <div class="field field-xs">
                      <label class="toggle-label">
                        <input type="checkbox" formControlName="secure" />
                        <span>TLS</span>
                      </label>
                    </div>
                  </div>
                  <div formGroupName="auth" class="form-row">
                    <div class="field">
                      <label for="t-user">Username *</label>
                      <input id="t-user" formControlName="user" placeholder="your-email&#64;gmail.com" />
                    </div>
                    <div class="field">
                      <label for="t-pass">App Password *</label>
                      <input id="t-pass" type="password" formControlName="pass"
                        placeholder="your-app-password" autocomplete="off" />
                    </div>
                  </div>
                  <div class="form-row">
                    <div class="field">
                      <label for="t-folder">Folder</label>
                      <input id="t-folder" formControlName="folder" placeholder="INBOX"
                        [class.field-disabled]="form.get('forwardMethod')?.value === 'smtp'" />
                      @if (form.get('forwardMethod')?.value === 'smtp') {
                        <span class="hint">Not used with SMTP forwarding</span>
                      }
                    </div>
                  </div>
                </div>
              </fieldset>

              <!-- Sources -->
              <fieldset>
                <legend>
                  <svg viewBox="0 0 20 20" fill="currentColor" width="16" height="16">
                    <path d="M5.127 3.502L5.25 3.5h9.5c.041 0 .082 0 .123.002A2.251 2.251 0 0012.75 2h-5.5a2.25 2.25 0 00-2.123 1.502zM1 10.25A2.25 2.25 0 013.25 8h13.5A2.25 2.25 0 0119 10.25v5.5A2.25 2.25 0 0116.75 18H3.25A2.25 2.25 0 011 15.75v-5.5z"/>
                    <path d="M3.218 5.002A2.25 2.25 0 015.25 3.5h9.5a2.25 2.25 0 012.032 1.502A2.25 2.25 0 0016.75 8H3.25a2.25 2.25 0 01-.032-3z" opacity="0.5"/>
                  </svg>
                  Source Accounts
                </legend>
                <div formArrayName="sources">
                  @for (source of sources.controls; track $index; let i = $index) {
                    <div class="source-card" [formGroupName]="i">
                      <div class="source-header">
                        <span class="source-label">Source {{ i + 1 }}</span>
                        @if (sources.length > 1) {
                          <button type="button" class="btn-remove" (click)="removeSource(i)"
                            title="Remove source">✕</button>
                        }
                      </div>
                      <div class="form-row">
                        <div class="field">
                          <label [for]="'s-name-' + i">Name *</label>
                          <input [id]="'s-name-' + i" formControlName="name" placeholder="Work Email" />
                        </div>
                      </div>
                      <div class="form-row">
                        <div class="field">
                          <label [for]="'s-host-' + i">Host *</label>
                          <input [id]="'s-host-' + i" formControlName="host" placeholder="imap.example.com" />
                        </div>
                        <div class="field field-sm">
                          <label [for]="'s-port-' + i">Port *</label>
                          <input [id]="'s-port-' + i" type="number" formControlName="port"
                            (input)="onPortChange(source)" />
                        </div>
                        <div class="field field-xs">
                          <label class="toggle-label">
                            <input type="checkbox" formControlName="secure" />
                            <span>TLS</span>
                          </label>
                        </div>
                      </div>
                      <div formGroupName="auth" class="form-row">
                        <div class="field">
                          <label [for]="'s-user-' + i">Username *</label>
                          <input [id]="'s-user-' + i" formControlName="user" placeholder="user&#64;example.com" />
                        </div>
                        <div class="field">
                          <label [for]="'s-pass-' + i">Password *</label>
                          <input [id]="'s-pass-' + i" type="password" formControlName="pass"
                            placeholder="password" autocomplete="off" />
                        </div>
                      </div>
                      <div class="form-row">
                        <div class="field">
                          <label [for]="'s-folders-' + i">Folders</label>
                          <input [id]="'s-folders-' + i" formControlName="folders"
                            placeholder="INBOX, Important" />
                          <span class="hint">Comma-separated list</span>
                        </div>
                        <div class="field">
                          <label [for]="'s-targetFolder-' + i">Target Folder</label>
                          <input [id]="'s-targetFolder-' + i" formControlName="targetFolder"
                            placeholder="Import/Work" />
                          <span class="hint">Override target mailbox for this source (IMAP only)</span>
                        </div>
                      </div>
                      <div class="form-row">
                        <div class="field field-xs">
                          <label class="toggle-label">
                            <input type="checkbox" formControlName="deleteAfterForward" />
                            <span>Delete after forward</span>
                          </label>
                        </div>
                      </div>
                    </div>
                  }
                </div>
                <button type="button" class="btn-add" (click)="addSource()">+ Add Source Account</button>
              </fieldset>

            </form>
          </div>

          <!-- Preview -->
          <div class="preview-panel">
            <div class="preview-sticky">
              <div class="preview-header">
                <span>config.json</span>
                <div class="preview-actions">
                  <button (click)="copy()" [class.copied]="copied()" [disabled]="form.invalid">
                    {{ copied() ? '✓ Copied' : '⎘ Copy' }}
                  </button>
                  <button (click)="download()" [disabled]="form.invalid" class="btn-download">
                    ↓ Download
                  </button>
                </div>
              </div>
              <pre class="preview-code"><code>{{ configJson() }}</code></pre>
              @if (form.invalid) {
                <div class="preview-warning">
                  Fill in all required fields (*) to enable download.
                </div>
              }
            </div>
          </div>
        </div>
      </div>
    </section>
  `,
  styles: [`
    .config-tool {
      padding: 6rem 1.5rem;
      background: var(--bg-primary);
    }

    .container {
      max-width: 1100px;
      margin: 0 auto;
    }

    h2 {
      font-size: clamp(1.75rem, 4vw, 2.5rem);
      text-align: center;
      letter-spacing: -0.02em;
      margin-bottom: 0.5rem;
    }

    .subtitle {
      text-align: center;
      color: var(--text-secondary);
      margin-bottom: 3rem;
      font-size: 1rem;

      code {
        background: var(--bg-surface);
        padding: 0.15em 0.4em;
        border-radius: 4px;
        font-size: 0.9em;
        color: var(--accent);
      }
    }

    .layout {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 2rem;
      align-items: start;
    }

    /* --- Form Panel --- */
    .form-panel {
      display: flex;
      flex-direction: column;
      gap: 1.5rem;
    }

    fieldset {
      border: 1px solid var(--border);
      border-radius: var(--radius-lg);
      padding: 1.25rem;
      background: var(--bg-surface);
    }

    legend {
      display: flex;
      align-items: center;
      gap: 0.4rem;
      font-weight: 600;
      font-size: 0.95rem;
      padding: 0 0.4rem;
      color: var(--accent);

      svg {
        opacity: 0.8;
      }
    }

    .legend-toggle {
      margin-left: auto;
      font-weight: 400;
    }

    .form-row {
      display: flex;
      gap: 0.75rem;
      margin-bottom: 0.75rem;
      flex-wrap: wrap;
    }

    .field {
      flex: 1;
      min-width: 140px;

      label:not(.toggle-label) {
        display: block;
        font-size: 0.8rem;
        color: var(--text-secondary);
        margin-bottom: 0.3rem;
        font-weight: 500;
      }

      input[type="text"],
      input[type="number"],
      input[type="password"],
      input:not([type]) {
        width: 100%;
        padding: 0.55rem 0.75rem;
        background: var(--bg-elevated);
        border: 1px solid var(--border);
        border-radius: var(--radius);
        color: var(--text-primary);
        font-size: 0.85rem;
        font-family: inherit;
        transition: border-color 0.2s;
        outline: none;

        &::placeholder {
          color: var(--text-muted);
        }

        &:focus {
          border-color: var(--border-focus);
        }

        &.ng-invalid.ng-touched {
          border-color: var(--danger);
        }
      }

      .hint {
        display: block;
        font-size: 0.75rem;
        color: var(--text-muted);
        margin-top: 0.2rem;
      }
    }

    .field-sm {
      flex: 0 0 100px;
      min-width: 80px;
    }

    .field-xs {
      flex: 0 0 auto;
      min-width: auto;
      display: flex;
      align-items: flex-end;
      padding-bottom: 0.55rem;
    }

    .field-disabled {
      opacity: 0.4;
      pointer-events: none;
    }

    .method-toggle {
      display: flex;
      gap: 1rem;
      margin-top: 0.2rem;
    }

    .radio-label {
      display: inline-flex;
      align-items: center;
      gap: 0.4rem;
      cursor: pointer;
      font-size: 0.85rem;
      color: var(--text-secondary);
      user-select: none;

      input[type="radio"] {
        accent-color: var(--accent);
        margin: 0;
      }
    }

    .toggle-label {
      display: inline-flex;
      align-items: center;
      gap: 0.5rem;
      cursor: pointer;
      font-size: 0.8rem;
      color: var(--text-secondary);
      user-select: none;
      white-space: nowrap;
      padding-right: 0.5rem;
      line-height: 1;

      input[type="checkbox"] {
        appearance: none;
        -webkit-appearance: none;
        width: 1rem;
        height: 1rem;
        margin: 0;
        border: 1.5px solid var(--border);
        border-radius: 3px;
        background: var(--bg-elevated);
        cursor: pointer;
        position: relative;
        flex-shrink: 0;
        transition: background 0.15s, border-color 0.15s;

        &:checked {
          background: var(--accent);
          border-color: var(--accent);
        }

        &:checked::after {
          content: '';
          position: absolute;
          left: 3px;
          top: 1px;
          width: 5px;
          height: 8px;
          border: solid #fff;
          border-width: 0 2px 2px 0;
          transform: rotate(45deg);
        }
      }

      span {
        line-height: 1;
      }
    }

    /* --- Source Cards --- */
    .source-card {
      background: var(--bg-elevated);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      padding: 1rem;
      margin-bottom: 0.75rem;
    }

    .source-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 0.75rem;
    }

    .source-label {
      font-size: 0.8rem;
      font-weight: 600;
      color: var(--text-secondary);
      text-transform: uppercase;
      letter-spacing: 0.05em;
    }

    .btn-remove {
      background: transparent;
      border: 1px solid var(--border);
      border-radius: var(--radius);
      color: var(--text-muted);
      cursor: pointer;
      padding: 0.2rem 0.5rem;
      font-size: 0.8rem;
      transition: color 0.2s, border-color 0.2s;

      &:hover {
        color: var(--danger);
        border-color: var(--danger);
      }
    }

    .btn-add {
      width: 100%;
      padding: 0.65rem;
      background: transparent;
      border: 1px dashed var(--border);
      border-radius: var(--radius);
      color: var(--accent);
      font-size: 0.85rem;
      font-weight: 500;
      cursor: pointer;
      transition: background 0.2s, border-color 0.2s;
      font-family: inherit;

      &:hover {
        background: var(--accent-subtle);
        border-color: var(--accent);
      }
    }

    /* --- Preview Panel --- */
    .preview-panel {
      position: relative;
    }

    .preview-sticky {
      position: sticky;
      top: 5rem;
      border: 1px solid var(--border);
      border-radius: var(--radius-lg);
      overflow: hidden;
      background: var(--bg-surface);
    }

    .preview-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 0.65rem 1rem;
      background: var(--bg-elevated);
      border-bottom: 1px solid var(--border);
      font-size: 0.8rem;
      font-weight: 600;
      color: var(--text-secondary);
    }

    .preview-actions {
      display: flex;
      gap: 0.4rem;

      button {
        padding: 0.35rem 0.75rem;
        border: 1px solid var(--border);
        border-radius: var(--radius);
        background: transparent;
        color: var(--text-secondary);
        font-size: 0.75rem;
        font-weight: 500;
        cursor: pointer;
        font-family: inherit;
        transition: color 0.2s, border-color 0.2s, background 0.2s;

        &:hover:not(:disabled) {
          color: var(--accent);
          border-color: var(--accent);
        }

        &:disabled {
          opacity: 0.4;
          cursor: not-allowed;
        }

        &.copied {
          color: var(--success);
          border-color: var(--success);
        }

        &.btn-download {
          background: var(--accent);
          color: #fff;
          border-color: var(--accent);

          &:hover:not(:disabled) {
            background: var(--accent-hover);
            border-color: var(--accent-hover);
            color: #fff;
          }
        }
      }
    }

    .preview-code {
      padding: 1rem;
      margin: 0;
      font-size: 0.8rem;
      line-height: 1.6;
      overflow-x: auto;
      max-height: 70vh;
      white-space: pre;
      color: var(--text-primary);
    }

    .preview-warning {
      padding: 0.6rem 1rem;
      background: rgba(248, 113, 113, 0.08);
      border-top: 1px solid var(--border);
      color: var(--danger);
      font-size: 0.8rem;
    }

    /* --- Responsive --- */
    @media (max-width: 768px) {
      .layout {
        grid-template-columns: 1fr;
      }

      .preview-sticky {
        position: static;
      }
    }
  `],
})
export class ConfigTool {
  private readonly fb = inject(FormBuilder);

  protected readonly copied = signal(false);

  protected readonly form = this.fb.group({
    forwardMethod: ['imap'],
    target: this.fb.group({
      host: ['imap.gmail.com', Validators.required],
      port: [993, [Validators.required]],
      secure: [true],
      auth: this.fb.group({
        user: ['', Validators.required],
        pass: ['', Validators.required],
      }),
      folder: ['INBOX'],
    }),
    sources: this.fb.array([this.createSource()]),

  });

  private readonly formValue = toSignal(
    this.form.valueChanges.pipe(startWith(this.form.getRawValue())),
    {initialValue: this.form.getRawValue()},
  );

  protected readonly configJson = computed(() => {
    const v = this.formValue();
    if (!v) return '';

    const method = v.forwardMethod || 'imap';
    const config: Record<string, unknown> = {
      target: {
        host: v.target?.host || '',
        port: Number(v.target?.port) || 993,
        secure: v.target?.secure ?? true,
        auth: {
          user: v.target?.auth?.user || '',
          pass: v.target?.auth?.pass || '',
        },
        ...(method === 'imap' && v.target?.folder && v.target.folder !== 'INBOX'
          ? {folder: v.target.folder}
          : {}),
      },
      ...(method !== 'imap' ? {forwardMethod: method} : {}),
      sources: (v.sources ?? []).map((s) => {
        const tf = (s as Record<string, unknown>)?.['targetFolder'] as string | undefined;
        return {
          name: s?.name || '',
          host: s?.host || '',
          port: Number(s?.port) || 993,
          secure: s?.secure ?? true,
          auth: {
            user: s?.auth?.user || '',
            pass: s?.auth?.pass || '',
          },
          folders: this.parseFolders(s?.folders as string),
          deleteAfterForward: s?.deleteAfterForward ?? false,
          ...(tf ? {targetFolder: tf} : {}),
        };
      }),
    };

    return JSON.stringify(config, null, 2);
  });

  protected get sources(): FormArray {
    return this.form.get('sources') as FormArray;
  }

  protected get targetGroup(): AbstractControl {
    return this.form.get('target')!;
  }

  protected createSource(): FormGroup {
    return this.fb.group({
      name: ['', Validators.required],
      host: ['', Validators.required],
      port: [993, Validators.required],
      secure: [true],
      auth: this.fb.group({
        user: ['', Validators.required],
        pass: ['', Validators.required],
      }),
      folders: ['INBOX'],
      deleteAfterForward: [false],
      targetFolder: [''],
    });
  }

  protected addSource(): void {
    this.sources.push(this.createSource());
  }

  protected removeSource(index: number): void {
    if (this.sources.length > 1) {
      this.sources.removeAt(index);
    }
  }

  protected onPortChange(group: AbstractControl): void {
    const port = Number(group.get('port')?.value);
    group
      .get('secure')
      ?.setValue(IMPLICIT_TLS_PORTS.has(port), {emitEvent: false});
  }

  protected async copy(): Promise<void> {
    await navigator.clipboard.writeText(this.configJson());
    this.copied.set(true);
    setTimeout(() => this.copied.set(false), 2000);
  }

  protected download(): void {
    const blob = new Blob([this.configJson()], {type: 'application/json'});
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'config.json';
    a.click();
    URL.revokeObjectURL(url);
  }

  private parseFolders(value: string | undefined): string[] {
    if (!value) return ['INBOX'];
    const folders = value
      .split(',')
      .map((f) => f.trim())
      .filter(Boolean);
    return folders.length > 0 ? folders : ['INBOX'];
  }
}
