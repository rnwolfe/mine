---
title: mine plugin
description: Install, manage, and discover mine plugins
---

Install, remove, and manage mine plugins. Plugins are standalone binaries that hook into the command pipeline and can register custom subcommands.

## List Installed Plugins

```bash
mine plugin         # list all installed plugins
mine plugin list    # same thing
```

Shows each plugin with a status indicator, name, version, and hook/command counts. Active plugins show a filled circle; disabled plugins show an empty circle.

## Install a Plugin

```bash
mine plugin install ./my-plugin
mine plugin install /path/to/mine-plugin-obsidian
```

Installs a plugin from a local directory containing a `mine-plugin.toml` manifest. mine reads the manifest, displays the requested permissions, and prompts for confirmation before installing.

The installation flow:

1. Parse `mine-plugin.toml` from the source directory
2. Validate the manifest (required fields, name format, stage/mode pairing)
3. Display plugin name, version, author, description
4. Display requested permissions for review
5. Prompt `Install this plugin? [y/N]`
6. Copy the plugin to `~/.local/share/mine/plugins/<name>/`
7. Register hooks and commands

### Example

```bash
$ mine plugin install ./todo-stats
  Installing todo-stats v0.1.0 by mine-examples
  Track todo completion stats and show a summary

  Permissions:
    No special permissions required

  Install this plugin? [y/N] y

  Installed todo-stats v0.1.0
  1 hooks registered, 1 commands available
```

## Remove a Plugin

```bash
mine plugin remove todo-stats
mine plugin rm todo-stats        # alias
mine plugin uninstall todo-stats # alias
```

Removes an installed plugin and unregisters its hooks and commands.

## Show Plugin Info

```bash
mine plugin info todo-stats
```

Displays detailed information about an installed plugin:

- Version, author, description, license
- Protocol version and install directory
- Enabled status
- Registered hooks (command pattern, stage, mode)
- Registered commands (name, description)
- Declared permissions

## Search for Plugins

```bash
mine plugin search obsidian        # search by keyword
mine plugin search                 # list all mine plugins
mine plugin search --tag logging   # filter by GitHub topic
```

Searches GitHub for repositories matching the `mine-plugin-*` naming convention. Results include the repository name, description, and star count.

### Rate Limits

GitHub's search API has rate limits for unauthenticated requests. Set `GITHUB_TOKEN` to increase your limit:

```bash
export GITHUB_TOKEN=ghp_...
mine plugin search obsidian
```

## Error Reference

| Error | Cause | Fix |
|-------|-------|-----|
| `reading manifest: open mine-plugin.toml: no such file or directory` | Source directory has no manifest | Ensure the directory contains `mine-plugin.toml` |
| `invalid manifest: plugin.name is required` | Manifest missing required field | Add the missing field to `mine-plugin.toml` |
| `invalid manifest: plugin.name "My Plugin" must be kebab-case` | Name not in kebab-case format | Use lowercase with hyphens (e.g., `my-plugin`) |
| `invalid manifest: hooks[0]: notify stage requires notify mode` | Stage/mode mismatch | Notify stage must use notify mode; all other stages use transform mode |
| `plugin "foo" not found` | Plugin not installed | Check spelling with `mine plugin list` or install it first |
| `Installation cancelled.` | User declined the permission prompt | Review the permissions and run `mine plugin install` again |

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `GITHUB_TOKEN` | Increases GitHub API rate limit for `mine plugin search` |

## See Also

- [Plugins feature overview](/features/plugins/) -- how the plugin system works
- [Building Plugins](/contributors/building-plugins/) -- create your own plugins
- [Plugin Protocol](/contributors/plugin-protocol/) -- the JSON communication contract
- [Hook command reference](/commands/hook/) -- user-local hooks (no plugin required)
