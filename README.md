<div align="center">



# <img src="https://sinedied.github.io/imapforward/logo.svg" alt="imapforward logo" height="32px"> imapforward

[![Build Status](https://img.shields.io/github/actions/workflow/status/sinedied/imapforward/ci.yml?style=flat-square)](https://github.com/sinedied/imapforward/actions)
[![Docker](https://img.shields.io/badge/ghcr.io-imapforward-blue?style=flat-square)](https://github.com/sinedied/imapforward/pkgs/container/imapforward)
[![License](https://img.shields.io/github/license/sinedied/imapforward?style=flat-square)](LICENSE)

Simple real-time IMAP email forwarder for syncing multiple email accounts into one.<br>
Built to replace deprecated [gmailify](https://support.google.com/mail/answer/7644837) tool for Gmail.

[Features](#features) · [Installation](#installation) · [Configuration](#configuration) · [Docker](#docker)

</div>

## Features

- **Real-time sync** — Uses IMAP IDLE for instant email forwarding
- **Two forwarding methods** — IMAP APPEND (preserves all headers) or SMTP (enables spam filtering)
- **Multiple sources** — Forward from multiple email accounts to a single Gmail
- **Original headers preserved** — Reply and Reply-All work with original senders and recipients
- **Selective folders** — Choose which folders to sync per source, with concurrent monitoring
- **Auto cleanup** — Optionally delete messages after successful forwarding
- **Production-grade** — Auto reconnect with exponential backoff, health check endpoint
- **Minimal footprint** — Static Go binary, no runtime dependencies
- **Docker ready** — Scratch-capable Alpine image with built-in health checks

## Installation

### Binary

Download from [GitHub Releases](https://github.com/sinedied/imapforward/releases), or build from source:

```bash
go install github.com/sinedied/imapforward@latest
```

### Docker

```bash
docker pull ghcr.io/sinedied/imapforward:latest
```

## Configuration

Create a `config.json` file. You can use the [online configuration generator](https://sinedied.github.io/imapforward/#config) to build it interactively.

```json
{
  "target": {
    "host": "imap.gmail.com",
    "port": 993,
    "secure": true,
    "auth": {
      "user": "your-email@gmail.com",
      "pass": "your-app-password"
    }
  },
  "sources": [
    {
      "name": "Work Email",
      "host": "imap.work.com",
      "port": 993,
      "secure": true,
      "auth": {
        "user": "user@work.com",
        "pass": "password"
      },
      "folders": ["INBOX"],
      "deleteAfterForward": false
    }
  ]
}
```

### Configuration Reference

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `target.host` | string | yes | — | Target server hostname (IMAP or SMTP) |
| `target.port` | number | yes | — | Target server port |
| `target.secure` | boolean | no | port-based | Use TLS (defaults to `true` for ports 465/993) |
| `target.auth.user` | string | yes | — | Username (also used as SMTP sender/recipient) |
| `target.auth.pass` | string | yes | — | Password or app password |
| `target.folder` | string | no | `"INBOX"` | Default target mailbox folder (IMAP method only). Auto-created if it doesn't exist |
| `forwardMethod` | string | no | `"imap"` | Forwarding method: `"imap"` (APPEND) or `"smtp"` |
| `sources[].name` | string | yes | — | Display name for the source |
| `sources[].host` | string | yes | — | IMAP server hostname |
| `sources[].port` | number | yes | — | IMAP server port |
| `sources[].secure` | boolean | no | port-based | Use TLS (defaults to `true` for ports 465/993) |
| `sources[].auth.user` | string | yes | — | IMAP username |
| `sources[].auth.pass` | string | yes | — | IMAP password |
| `sources[].folders` | string[] | no | `["INBOX"]` | Folders to monitor |
| `sources[].deleteAfterForward` | boolean | no | `false` | Delete messages after forwarding |
| `sources[].targetFolder` | string | no | — | Target mailbox for this source. Falls back to `target.folder`. Auto-created if it doesn't exist |
| `healthCheck.port` | number | no | `8080` | HTTP health check server port |

> [!TIP]
> For Gmail, you need to have 2FA enabled and use an [App Password](https://support.google.com/accounts/answer/185833) instead of your regular password.

### Forwarding Methods

imapforward supports two forwarding methods:

#### IMAP APPEND (default)

```json
{ "forwardMethod": "imap" }
```

Appends the raw RFC822 message directly to the target mailbox via IMAP. This preserves **all** original headers exactly — From, To, CC, Reply-To, Date, Message-ID, etc. Replies and Reply-All preserve the original sender and recipients. However, since messages bypass Gmail's intake pipeline, **spam filtering is not applied**.

#### SMTP Forward

```json
{
  "target": {
    "host": "smtp.gmail.com",
    "port": 587,
    "auth": { "user": "you@gmail.com", "pass": "your-app-password" }
  },
  "forwardMethod": "smtp"
}
```

Forwards messages via SMTP. Gmail's spam filters process the message normally. A `Reply-To` header is automatically added (if not present) pointing to the original `From` address, so Reply and Reply-All work with the original sender and recipients. Gmail may rewrite the `From` header to match the authenticated sender.

## Usage

### CLI

```bash
# Run with default config.json in current directory
imapforward

# Set log level
imapforward -log-level debug

# Custom config path
imapforward -config /path/to/config.json
```

### CLI Options

| Option | Description |
|--------|-------------|
| `-config <path>` | Config file path (default: `config.json`) |
| `-log-level <level>` | Log level: `debug`, `info`, `warn`, `error` (default: `info`) |
| `-version` | Show version |
| `-help` | Show help |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `IMAPFORWARD_CONFIG` | Override config file path |
| `LOG_LEVEL` | Override log level |

## Docker

### Run with Docker

```bash
docker run -d \
  --name imapforward \
  -v $(pwd)/config.json:/app/config.json:ro \
  ghcr.io/sinedied/imapforward:latest
```

### Docker Compose

```yaml
services:
  imapforward:
    image: ghcr.io/sinedied/imapforward:latest
    restart: unless-stopped
    volumes:
      - ./config.json:/app/config.json:ro
```

### Health Check

The health check server is always enabled (default port `8080`). You can customize the port in your config:

```json
{
  "healthCheck": {
    "port": 9090
  }
}
```

The HTTP health endpoint is available at:

```bash
curl http://localhost:8080/health
```

Response:

```json
{
  "status": "ok",
  "sources": [
    {
      "name": "Work Email",
      "connected": true,
      "lastSync": "2026-02-25T10:30:00.000Z",
      "error": null
    }
  ]
}
```

Status values: `ok` (all connected), `degraded` (some connected), `error` (none connected).

## How It Works

1. Connects to each configured IMAP source account (one connection per folder)
2. Scans for unseen messages that haven't been forwarded yet
3. Forwards each message to the target using the configured method:
   - **IMAP**: Appends raw RFC822 to the target mailbox, preserving all original headers
   - **SMTP**: Sends via SMTP with Reply-To injection for reply/reply-all support
4. Marks forwarded messages with a `$Forwarded` IMAP flag
5. Enters IMAP IDLE mode to watch for new messages in real-time
6. Automatically reconnects with exponential backoff on connection loss
