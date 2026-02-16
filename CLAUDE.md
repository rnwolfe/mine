# mine — Operating Manual

> This is the agentic knowledge base for the `mine` CLI tool. It captures architecture
> decisions, patterns, lessons learned, and development practices. It is the source of
> truth for how to work on this project. Not user-facing documentation.

## Project Overview

**mine** is a developer CLI supercharger — a single Go binary that replaces the sprawl of
productivity tools with one fast, delightful, opinionated companion.

- **Language**: Go 1.25+
- **CLI**: Cobra
- **TUI**: Lipgloss (+ Bubbletea for interactive views, coming)
- **Storage**: SQLite via modernc.org/sqlite (pure Go, no CGo)
- **Config**: TOML at `~/.config/mine/config.toml` (XDG-compliant)
- **Binary**: Single static binary, ~7.6 MB stripped

## Build & Test

```bash
make build        # Build to bin/mine
make test         # Run all tests (with race detector)
make cover        # Coverage report
make lint         # go vet ./...
make dev ARGS="todo"  # Quick dev cycle
make install      # Install to PATH
```

- ALWAYS run `make test` after code changes
- ALWAYS run `make build` before committing
- NEVER commit if tests fail

## File Organization

```
mine/
├── CONTRIBUTING.md  # How to contribute (root-level, standard OSS)
├── CHANGELOG.md     # Release history (root-level, standard OSS)
├── LICENSE          # MIT license
├── README.md        # Project overview and quick start
├── CLAUDE.md        # THIS FILE — agentic knowledge base
├── cmd/             # Cobra command definitions (thin orchestration layer)
├── internal/        # Core domain logic (NOT exported)
│   ├── config/      # XDG config management
│   ├── store/       # SQLite data layer
│   ├── ui/          # Theme, styles, print helpers
│   ├── todo/        # Todo domain
│   ├── hook/        # Hook pipeline (stages, registry, exec)
│   ├── plugin/      # Plugin system (manifest, lifecycle, runtime, search)
│   ├── craft/       # Scaffolding recipe engine (data-driven, embed.FS)
│   ├── version/     # Build-time version info
│   └── ...          # New domains go here
├── docs/            # User-facing documentation (install, commands, architecture)
│   └── internal/    # Agentic docs (vision, decisions, status — NOT user-facing)
├── scripts/         # Install, release, CI helpers
│   └── autodev/     # Autonomous dev workflow scripts
├── site/            # Landing page (Vercel, auto-deploys)
└── .github/         # CI/CD workflows, CODEOWNERS, PR template
```

Rules:
- `cmd/` files are thin — parse args, call domain logic, format output
- `internal/` packages own their domain — they don't import each other unnecessarily
- Keep files under 500 lines
- Tests live next to the code they test (`_test.go` suffix)
- Core OSS docs (README, CONTRIBUTING, CHANGELOG, LICENSE) live at repo root
- User-facing docs live in `docs/`
- Agentic/internal docs live in `docs/internal/`

## Architecture Patterns

1. **Domain separation**: Each feature is a package under `internal/`
2. **Store pattern**: SQLite via `store.DB` wrapper — domains get `*sql.DB` via `db.Conn()`
3. **UI consistency**: All output through `internal/ui` helpers — never raw `fmt.Println`
4. **Config**: Single TOML file, loaded once, XDG-compliant paths
5. **Progressive migration**: Schema changes via `store.migrate()` auto-applied on open
6. **Plugin pipeline**: Commands traverse four hook stages: prevalidate → preexec → postexec → notify.
   Hooks are either `transform` (modify context, sequential) or `notify` (fire-and-forget, async).
   Pipeline is zero-cost when no hooks are registered.
7. **Plugin protocol**: Plugins are standalone binaries invoked via JSON-over-stdin. Three invocation
   types: `hook`, `command`, `lifecycle`. Plugins declare capabilities in `mine-plugin.toml`.
   Permissions are sandboxed (env vars, filesystem, network are opt-in).
8. **Recipe engine**: Scaffolding templates are data-driven via `embed.FS`. Built-in recipes
   are Go structs with embedded template files. Users can add recipes by dropping template
   directories into `~/.config/mine/recipes/<category>-<name>/`. Templates use Go
   `text/template` with `{{.Dir}}` as the project directory name.

## Design Principles

1. **Speed**: Every local command < 50ms. No spinners for local ops.
2. **Single binary**: No runtime dependencies. `curl | sh` install.
3. **Opinionated defaults**: Works out of the box. Escape hatches exist.
4. **Whimsical but competent**: Fun personality in messages. Serious about results.
5. **Local first**: Data stays on machine. Cloud features opt-in.
6. **XDG-compliant**: `~/.config/mine/`, `~/.local/share/mine/`, `~/.cache/mine/`

## Personality Guide

- Use emoji sparingly and consistently (see `ui/theme.go` icon constants)
- Greeting should feel like a friend, not a robot
- Tips should be actionable, not generic
- Error messages should say what went wrong AND what to do about it
- Celebrate small wins (completing a todo, finishing a focus session)
- Never be annoying or preachy

## Security Rules

- NEVER hardcode secrets or API keys
- NEVER commit .env files
- Validate all user input at system boundaries
- Sanitize file paths (prevent directory traversal)
- SQLite uses WAL mode with busy timeout (safe for concurrent reads)

## Development Workflow

- **main is sacred.** All changes go through PRs. No direct pushes.
- Branch naming: `feat/`, `fix/`, `chore/`, `docs/` prefixes
- PRs require CI passing (`test` job). Copilot provides automated review.
- Human merges PRs after reviewing.
- CODEOWNERS: `@rnwolfe` reviews everything
- Site: https://mine.rwolfe.io (Vercel)
- Repo: https://github.com/rnwolfe/mine

## Autonomous Development Workflow

An event-driven GitHub Actions pipeline that autonomously implements issues end-to-end.

### How it works

Three workflows form a loop:

1. **`autodev-dispatch`** — Runs on a 4-hour cron (or manual trigger). Picks the oldest
   `agent-ready` issue, labels it `in-progress`, and triggers the implement workflow.
2. **`autodev-implement`** — Checks out `main`, creates a branch, runs the agent (Claude
   via `claude-code-action@v1`) to implement the issue, pushes, and opens a PR.
   The PR triggers CI and `claude-code-review` automatically.
3. **`autodev-review-fix`** — Fires when a review requests changes on an `autodev` PR.
   Extracts review feedback, runs the agent to fix issues, and pushes. This re-triggers
   CI + review, creating a feedback loop until the review passes or the iteration limit hits.
   Human merges manually after all checks pass.

### Labels

| Label | Meaning |
|-------|---------|
| `agent-ready` | Issue is ready for autonomous implementation |
| `in-progress` | Issue is currently being worked on |
| `autodev` | PR was created by the autonomous workflow |
| `needs-human` | Autodev hit a limit and needs human intervention |

### Secrets required

| Secret | Purpose |
|--------|---------|
| `CLAUDE_CODE_OAUTH_TOKEN` | OAuth token for Claude Code agent execution |
| `AUTODEV_TOKEN` | Fine-grained PAT with Contents, Pull requests, Issues (read/write). Used for push/PR operations so events trigger downstream workflows (GITHUB_TOKEN events don't trigger other workflows). |

### Circuit breakers

- **Max concurrency**: Only 1 `autodev` PR open at a time (prevents merge conflicts)
- **Max iterations**: 3 review-fix cycles before adding `needs-human` label
- **Timeouts**: 30 min for implementation, 20 min for review fixes
- **Protected files**: Agent cannot modify CLAUDE.md, workflows, or autodev scripts

### Model-agnostic design

Each workflow has a clearly delimited `AGENT EXECUTION` block. To swap providers:
1. Replace the `claude-code-action@v1` step with the target provider's action
2. Update `scripts/autodev/agent-exec.sh` for local testing
3. Set `AUTODEV_PROVIDER` env var

### Triggering autonomous development

1. Create a GitHub issue with clear acceptance criteria
2. Add the `agent-ready` label
3. Wait for the next cron run, or manually trigger `autodev-dispatch` from the Actions tab
4. Optionally pass a specific issue number via the workflow dispatch input

## GitHub Issue Workflow

When creating a PR that implements a GitHub issue, follow this workflow to ensure acceptance criteria are verified and issues auto-close on merge.

### Before Creating the PR

1. **Read the original issue**
   ```bash
   gh issue view N --json body,title
   ```
   Extract acceptance criteria (look for checkbox lists in the issue body).

2. **Verify each acceptance criterion**
   - Review your code changes against each criterion
   - Confirm each one is satisfied by the implementation
   - Note any criteria that are NOT met (incomplete scope is OK, document it)

### Creating the PR

1. **Document acceptance criteria in PR body**

   Add an "## Acceptance Criteria" section in your PR body:

   ```markdown
   ## Acceptance Criteria

   Verified against issue #N:
   - [x] Criterion 1 — Met by [specific implementation detail]
   - [x] Criterion 2 — Met by [specific implementation detail]
   - [ ] Criterion 3 — Out of scope for this PR, will address in #M
   ```

2. **Update the original issue checkboxes**

   For completed criteria, check them off in the issue itself:
   ```bash
   # Read current issue body
   gh issue view N --json body -q .body > /tmp/issue-body.txt

   # Edit the file to check boxes (change [ ] to [x])
   # Then update the issue
   gh issue edit N --body "$(cat /tmp/issue-body.txt)"
   ```

   This makes completion status visible directly in the issue.

3. **Use closing keywords**

   Include one of these in your PR title or body:
   - `Fixes #N`
   - `Closes #N`
   - `Resolves #N`

   This triggers GitHub's auto-close behavior when the PR merges.

### What If Issue Has No Acceptance Criteria?

If the issue doesn't have clear acceptance criteria:
- Note this in your PR body: "Issue #N has no formal acceptance criteria"
- List what you implemented anyway for reviewer clarity
- Consider adding a comment to the issue suggesting criteria for future similar issues

### Manual Verification Fallback

If a human asks you to verify acceptance criteria for an already-merged PR:

**Command:** Comment tag like `@claude please verify acceptance criteria against PR #N`

**Your response:**
1. Read the merged PR: `gh pr view N --json files,additions,deletions`
2. Read the issue: `gh issue view M --json body`
3. Verify each criterion against the code changes
4. Update the issue checkboxes if not already done
5. Add a comment to the issue summarizing verification results

**Example verification comment:**
```markdown
Verified acceptance criteria against PR #22:

- [x] Criterion 1 — ✅ Met (implemented in commit abc123)
- [x] Criterion 2 — ✅ Met (tests added in commit def456)
- [ ] Criterion 3 — ⚠️ Not addressed in this PR

Overall: 2/3 criteria met. Criterion 3 should be tracked separately.
```

## Release Process

- Tags trigger releases via GoReleaser (GitHub Actions)
- Format: `vMAJOR.MINOR.PATCH` (semver)
- CHANGELOG.md updated before tagging
- Binaries: linux/darwin x amd64/arm64

## Backlog Management

Feature backlog is tracked via GitHub Issues with labels:
- `feature` — new feature requests
- `enhancement` — improvements to existing features
- `phase/1`, `phase/2`, `phase/3` — roadmap phase
- `good-first-issue` — approachable for new contributors
- `spec` — has a spec document in `docs/internal/specs/`

Workflow:
1. Features start as GitHub Issues with clear acceptance criteria
2. Complex features get a spec doc in `docs/internal/specs/`
3. When ready to implement, create a branch from the issue
4. Issues reference the spec; PRs reference the issue

## Lessons Learned

### L-001: Git config name parsing
Git config values may be quoted (`name = "Ryan Wolfe"`). Always strip quotes
when parsing gitconfig values. Fixed in `cmd/init.go:gitUserName()`.

### L-002: TOML encoding of pre-quoted strings
If a value already contains quotes, TOML encoder will double-escape them.
Always clean input before saving to config.

### L-003: Working directory drift
When using `cd` in Bash tool calls (e.g., `cd site && vercel deploy`), the CWD
persists across subsequent calls. Always use absolute paths or explicitly `cd`
back to project root for subsequent commands.

### L-004: Vercel project naming
When deploying from a subdirectory (`site/`), Vercel uses the directory name
as the project name. Deploy from project root or use `--name` flag to control.

### L-005: GitHub Rulesets API schema sensitivity
The rulesets API (`POST /repos/{owner}/{repo}/rulesets`) is very picky about
the `rules[].parameters` shape. The `pull_request` type requires ALL five boolean
params to be present. When in doubt, create the ruleset in UI first, export it,
and use that JSON as the template.

### L-006: Self-approval impossible on GitHub
When pushing PRs via `gh` under your own token, you can't approve your own PRs.
Branch protection requiring approvals blocks the author. Solution: use CI checks
as the gate and Copilot for automated review, human merges manually.

### L-007: Third-party scaffolding cleanup
The claude-flow CLI scaffolded 355 files (.claude/agents/, .claude-flow/, .swarm/,
.mcp.json, hooks in settings.json) as part of initial setup. These were generic
templates unrelated to the Go project. Lesson: audit scaffolding tools before
committing. Remove attribution settings (`settings.json.attribution`) immediately
to prevent unwanted co-author credits in git history.

### L-008: Copilot review catches real issues
Copilot code review found 7 legitimate issues on first PR: bc dependency in CI,
unchecked errors in tests, duplicated test setup, unsafe `rm $(which ...)`,
missing curl safety note, and doc/code mismatch. Treat it as a real reviewer.

### L-009: Plugin stage/mode pairing matters
Notify mode hooks only make sense at the notify stage. A hook declared as
`stage = "preexec"` with `mode = "notify"` will silently never execute because
the pipeline skips notify-mode hooks during transform stages. Manifest validation
now enforces: notify stage ↔ notify mode, everything else ↔ transform mode.

### L-010: Fire-and-forget means fire-and-forget
The notify stage was originally blocking via `wg.Wait()`, defeating the purpose.
Notify hooks should never block command completion — the goroutine is launched
and the command returns immediately. Tests that verify notify execution need
polling/deadline logic instead of synchronous assertions.

### L-011: Stream large plugin binaries
Plugin binaries can be 10-50MB. Reading them entirely into memory via
`os.ReadFile` wastes memory. Use `io.Copy` between file handles to stream
the copy. Same pattern applies anywhere large files are moved on disk.

### L-012: Acceptance criteria must be explicitly verified
Issue #8 was implemented in PR #22, but the acceptance criteria were never verified
and the issue didn't auto-close because the PR didn't use closing keywords. Agents
must read the issue, verify each criterion, update issue checkboxes, and use
`Fixes #N` / `Closes #N` / `Resolves #N` in the PR body. See "GitHub Issue Workflow"
section for the full workflow.

### L-013: Iteration tracking via HTML comments
Autodev tracks review-fix iteration count in PR body HTML comments
(`<!-- autodev-state: {"iteration": N} -->`). This survives PR body edits and is
invisible to readers. `grep -oP` extracts the value. Always bump the counter after
each review-fix cycle, and check it before starting a new one.

### L-014: GITHUB_TOKEN cannot trigger downstream workflows
GitHub's security policy prevents ALL events created by GITHUB_TOKEN from triggering
other workflows (not just pushes — also PR open/close/reopen). The close/reopen
workaround doesn't work because those events also come from GITHUB_TOKEN. Use a
Personal Access Token (PAT) stored as a repo secret (`AUTODEV_TOKEN`) for operations
that need to trigger CI, code review, or other downstream workflows.

## Key Files

| File | Purpose |
|------|---------|
| `cmd/root.go` | Dashboard, command registration |
| `cmd/todo.go` | Todo CRUD commands |
| `cmd/plugin.go` | Plugin CLI commands (install, remove, search, info) |
| `internal/ui/theme.go` | Colors, icons, style constants |
| `internal/store/store.go` | DB connection, migrations |
| `internal/todo/todo.go` | Todo domain logic + queries |
| `internal/config/config.go` | Config load/save, XDG paths |
| `internal/hook/hook.go` | Hook types, Context, Handler interface |
| `internal/hook/pipeline.go` | Hook pipeline (Wrap, stage execution, flag rewrites) |
| `internal/hook/discover.go` | User hook discovery, script creation, testing |
| `internal/hook/exec.go` | ExecHandler — runs external hook scripts |
| `internal/plugin/manifest.go` | Plugin manifest parsing and validation |
| `internal/plugin/lifecycle.go` | Plugin install, remove, list, registry management |
| `internal/plugin/runtime.go` | Plugin invocation (hooks, commands, lifecycle events) |
| `internal/plugin/permissions.go` | Permission sandboxing, env builder, audit log |
| `internal/plugin/search.go` | GitHub search for mine plugins |
| `cmd/stash.go` | Stash CLI commands (track, commit, log, restore, sync) |
| `internal/stash/stash.go` | Stash domain logic — git-backed versioning, manifest, sync |
| `cmd/craft.go` | Craft CLI commands (dev, ci, git, list) |
| `internal/craft/recipe.go` | Recipe engine, registry, template execution |
| `internal/craft/builtins.go` | Built-in recipe definitions (go, node, python, rust, docker, github CI) |
| `scripts/autodev/config.sh` | Autodev shared constants, logging, utilities |
| `scripts/autodev/pick-issue.sh` | Issue selection with concurrency guard |
| `scripts/autodev/parse-reviews.sh` | Extract review feedback for agent consumption |
| `scripts/autodev/check-gates.sh` | Quality gate verification (CI, iterations, mergeable) |
| `scripts/autodev/open-pr.sh` | PR creation with auto-merge and iteration tracking |
| `scripts/autodev/agent-exec.sh` | Model-agnostic agent execution abstraction |
