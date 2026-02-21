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

## Project Session (Create or Attach)

```bash
mine tmux project                    # session named after current directory
mine tmux project ~/code/myapp       # session named "myapp"
mine tmux project --layout dev-setup # create with a saved layout applied
mine tmux proj                       # alias
```

Creates a tmux session named after the target directory's basename, or attaches if a session with that name is already running. This single command replaces the `mine tmux ls` + conditional `new`/`attach` workflow for project-based sessions.

- If the session **does not exist**: it is created and you are attached. With `--layout`, the saved layout is applied to the new session before attaching.
- If the session **already exists**: you are attached directly. The `--layout` flag is ignored on attach.
- The `--layout` value is pre-validated — an error is returned immediately if the layout does not exist.

**Shell helper** — `tp` wraps this command for quick use:

```bash
tp             # project session for cwd
tp ~/code/api  # project session for ~/code/api
```

Add `tp` to your shell with `mine shell init`.

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

### Preview a Layout

```bash
mine tmux layout preview dev-setup
```

Displays a layout's name, save timestamp, and a table of windows with their pane counts and directories — without loading or modifying any tmux session. Works outside of tmux, making it useful for verifying you have the right layout before running `layout load`.

## Examples

```bash
# Create or attach to a project session (recommended workflow)
mine tmux project ~/code/myapi

# Same thing, from inside the project directory
cd ~/code/myapi && mine tmux project

# Create and attach, applying a saved layout
mine tmux project ~/code/myapi --layout dev-3pane

# Use the shell helper
tp ~/code/myapi

# Create and attach to a project session (explicit)
mine tmux new myapi

# Save your dev layout (3 panes: editor, server, tests)
mine tmux layout save dev-3pane

# Preview a layout before loading it
mine tmux layout preview dev-3pane

# Later, restore it
mine tmux layout load dev-3pane

# Switch between sessions
mine tmux

# Rename a session
mine tmux rename old-name new-name

# Clean up
mine tmux kill old-project
```
