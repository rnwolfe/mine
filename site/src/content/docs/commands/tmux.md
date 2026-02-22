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
mine tmux new                        # auto-names from current directory
mine tmux new myproject              # explicit name
mine tmux new myproject --layout dev # create and immediately apply a saved layout
```

Creates a new tmux session. If no name is given, the session is named after the current directory.

Use `--layout <name>` to apply a saved layout immediately after the session is created. If the named layout does not exist, the session is **not** created and an error is printed — no side effects.

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

### Interactive Layout Picker

```bash
mine tmux layout              # inside tmux + TTY: fuzzy picker; else: help text
```

When run inside a tmux session with a TTY, `mine tmux layout` (bare, no subcommand) opens an interactive fuzzy-search picker over all saved layouts. Selecting a layout immediately loads it into the current session. Press Esc to cancel without applying any layout.

Outside of tmux or when stdout is not a TTY, the command falls back to showing the subcommand help text. All subcommands (`save`, `load`, `ls`, `preview`, `delete`) are always accessible directly.

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

### Delete a Layout

```bash
mine tmux layout delete dev-setup   # delete by name
mine tmux layout delete             # interactive picker (TTY)
```

Permanently removes a saved layout. With no arguments, opens an interactive picker to select which layout to delete. Returns an error if the named layout does not exist.

## Windows

Manage windows within a tmux session. All window subcommands default to the current session when inside tmux. Use `--session <name>` to target a specific session.

### List Windows

```bash
mine tmux window ls             # windows in current session
mine tmux window ls --session s # windows in session "s"
mine tmux window list           # alias
```

Lists all windows in the session with their index, name, and active indicator (`*`).

### Create a Window

```bash
mine tmux window new <name>
mine tmux window new editor --session myproject
```

Creates a new named window in the current (or specified) session.

### Kill a Window

```bash
mine tmux window kill <name>     # kill by exact name
mine tmux window kill            # interactive picker (TTY)
mine tmux window kill --session s editor
```

Kills a window. With no name, opens an interactive fuzzy picker to select the window. Falls back to listing windows when stdout is not a TTY.

### Rename a Window

```bash
mine tmux window rename old new  # rename directly, no prompts
mine tmux window rename oldname  # select by name, then prompt for new name
mine tmux window rename          # interactive picker, then prompt for new name
mine tmux window rename --session s editor code
```

Renames a window. In 2-arg mode the rename is immediate. In 1-arg mode the window is matched by exact name and you are prompted for the new name. With no args, an interactive picker lets you select the window and then prompts for the new name.

### Flags

| Flag | Description |
|------|-------------|
| `--session <name>` | Target session (defaults to current session inside tmux) |

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

# Create a new session and immediately apply your saved layout
mine tmux new myapi --layout dev-3pane

# Preview a layout before loading it
mine tmux layout preview dev-3pane

# Restore a layout — interactive picker inside tmux, or specify by name
mine tmux layout
mine tmux layout load dev-3pane

# Remove a layout you no longer need
mine tmux layout delete dev-3pane

# Manage windows within a session
mine tmux window ls
mine tmux window new editor
mine tmux window kill old-window
mine tmux window rename old-name new-name

# Switch between sessions
mine tmux

# Rename a session
mine tmux rename old-name new-name

# Clean up
mine tmux kill old-project
```
