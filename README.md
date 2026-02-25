<div align="center">

# 📬 imapforward

[![Build Status](https://img.shields.io/github/actions/workflow/status/sinedied/imapforward/ci.yml?style=flat-square)](https://github.com/sinedied/imapforward/actions)
[![npm version](https://img.shields.io/npm/v/imapforward?style=flat-square)](https://www.npmjs.com/package/imapforward)
[![Docker](https://img.shields.io/badge/ghcr.io-imapforward-blue?style=flat-square)](https://github.com/sinedied/imapforward/pkgs/container/imapforward)
[![License](https://img.shields.io/github/license/sinedied/imapforward?style=flat-square)](LICENSE)

A simple IMAP email forwarder for syncing multiple email accounts into Gmail.

[Features](#features) · [Installation](#installation) · [Configuration](#configuration) · [Docker](#docker)

</div>

## Features

- **Real-time sync** — Uses IMAP IDLE for instant email forwarding
- **Multiple sources** — Forward from multiple email accounts to a single Gmail
- **Original headers preserved** — Emails arrive with their original From, Reply-To, and other headers intact
- **Selective folders** — Choose which folders to sync (e.g. only INBOX)
- **Auto cleanup** — Optionally delete messages after successful forwarding
- **Production-grade** — Auto reconnect with exponential backoff, health check endpoint
- **Minimal footprint** — Only 1 runtime dependency (`imapflow`)
- **Docker ready** — Alpine-based image with built-in health checks

## Installation

### npm

```bash
npm install -g imapforward
```

### Docker

```bash
docker pull ghcr.io/sinedied/imapforward:latest
```

## Configuration

Create a `config.json` file:

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
  ],
  "healthCheck": {
    "port": 8080
  }
}
```

### Configuration Reference

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `target.host` | string | yes | — | Target IMAP server hostname |
| `target.port` | number | yes | — | Target IMAP server port |
| `target.secure` | boolean | no | port-based | Use TLS (defaults to `true` for ports 465/993) |
| `target.auth.user` | string | yes | — | Target IMAP username |
| `target.auth.pass` | string | yes | — | Target IMAP password or app password |
| `target.folder` | string | no | `"INBOX"` | Target mailbox folder to append messages to |
| `sources[].name` | string | yes | — | Display name for the source |
| `sources[].host` | string | yes | — | IMAP server hostname |
| `sources[].port` | number | yes | — | IMAP server port |
| `sources[].secure` | boolean | yes | — | Use TLS |
| `sources[].auth.user` | string | yes | — | IMAP username |
| `sources[].auth.pass` | string | yes | — | IMAP password |
| `sources[].folders` | string[] | no | `["INBOX"]` | Folders to monitor |
| `sources[].deleteAfterForward` | boolean | no | `false` | Delete messages after forwarding |
| `healthCheck.port` | number | no | — | HTTP health check port (disabled if omitted) |

> [!TIP]
> For Gmail, you need to have 2FA enabled and use an [App Password](https://support.google.com/accounts/answer/185833) instead of your regular password.

## Usage

### CLI

```bash
# Using default config.json in current directory
imapforward

# Custom config path
imapforward --config /path/to/config.json

# Set log level
imapforward --log-level debug
```

### CLI Options

| Option | Short | Description |
|--------|-------|-------------|
| `--config <path>` | `-c` | Config file path (default: `config.json`) |
| `--log-level <level>` | `-l` | Log level: `debug`, `info`, `warn`, `error` (default: `info`) |
| `--version` | `-v` | Show version |
| `--help` | `-h` | Show help |

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

When `healthCheck.port` is configured, an HTTP health endpoint is available:

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

1. Connects to each configured IMAP source account
2. Scans for unseen messages that haven't been forwarded yet
3. Appends each message to the target mailbox via IMAP preserving all original headers (raw RFC822)
4. Marks forwarded messages with a `$Forwarded` IMAP flag
5. Enters IMAP IDLE mode to watch for new messages in real-time
6. Automatically reconnects with exponential backoff on connection loss

## Development

```bash
# Install dependencies
npm install

# Run in development mode
npm run dev -- --config config.json

# Build
npm run build

# Lint
npm run lint

# Test
npm test
```
