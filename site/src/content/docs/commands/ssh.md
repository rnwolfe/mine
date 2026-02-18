---
title: mine ssh
description: SSH config management, key management, tunnels, and connection helpers
---

Manage SSH hosts, keys, and tunnels with `~/.ssh/config` as the source of truth.

## Fuzzy Host Picker

```bash
mine ssh
```

Opens an interactive fuzzy-searchable list of configured SSH hosts. Select a host to connect. Falls back to a plain list when no TTY is detected (e.g. in scripts).

## List Hosts

```bash
mine ssh hosts
```

Lists all non-wildcard Host entries from `~/.ssh/config`.

## Add a Host

```bash
mine ssh add myserver
mine ssh add          # prompts for alias
```

Interactively generates a Host block and appends it to `~/.ssh/config`. Prompts for:
- **Alias** — the name you'll use to connect (`ssh myserver`)
- **HostName** — IP address or FQDN
- **User** — remote username
- **Port** — defaults to 22 (omitted from config if default)
- **IdentityFile** — path to SSH key (optional)

## Remove a Host

```bash
mine ssh remove myserver
```

Cleanly removes the named Host block from `~/.ssh/config` without corrupting the file.

## Generate a Key

```bash
mine ssh keygen                # creates ~/.ssh/id_ed25519
mine ssh keygen deploy         # creates ~/.ssh/deploy
```

Generates an ed25519 SSH key pair with sensible defaults (no passphrase). Uses `ssh-keygen` under the hood.

## Copy Key to Remote

```bash
mine ssh copyid myserver
```

Copies your default public key to a remote host using `ssh-copy-id`.

## List Keys

```bash
mine ssh keys
```

Lists all SSH key pairs in `~/.ssh/` with:
- Fingerprint (SHA256)
- Which configured hosts use each key
- Path to the private key

## Start a Tunnel

```bash
mine ssh tunnel myserver 8080:80      # local 8080 → remote 80
mine ssh tunnel db 5433:5432          # local 5433 → remote 5432
```

Starts an SSH port-forwarding tunnel in the foreground. Press `Ctrl+C` to stop. Uses `ssh -N -L` with `ExitOnForwardFailure=yes`.

## Shell Functions

The following SSH helper functions are included in `mine shell init`:

| Function | Description |
|----------|-------------|
| `sc <alias>` | Quick connect: `ssh <alias>` |
| `scp2 <src> <dest>` | Resumable copy: `rsync -avzP --partial` over SSH |
| `stun <alias> <L:R>` | Quick tunnel: `ssh -N -L local:localhost:remote alias` |
| `skey [file]` | Copy default public key to clipboard |

All functions include `--help` for usage documentation and work in bash, zsh, and fish.

### Usage examples

```bash
# Connect to a host
sc myserver

# Copy files with progress and resume
scp2 myserver:/var/log/app.log ./logs/

# Start a tunnel (local 8080 → remote 80)
stun myserver 8080:80

# Copy your public key to clipboard
skey
skey ~/.ssh/id_ed25519.pub
```

## Hook Integration

Every `mine ssh` command fires hook events, allowing plugin hooks to intercept SSH workflows:

| Command | Hook name |
|---------|-----------|
| `mine ssh` | `ssh` |
| `mine ssh hosts` | `ssh.hosts` |
| `mine ssh add` | `ssh.add` |
| `mine ssh remove` | `ssh.remove` |
| `mine ssh keygen` | `ssh.keygen` |
| `mine ssh copyid` | `ssh.copyid` |
| `mine ssh keys` | `ssh.keys` |
| `mine ssh tunnel` | `ssh.tunnel` |

See [hooks](/commands/hook) for how to create and register hooks.

## Examples

```bash
# Set up a new server
mine ssh add staging
mine ssh keygen staging
mine ssh copyid staging
sc staging

# Quick tunnel to a database
stun db 5433:5432

# Audit your keys
mine ssh keys
```
