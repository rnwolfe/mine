---
type: reference
title: Env Encryption Spec
created: 2026-02-18
tags:
  - env
  - encryption
  - security
related:
  - "[[env-profiles]]"
  - "[[vault-encryption]]"
---

# Env Encryption Spec

Internal design document for encrypted env profile storage used by `mine env`.

## File Format

- Base directory: `$XDG_DATA_HOME/mine/envs/` (default `~/.local/share/mine/envs/`)
- Profile filename: `<profile>.age`
- Profile plaintext payload (before encryption): JSON object

```json
{
  "vars": {
    "API_URL": "https://api.example.com",
    "API_TOKEN": "secret"
  }
}
```

## Encryption Scheme

- Library: `filippo.io/age` with ASCII armor (`filippo.io/age/armor`)
- Mode: passphrase-based `age.NewScryptRecipient` / `age.NewScryptIdentity`
- Input passphrase sources:
1. `MINE_ENV_PASSPHRASE`
2. `MINE_VAULT_PASSPHRASE`
3. Interactive prompt via `term.ReadPassword` when TTY is available

If no passphrase is available in non-interactive mode, commands fail fast.

## Write Protocol

Profile writes are atomic:

1. Validate profile name and env keys
2. Marshal payload JSON
3. Encrypt payload with age scrypt recipient
4. Write to temp file in target directory
5. Rename temp file to final `.age` path

This prevents partial/corrupt profile writes on failures.

## Validation Rules

- Key format: `[A-Za-z_][A-Za-z0-9_]*`
- Profile name format: `[a-zA-Z0-9][a-zA-Z0-9._-]*`

Invalid keys or profile names are rejected before disk writes.

## Error Handling

- Wrong passphrase: return `ErrWrongPassphrase`
- Corrupted profile payload: return `ErrCorruptedProfile`
- Missing profile: surface `os.ErrNotExist` to caller

No silent fallback to plaintext storage.

