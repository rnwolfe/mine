# Plugin Manifest Specification

> Internal spec for `mine-plugin.toml` — the manifest format for mine plugins.

## Overview

Every mine plugin must include a `mine-plugin.toml` file at its repository root.
This manifest declares metadata, hooks, commands, and permissions.

## Schema

```toml
# Required metadata
[plugin]
name = "obsidian-sync"                    # unique identifier (kebab-case)
version = "0.2.0"                         # semver
description = "Sync todos to Obsidian"    # one-line description
author = "rnwolfe"                        # GitHub username
license = "MIT"                           # SPDX identifier
min_mine_version = "0.2.0"               # minimum compatible mine version
protocol_version = "1.0.0"               # protocol version this plugin targets

# Optional: binary entry point (defaults to "mine-plugin-" + name)
entrypoint = "mine-plugin-obsidian-sync"

# Hook registrations
[[hooks]]
command = "todo.add"                      # command pattern (supports wildcards)
stage = "postexec"                        # prevalidate | preexec | postexec | notify
mode = "notify"                           # transform | notify
timeout = "10s"                           # optional, overrides default

[[hooks]]
command = "todo.done"
stage = "notify"
mode = "notify"

# Custom command registrations
[[commands]]
name = "sync"                             # invoked as `mine obsidian sync`
description = "Sync todos to Obsidian vault"
args = "[--vault <path>]"                 # usage hint

# Permission declarations
[permissions]
network = true                            # needs outbound network access
filesystem = ["~/.obsidian", "~/notes"]   # allowed filesystem paths
store = false                             # access to mine's SQLite store
config_read = true                        # read mine config
config_write = false                      # write mine config
env_vars = ["OBSIDIAN_VAULT"]             # allowed environment variables
```

## Field Reference

### [plugin] — Required

| Field              | Type   | Required | Description |
|--------------------|--------|----------|-------------|
| name               | string | yes      | Unique plugin identifier (kebab-case) |
| version            | string | yes      | Semver version |
| description        | string | yes      | One-line description |
| author             | string | yes      | GitHub username |
| license            | string | no       | SPDX license identifier |
| min_mine_version   | string | no       | Minimum compatible mine version |
| protocol_version   | string | yes      | Protocol version (currently "1.0.0") |
| entrypoint         | string | no       | Binary name (defaults to `mine-plugin-` + name) |

### [[hooks]] — Optional, repeatable

| Field   | Type   | Required | Description |
|---------|--------|----------|-------------|
| command | string | yes      | Command pattern with wildcard support |
| stage   | string | yes      | Pipeline stage |
| mode    | string | yes      | `transform` or `notify` |
| timeout | string | no       | Override default timeout (e.g. "10s") |

### [[commands]] — Optional, repeatable

| Field       | Type   | Required | Description |
|-------------|--------|----------|-------------|
| name        | string | yes      | Command name (under plugin namespace) |
| description | string | yes      | Help text |
| args        | string | no       | Usage hint for arguments |

### [permissions] — Optional

| Field        | Type     | Default | Description |
|--------------|----------|---------|-------------|
| network      | bool     | false   | Outbound network access |
| filesystem   | []string | []      | Allowed filesystem paths |
| store        | bool     | false   | Access to mine's data store |
| config_read  | bool     | false   | Read mine's config |
| config_write | bool     | false   | Write mine's config |
| env_vars     | []string | []      | Allowed environment variables |

## Versioning Strategy

- Plugin version follows semver independently of mine version
- `protocol_version` tracks the JSON protocol compatibility
- `min_mine_version` declares minimum compatible mine version
- Breaking protocol changes bump the major protocol version
