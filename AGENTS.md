# imapforward

A simple IMAP email forwarder for syncing multiple email accounts into Gmail.

## Overview

- Go CLI tool / daemon that monitors IMAP mailboxes via IDLE and forwards emails to a Gmail target
- Two forwarding methods: IMAP APPEND (raw RFC822, bypasses spam filters) or SMTP (passes through spam filters, adds Reply-To for reply support)
- Preserves original email headers for reply/reply-all functionality
- Uses `go-imap/v2` for IMAP — only runtime dependency
- Supports multiple source accounts → single target, selective folder monitoring, optional cleanup

## Key Technologies and Frameworks

- **Runtime**: Go 1.22+
- **IMAP**: `github.com/emersion/go-imap/v2` — modern async IMAP client with IDLE support
- **SMTP**: Go standard library `net/smtp` — for SMTP forwarding method
- **Testing**: Go standard `testing` package
- **Build**: `go build` with ldflags for version injection
- **CI/CD**: GitHub Actions

## Project Structure

```
go.mod / go.sum          # Go module files
src/
  main.go                # Entry point with arg parsing and signal handling
  config.go              # Config loading and validation
  config_test.go         # Config unit tests
  forwarder.go           # Per-source IMAP→target forwarding logic
  forwarder_test.go      # Forwarder unit tests
  sender.go              # Sender interface + IMAP/SMTP implementations + ensureReplyTo
  sender_test.go         # Sender unit tests (ensureReplyTo, header manipulation)
  manager.go             # Manages N source forwarder goroutines
  health.go              # HTTP health check server
  health_test.go         # Health server tests
  logger.go              # Minimal structured logger
website/                 # Angular website (separate, not part of Go build)
```

## Development Workflow

```bash
go build -o imapforward ./src/  # Build
go test ./src/                  # Run tests
go test -v ./src/               # Run tests with verbose output
go run ./src/                   # Run directly (no build needed)
go vet ./src/                   # Static analysis
```

A task is only complete when `go build`, `go vet`, and `go test` all pass.

## Coding Guidelines

- Go 1.22+ with standard library conventions
- Strict error handling — no ignored errors
- Interfaces for testability (IMAPClient, Sender)
- Context-based cancellation for graceful shutdown
- Concurrent folder monitoring (one IMAP connection per folder)
- Co-located test files (`*_test.go` next to source files)
- Minimal dependencies — use Go standard library when possible
- Any config or feature change **must** also update `README.md`, `config.example.json`, and the website config editor (`website/src/app/components/config-tool.ts`) and setup guide (`website/src/app/components/setup.ts`)
