# todo-stats

A minimal shell-based mine plugin that tracks todo completions and shows a summary.

## What it demonstrates

- Simplest possible plugin (shell script, no dependencies beyond coreutils)
- Notify hook on `todo.done` (fire-and-forget, appends to a log file)
- Custom `summary` command that reads the log file
- Lifecycle `health` handler

## Install

```sh
mine plugin install ./docs/examples/plugins/todo-stats
```

## Usage

Complete some todos, then run:

```sh
mine todo-stats summary
```

## How it works

Every time you complete a todo (`mine todo done`), the notify hook appends a
timestamp to `~/.local/share/mine/todo-stats.log`. The `summary` command reads
that file and prints a count with first/last completion dates.

## Files

| File | Purpose |
|------|---------|
| `mine-plugin.toml` | Plugin manifest |
| `mine-plugin-todo-stats` | Plugin binary (shell script) |
