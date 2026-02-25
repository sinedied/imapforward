# imapforward

A simple IMAP email forwarder for syncing multiple email accounts into Gmail.

## Overview

- Node.js 22+ CLI tool / daemon that monitors IMAP mailboxes via IDLE and forwards emails to a Gmail target via IMAP APPEND
- Preserves original email headers by appending raw RFC822 messages directly to the target mailbox
- Uses `imapflow` for IMAP — only runtime dependency!
- Supports multiple source accounts → single target, selective folder monitoring, optional cleanup

## Key Technologies and Frameworks

- **Runtime**: Node.js 22+, TypeScript (ES2024 target, Node16 module resolution)
- **IMAP**: `imapflow` — modern async/await IMAP client with IDLE support, also used for target APPEND
- **Testing**: Vitest
- **Linting**: XO + Prettier (single quotes, no bracket spacing)
- **Build**: `tsc` (TypeScript compiler)
- **CI/CD**: GitHub Actions, semantic-release

## Project Structure

```
src/
  cli.ts                 # Entry point with arg parsing and signal handling
  config.ts              # Config loading and validation
  forwarder.ts           # Per-source IMAP→IMAP forwarding logic
  connection-manager.ts  # Manages N source connections
  health.ts              # HTTP health check server
  logger.ts              # Minimal structured logger
  *.test.ts              # Unit tests (co-located)
```

## Development Workflow

```bash
npm install        # Install dependencies
npm run build      # Compile TypeScript → dist/
npm run lint       # XO + Prettier check
npm run lint:fix   # Auto-fix lint issues
npm test           # Run tests (vitest)
npm run dev        # Run with tsx (no build needed)
```

A task is only complete when `build`, `lint`, and `test` all pass.

## Coding Guidelines

- ES modules (`"type": "module"`) with `.js` extensions in imports
- Strict TypeScript — no `any`, explicit return types on public APIs
- Conventional commits for all changes
- XO + Prettier formatting (single quotes, no bracket spacing)
- Minimal dependencies — use Node.js built-in modules when possible
- Co-located test files (`*.test.ts` next to source files)
