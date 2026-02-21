# tag-enforcer

A Go-based mine plugin that enforces tagging policies on todos.

## What it demonstrates

- Native Go plugin with proper struct definitions matching the protocol
- Prevalidate stage — modifies context before validation runs
- Postexec stage — observes context after command execution
- Glob pattern matching (`todo.*` matches all todo subcommands)
- Custom `policy` command
- Config read permission
- Protocol version checking
- Error protocol (JSON to stderr, non-zero exit)

## Build

```sh
cd docs/examples/plugins/tag-enforcer
go build -o mine-plugin-tag-enforcer .
```

## Install

Build first, then install the directory:

```sh
mine plugin install ./docs/examples/plugins/tag-enforcer
```

## Usage

The plugin automatically adds an `untagged` tag to todos created without tags:

```sh
mine todo add "buy milk"              # gets tagged as "untagged"
mine todo add "buy milk" --tags work  # keeps the "work" tag
```

Show the current policy:

```sh
mine tag-enforcer policy
```

## How it works

At the `prevalidate` stage on `todo.add`, the plugin inspects the context's
flags for a `tags` key. If absent, it injects `tags=untagged` into the context
and returns the modified context as a transform response. This happens before
the todo is created, so the tag is part of the todo from the start.

At the `postexec` stage on `todo.*`, the plugin receives the context after
command execution. This is a passthrough — the context is returned unchanged.
In a real plugin, this would be a good place to log or audit.

## Files

| File | Purpose |
|------|---------|
| `mine-plugin.toml` | Plugin manifest with two hooks, one command, config permission |
| `main.go` | Plugin implementation |
| `go.mod` | Go module definition |
