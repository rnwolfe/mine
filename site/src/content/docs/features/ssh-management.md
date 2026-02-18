---
title: SSH Management
description: Manage SSH hosts, keys, and tunnels with a fuzzy picker and shell helpers
---

Stop grepping through `~/.ssh/config`. `mine ssh` gives you a fuzzy host picker, key management, tunnel shortcuts, and shell helper functions — all using your existing SSH config as the source of truth.

## Key Capabilities

- **Fuzzy host picker** — interactive searchable list of configured SSH hosts
- **Host management** — add and remove hosts from `~/.ssh/config` interactively
- **Key management** — generate ed25519 keys, list key fingerprints, copy keys to remotes
- **Port forwarding** — start SSH tunnels with a simple `host local:remote` syntax
- **Shell functions** — `sc`, `scp2`, `stun`, `skey` shortcuts via `mine shell init`

## Quick Example

```bash
# Connect to a host interactively
mine ssh

# Set up a new server
mine ssh add staging
mine ssh keygen staging
mine ssh copyid staging

# Start a tunnel
mine ssh tunnel db 5433:5432
```

## How It Works

The bare `mine ssh` command opens a fuzzy picker showing all your configured hosts — type to filter, Enter to connect. For setup, `mine ssh add` walks you through creating a host block interactively (alias, hostname, user, port, identity file) and appends it to `~/.ssh/config`.

Key management wraps `ssh-keygen` with secure defaults (ed25519). `mine ssh keys` lists all your key pairs with fingerprints and shows which hosts use each key. Tunnels use `ssh -N -L` with `ExitOnForwardFailure` for reliable port forwarding.

The shell functions from `mine shell init` add one-liners: `sc myserver` to connect, `scp2` for resumable copies, `stun` for quick tunnels, and `skey` to copy your public key to the clipboard.

## Learn More

See the [command reference](/commands/ssh/) for all subcommands, flags, and detailed usage.
