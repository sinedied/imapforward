# Contributing

Thanks for your interest in contributing to **imapforward**!

## Getting Started

1. Fork and clone the repo
2. Install dependencies: `npm install`
3. Create a branch for your change: `git checkout -b my-feature`

## Development

```bash
npm run build      # Compile TypeScript
npm run lint       # Check formatting (XO + Prettier)
npm run lint:fix   # Auto-fix lint issues
npm test           # Run tests (Vitest)
npm run dev        # Run with tsx (no build needed)
```

All contributions must pass `build`, `lint`, and `test` before merging.

## Submitting Changes

1. Make your changes with clear, focused commits using [Conventional Commits](https://www.conventionalcommits.org/)
2. Ensure all checks pass: `npm run build && npm run lint && npm test`
3. Open a pull request against `main`

## Guidelines

- Keep dependencies minimal — prefer Node.js built-ins
- Use strict TypeScript (no `any`)
- Co-locate tests next to source files (`*.test.ts`)
- Use ES module imports with `.js` extensions

## Reporting Issues

Open an issue with a clear description and steps to reproduce.
