---
title: Privacy & Analytics
description: What mine collects, why, and how to opt out.
---

mine includes lightweight, anonymous usage analytics to help us understand adoption patterns and prioritize features. **Privacy is a first-class concern** — we collect the absolute minimum and never touch your personal data.

## What we collect

| Field | Example | Why |
|-------|---------|-----|
| Installation ID | `a1b2c3d4-...` (random UUID) | Count unique installs without identifying you |
| mine version | `0.1.0` | Track release adoption |
| OS / Architecture | `linux/amd64` | Know which platforms to prioritize |
| Command name | `todo`, `craft` | Understand feature usage (never arguments or data) |
| Date | `2026-02-17` | Day-granularity activity, not exact timestamps |

## What we NEVER collect

- Command arguments, flags, or values
- Todo content, file paths, config values, or secrets
- IP addresses (our ingest strips them)
- Anything that could identify you personally

## How it works

- Analytics run **after** each command with a short (2-second) HTTP timeout. Thanks to daily dedup, the network is almost never hit.
- **Daily dedup**: only one ping per command per day — not per invocation.
- **Fails silently**: if you're offline or the endpoint is unreachable, nothing happens.
- The installation ID is a random UUIDv4 stored in `~/.local/share/mine/analytics_id`. It is not tied to your identity in any way.

## Opting out

Disable analytics with a single command:

```bash
mine config set analytics false
```

Re-enable anytime:

```bash
mine config set analytics true
```

When analytics are disabled, `mine` makes zero network requests.

## Data schema

The JSON payload sent to our analytics endpoint:

```json
{
  "install_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "version": "0.1.0",
  "os": "linux",
  "arch": "amd64",
  "command": "todo",
  "date": "2026-02-17"
}
```

No additional fields are ever added without updating this page.
