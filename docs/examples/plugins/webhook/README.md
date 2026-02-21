# webhook

A Python-based mine plugin that sends webhook notifications on todo events.

## What it demonstrates

- Transform hook (`preexec` on `todo.add`) — validates input before execution
- Notify hook (`notify` on `todo.done`) — fire-and-forget HTTP POST
- Network permission declaration
- Env var permission declaration (`WEBHOOK_URL`)
- Error protocol (JSON to stderr, non-zero exit)
- Protocol version checking
- Custom command for configuration help

## Install

```sh
mine plugin install ./docs/examples/plugins/webhook
```

## Usage

Set the webhook URL and use mine as normal:

```sh
export WEBHOOK_URL="https://example.com/hook"
mine todo add "ship feature"
mine todo done 1
```

Run the config command to check setup:

```sh
mine webhook help
```

## How it works

On `todo add`, the preexec transform hook validates that the todo text is
non-empty. If validation fails, the command is blocked with an error.

On `todo done`, the notify hook POSTs the context to `WEBHOOK_URL`. Since it
runs as a notify hook, failures are silent and don't block command completion.

## Files

| File | Purpose |
|------|---------|
| `mine-plugin.toml` | Plugin manifest with network and env_vars permissions |
| `mine-plugin-webhook` | Plugin binary (Python 3 script) |
