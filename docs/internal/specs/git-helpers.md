# Git Helpers Internal Spec

> Internal spec for `mine git` — sweep safety rules, PR generation logic, and design decisions.
> Reference implementation: `internal/git/git.go`, `cmd/git.go`.

## Sweep Safety Rules

`mine git sweep` deletes merged local branches. The following rules govern what is safe to delete:

### What is deleted

- Local branches returned by `git branch --merged` (i.e. fully merged into current HEAD)

### What is always protected (never deleted)

| Branch | Reason |
|--------|--------|
| `main` | Primary trunk branch |
| `master` | Legacy primary trunk name |
| `develop` | Common integration branch |
| Current branch | Cannot delete the branch you are on |

These are hardcoded in `MergedBranches()`. Future: allow user-configurable protected branches via config.

### Confirmation behavior

- Always shows list of branches to be deleted before acting
- Prompts `[y/N]` — default is No (safe default)
- Aborts on any response other than `y` or `yes`

### Remote pruning

After deleting local branches, `git remote prune origin` is called to remove stale `origin/*`
remote-tracking refs. This is safe: it only removes references to remote branches that no
longer exist on the remote.

## PR Generation Logic

`mine git pr` generates a PR using branch name and commit history.

### Title generation (`branchToTitle`)

Branch names are converted to PR titles using these rules:

1. Strip common prefixes: `feat/`, `fix/`, `chore/`, `docs/`, `refactor/`, `test/`
2. Map prefix to conventional commit type: `feat/` → `feat: `
3. Replace `-` and `_` with spaces in the remainder
4. If no known prefix, just clean up separators

Examples:
- `feat/add-user-auth` → `feat: add user auth`
- `fix/null-pointer` → `fix: null pointer`
- `my-feature-branch` → `my feature branch`

### Base branch detection (`DefaultBase`)

Checks these candidates in order: `main`, `master`, `develop`. Uses the first one that exists
via `git rev-parse --verify`. Defaults to `main` if none found.

### Body generation

PR body is structured as:

```markdown
## Summary

- <commit subject 1>
- <commit subject 2>
...

## Test Plan

- [ ] Manual testing
```

Commits are sourced from `git log <base>..<branch> --pretty=format:%s --no-merges`.

### gh CLI integration

- `git.HasGhCLI()` checks PATH for `gh` binary
- If present: runs `gh pr create --title <title> --body <body> --base <base>`
- If absent: prints the generated title/body for manual use and links to https://cli.github.com

## Changelog Generation

`mine git changelog` produces Markdown grouped by conventional commit type.

### Grouping logic (`parseConventionalType`)

Extracts the type from `<type>[(<scope>)][!]: <subject>`:

1. Find first occurrence of `:`, `(`, or `!`
2. Extract text before that character
3. Strip any scope suffix (`feat(api)` → `feat`)
4. Lowercase and match against known types

Known types mapped to sections:
- `feat` → Features
- `fix` → Bug Fixes
- `docs` → Documentation
- `refactor` → Refactoring
- `chore` → Chores
- Everything else → Other

### Section ordering

Features → Bug Fixes → Documentation → Refactoring → Chores → Other

Empty sections are omitted from output.

## WIP Round-Trip

`mine git wip` / `mine git unwip` form a safe save/restore pair:

- `wip`: `git add -A && git commit -m "wip"`
- `unwip`: checks last commit message == "wip" (case-insensitive), then `git reset --soft HEAD~1`

The check prevents accidentally undoing a real commit whose message happens to be short.

## Testability

All external git calls go through the package-level `runGit` function variable.
Tests override this to return canned output without needing an actual git repository:

```go
origRunGit := runGit
defer func() { runGit = origRunGit }()
runGit = func(args ...string) (string, error) { ... }
```

Similarly, `SwitchBranch`, `DeleteBranch`, `PruneRemote`, `UndoLastCommit`, `WipCommit`,
and `InstallAlias` are all exported function variables to allow injection in integration tests.

## Shell Functions

Five git helper functions are injected via `mine shell init`:

| Name | Bash/Zsh | Fish | Description |
|------|----------|------|-------------|
| `gc` | `git commit -m "$*"` | `git commit -m "$argv"` | Commit shorthand |
| `gca` | `git commit --amend -m "$*"` | `git commit --amend -m "$argv"` | Amend shorthand |
| `gp` | `git push --set-upstream origin $(branch)` | same | Push with tracking |
| `gpl` | `git pull --rebase` | same | Pull with rebase |
| `gsw` | `git switch "$1"` | `git switch $argv[1]` | Switch shorthand |

All functions support `--help` for inline documentation.
