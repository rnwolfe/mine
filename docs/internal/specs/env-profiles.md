---
type: reference
title: Env Profiles Spec
created: 2026-02-18
tags:
  - env
  - profiles
  - storage
related:
  - "[[env-encryption]]"
---

# Env Profiles Spec

Internal design document for env profile naming, layout, and active-profile association.

## Storage Layout

Each project gets an isolated profile directory under `~/.local/share/mine/envs/`.
Directory names are derived from `sha256(project_path)` in hex form.

```text
~/.local/share/mine/envs/
  <sha256(project_path)>/
    local.age
    staging.age
    prod.age
```

This avoids storing raw project paths on disk while keeping deterministic lookup.

## Active Profile Tracking

Active profile per project is stored in SQLite table `env_projects`:

- `project_path` (TEXT, primary key)
- `active_profile` (TEXT)
- `updated_at` (timestamp)

Behavior:

- If a project row does not exist, active profile defaults to `local`
- `env switch <name>` verifies the profile exists, then updates `env_projects`

## Profile Operations

- `CurrentProfile(project)`:
1. Read active profile from `env_projects` or default to `local`
2. Load encrypted profile from disk
3. For default profile, missing file resolves to empty map

- `SetVar(project, profile, key, value)`:
1. Load existing profile or create empty map if missing
2. Upsert key/value
3. Save encrypted profile

- `UnsetVar(project, profile, key)`:
1. Load profile
2. Delete key
3. Save encrypted profile

- `Diff(project, a, b)`:
1. Load both profiles
2. Return sorted key-only sets: `Added`, `Removed`, `Changed`

- `ExportLines(project, profile, shell)`:
1. Load profile
2. Return deterministic sorted lines
3. Use shell-specific quoting for `posix` or `fish`

## Shell Integration Contract

`mine shell init` provides `menv`:

- Bash/Zsh: runs `mine env export` then `eval`
- Fish: runs `mine env export --shell fish` then `source`

`menv` intentionally mutates only the current shell session.

