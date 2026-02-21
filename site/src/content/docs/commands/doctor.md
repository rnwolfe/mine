---
title: mine doctor
description: Check your mine setup for problems and get actionable fixes
---

Run a suite of health checks and report what's working — and what isn't — with actionable fix suggestions.

## Usage

```bash
mine doctor
```

## Output

```
  ✓  Config           ~/.config/mine/config.toml found and valid
  ✓  Store            SQLite database opens and responds
  ✓  Git              git 2.43.0 found in PATH
  ✗  Shell helpers    shell integration not detected
                      → Run mine init to install shell helpers (p, pp, menv)
  ✓  AI               Provider: claude (claude-sonnet-4-5)
  ✓  Analytics        Enabled (opt out: mine config set analytics false)
```

Exit code is `0` if all checks pass, `1` if any check fails.

## Checks

| Check | Pass Condition | Fix on Failure |
|-------|---------------|----------------|
| **Config** | `~/.config/mine/config.toml` exists and parses without error | Run `mine init` |
| **Store** | SQLite database opens and responds to queries | Re-run `mine init` or check disk space |
| **Git** | `git` is found in `$PATH` | Install git from your package manager |
| **Shell helpers** | `mine init` was completed and user name is set | Run `mine init` to install `p`, `pp`, `menv` |
| **AI** | An AI provider is configured | Run `mine ai config` |
| **Analytics** | Always passes — shows current status | Opt out: `mine config set analytics false` |

## Examples

```bash
# Run all checks
mine doctor

# Pipe output for scripting (exit code signals pass/fail)
mine doctor && echo "all good"
```

## Tips

- Run `mine doctor` after any major system change (new machine, shell switch, mine upgrade).
- If **Shell helpers** fails, run `mine init` — it will detect that everything else is set up and only install the shell integration.
- The **Analytics** check always passes; it's informational only.
