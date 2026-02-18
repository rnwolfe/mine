---
title: mine git
description: Git workflow supercharger — a fast convenience layer over common git operations
---

Git workflow supercharger. Not a git replacement — a git accelerator. Shells out to the
`git` binary with no extra dependencies.

## Branch Picker (bare command)

```bash
mine git
```

Opens a fuzzy-searchable branch picker (Bubbletea). Pick a branch and press Enter to
switch. Falls back to a plain list when no TTY is available (e.g., scripts, CI).

```bash
mine git    # interactive picker
mine g      # same, via alias
```

## mine git sweep

Delete merged local branches and prune stale remote-tracking refs.

```bash
mine git sweep
```

- Only deletes branches already merged into the current branch
- Skips protected branches: `main`, `master`, `develop`, and the current branch
- Confirms before deleting
- Also prunes stale `origin/*` refs

## mine git undo

Soft-reset the last commit, keeping all changes staged.

```bash
mine git undo
```

Shows the commit message and prompts for confirmation before resetting.

## mine git wip

Stage everything and create a quick WIP commit.

```bash
mine git wip
```

Equivalent to `git add -A && git commit -m "wip"`. Use `mine git unwip` to undo.

## mine git unwip

Undo the last commit if its message is "wip".

```bash
mine git unwip
```

Fails with a clear error if the last commit is not a WIP commit.

## mine git pr

Create a pull request from the current branch.

```bash
mine git pr
```

- Auto-detects the base branch (`main`, `master`, or `develop`)
- Generates a PR title from the branch name (e.g. `feat/add-oauth` → `feat: add oauth`)
- Generates a PR body from the commit log
- Uses the `gh` CLI to create the PR if available
- Falls back to printing the generated title/body if `gh` is not installed

## mine git log

Pretty commit log — compact, colored, graph.

```bash
mine git log
```

Shows the last 30 commits with author, subject, relative date, and branch graph.

## mine git changelog

Generate a Markdown changelog from conventional commits between two refs.

```bash
mine git changelog                    # auto-detect base..HEAD
mine git changelog --from v1.0.0
mine git changelog --from v1.0.0 --to v2.0.0
```

Commits are grouped into sections: Features, Bug Fixes, Documentation, Refactoring,
Chores, and Other. Only sections with commits are included.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--from`, `-f` | auto (base branch) | Start ref |
| `--to`, `-t` | `HEAD` | End ref |

## mine git aliases

Install opinionated git aliases to `~/.gitconfig`.

```bash
mine git aliases
```

Installs these aliases (with confirmation):

| Alias | Command | Description |
|-------|---------|-------------|
| `git co` | `checkout` | Shortcut for checkout |
| `git br` | `branch` | Shortcut for branch |
| `git st` | `status -sb` | Compact status |
| `git lg` | `log --oneline --graph --decorate --all` | Pretty graph log |
| `git last` | `log -1 HEAD --stat` | Last commit with stats |
| `git unstage` | `reset HEAD --` | Unstage a file |
| `git undo` | `reset --soft HEAD~1` | Soft undo last commit |
| `git wip` | `!git add -A && git commit -m "wip"` | Quick WIP commit |
| `git aliases` | `config --get-regexp alias` | List all aliases |

## Shell Functions

When you run `eval "$(mine shell init)"`, these git helper functions are added to your
shell:

| Function | Description |
|----------|-------------|
| `gc <msg>` | `git commit -m` shorthand |
| `gca <msg>` | `git commit --amend -m` shorthand |
| `gp` | `git push` with upstream tracking |
| `gpl` | `git pull --rebase` |
| `gsw <branch>` | `git switch` shorthand |

All functions work in bash, zsh, and fish.

## Examples

```bash
# Switch branches interactively
mine git

# Clean up after a merge
mine git sweep

# Save work in progress
mine git wip
# ... later ...
mine git unwip

# Open a PR
mine git pr

# Generate a changelog for a release
mine git changelog --from v1.2.0

# Pretty log
mine git log

# Install git aliases
mine git aliases
```
