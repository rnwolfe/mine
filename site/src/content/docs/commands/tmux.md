---
title: mine tmux
description: Tmux session management, layout persistence, and fuzzy session picker
---

Manage tmux sessions, attach, and save/restore window layouts. Requires `tmux` to be installed.

## Fuzzy Session Picker

```bash
mine tmux       # interactive picker (TTY) or plain list (piped)
mine tx         # alias
```

Opens a fuzzy-searchable list of running tmux sessions. Select a session and press Enter to attach. Falls back to a plain list when stdout is not a TTY.

## Create a Session

```bash
mine tmux new           # auto-names from current directory
mine tmux new myproject # explicit name
```

Creates a new tmux session. If no name is given, the session is named after the current directory.

## List Sessions

```bash
mine tmux ls
mine tmux list
```

Lists all running tmux sessions with window counts. Attached sessions are marked with `*`.

## Attach to a Session

```bash
mine tmux attach myproject   # fuzzy match by name
mine tmux a proj             # short alias, fuzzy matches "myproject"
```

Attaches or switches to a session. The name is fuzzy-matched against running sessions, so partial names work. Without a name, opens the interactive picker.

## Kill a Session

```bash
mine tmux kill myproject     # fuzzy match by name
mine tmux kill               # interactive picker
```

Kills a tmux session. Supports fuzzy matching and the interactive picker.

## Rename a Session

```bash
mine tmux rename old new     # rename directly, no prompts
mine tmux rename oldname     # fuzzy-match session, then prompt for new name
mine tmux rename             # interactive picker, then prompt for new name
```

Renames a tmux session. In 2-arg mode the rename happens immediately. In 1-arg mode the session is fuzzy-matched by name and you are prompted for the new name. With no args, an interactive picker lets you select the session and then prompts for the new name.

## Layouts

Save and restore window/pane layouts.

### Save a Layout

```bash
mine tmux layout save dev-setup
```

Saves the current tmux session's window and pane layout. Must be run from inside a tmux session.

### Load a Layout

```bash
mine tmux layout load dev-setup   # load by name
mine tmux layout load             # interactive picker (TTY) or list (piped)
```

Restores a previously saved layout. Must be run from inside a tmux session. Without a name, opens an interactive fuzzy picker over saved layouts. Falls back to listing available layout names when stdout is not a TTY.

### List Saved Layouts

```bash
mine tmux layout ls
mine tmux layout list
```

Lists all saved layouts with window counts, window names, and save timestamps.

## Examples

```bash
# Create and attach to a project session
mine tmux new myapi

# Save your dev layout (3 panes: editor, server, tests)
mine tmux layout save dev-3pane

# Later, restore it
mine tmux layout load dev-3pane

# Switch between sessions
mine tmux

# Rename a session
mine tmux rename old-name new-name

# Clean up
mine tmux kill old-project
```
