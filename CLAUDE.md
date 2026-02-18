# mine — Operating Manual

> This is the agentic knowledge base for the `mine` CLI tool. It captures architecture
> decisions, patterns, lessons learned, and development practices. It is the source of
> truth for how to work on this project. Not user-facing documentation.

## Project Overview

**mine** is a developer CLI supercharger — a single Go binary that replaces the sprawl of
productivity tools with one fast, delightful, opinionated companion.

- **Language**: Go 1.25+
- **CLI**: Cobra
- **TUI**: Lipgloss + Bubbletea (reusable fuzzy-search picker in `internal/tui/`)
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
│   ├── tui/         # Reusable TUI components (fuzzy-search picker)
│   ├── tmux/        # Tmux session management and layout persistence
│   ├── env/         # Encrypted per-project environment profiles
│   ├── version/     # Build-time version info
│   └── ...          # New domains go here
├── docs/            # Agentic/internal docs only (vision, decisions, status, specs, plans)
│   ├── internal/    # Agentic docs (vision, decisions, status — NOT user-facing)
│   ├── plans/       # Implementation plans and design docs
│   └── examples/    # Example plugin manifests and configs
├── scripts/         # Install, release, CI helpers
│   └── autodev/     # Autonomous dev workflow scripts
├── site/            # Documentation site (Astro Starlight, Vercel, auto-deploys)
│   ├── src/
│   │   ├── content/docs/  # User-facing documentation (markdown)
│   │   ├── styles/        # Custom CSS (gold/amber theme)
│   │   └── assets/        # Logos, images
│   ├── astro.config.mjs   # Astro + Starlight config (sidebar, plugins)
│   ├── package.json       # npm dependencies
│   └── vercel.json        # Vercel deployment config
└── .github/         # CI/CD workflows, CODEOWNERS, PR template
```

Rules:
- `cmd/` files are thin — parse args, call domain logic, format output
- `internal/` packages own their domain — they don't import each other unnecessarily
- Keep files under 500 lines
- Tests live next to the code they test (`_test.go` suffix)
- Core OSS docs (README, CONTRIBUTING, CHANGELOG, LICENSE) live at repo root
- User-facing docs live in `site/src/content/docs/` (Starlight markdown)
  - `features/` — high-level feature overview pages (what it does, quick example, how it works)
  - `commands/` — full command reference pages (all flags, subcommands, error table)
- Agentic/internal docs live in `docs/internal/`, `docs/plans/`

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
9. **TUI picker pattern**: Interactive commands use `internal/tui.Run()` — a reusable
   Bubbletea-based fuzzy-search picker. Callers implement `tui.Item` (FilterValue, Title,
   Description) and pass items to `tui.Run()` with options. Falls back to plain list output
   when `tui.IsTTY()` returns false. New interactive features (AI sessions, SSH, port
   forwarding) should reuse this abstraction.
10. **External binary integration**: Features wrapping external tools (tmux, git, etc.)
    shell out via `exec.Command` with structured output parsing. Attach/switch commands
    that replace the process use an injectable `execSyscall` var for testability.
11. **User-local hooks**: Scripts in `~/.config/mine/hooks/` are auto-discovered by
    filename convention (`<command-pattern>.<stage>.<ext>`) and registered into the
    hook pipeline at startup. Filenames are parsed right-to-left (extension, then stage,
    remainder is command pattern). Scripts must be executable (+x). Transform hooks
    chain alphabetically; notify hooks run in parallel. CLI: `mine hook list/create/test`.
12. **Env profile pattern**: Per-project encrypted environment profiles via `internal/env`.
    Profile files are age-encrypted JSON stored at `$XDG_DATA_HOME/mine/envs/<sha256(project_path)>/<profile>.age`.
    Active profile per project is tracked in the `env_projects` SQLite table (defaults to `local`).
    Passphrase sourced from `MINE_ENV_PASSPHRASE`, `MINE_VAULT_PASSPHRASE`, or TTY prompt — never stored.
    CLI: `mine env show/set/unset/diff/switch/export/template/inject`. Shell helper: `menv`.

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
- **Conventional commits**: PR squash merges use `type: description` format
  (e.g. `feat: add phased review pipeline`, `fix: prevent duplicate reviews`).
  Valid types: `feat`, `fix`, `chore`, `docs`, `refactor`, `test`, `ci`.
- PRs require CI passing (`test` job). Copilot provides automated review.
- Human merges PRs after reviewing.
- CODEOWNERS: `@rnwolfe` reviews everything
- Site: https://mine.rwolfe.io (Vercel)
- Repo: https://github.com/rnwolfe/mine

## Autonomous Development Workflow

An event-driven GitHub Actions pipeline that autonomously implements issues end-to-end.

### How it works

Three workflows form a loop with a phased review pipeline:

1. **`autodev-dispatch`** — Runs on a 4-hour cron (or manual trigger). Picks the oldest
   `agent-ready` issue, labels it `in-progress`, and triggers the implement workflow.
2. **`autodev-implement`** — Checks out `main`, creates a branch, runs the agent (Claude
   via `claude-code-action@v1`) to implement the issue, pushes, and opens a PR.
   The PR triggers CI and Copilot review.
3. **`autodev-review-fix`** — Phased review pipeline that routes fixes based on review phase:
   - **Copilot phase**: Iterates on Copilot feedback up to 3 times
   - **Claude phase**: Triggered after Copilot is satisfied (adds `claude-review-requested` label)
   - **Done**: Agent addresses Claude's feedback, creates follow-up issues for unresolved items
4. **`claude-code-review`** — Only triggers when explicitly requested: via `claude-review-requested`
   label (autodev pipeline) or `@claude` mention in a PR comment (manual request).

### Review pipeline flow

```
implement → push → CI + Copilot review
                        ↓
              autodev-review-fix (copilot phase)
              ├─ Has comments & iter < 3 → agent fixes → push → loop
              ├─ Has comments & iter >= 3 → transition to claude
              └─ No comments → transition to claude
                        ↓
              claude-code-review.yml (triggered by label)
                        ↓
              autodev-review-fix (claude phase)
              └─ Agent fixes + creates follow-up issues
                        ↓
              Phase → done, completion comment posted
                        ↓
              Human merges
```

### Phase state tracking

PR body contains an HTML comment tracking pipeline state:
```
<!-- autodev-state: {"phase": "copilot", "copilot_iterations": 0} -->
```
Phases: `copilot` → `claude` → `done`

### Labels

| Label | Meaning |
|-------|---------|
| `agent-ready` | Issue is ready for autonomous implementation |
| `in-progress` | Issue is currently being worked on |
| `autodev` | PR was created by the autonomous workflow |
| `needs-human` | Autodev hit a limit and needs human intervention |
| `claude-review-requested` | Copilot phase done, ready for Claude review |

### Secrets required

| Secret | Purpose |
|--------|---------|
| `CLAUDE_CODE_OAUTH_TOKEN` | OAuth token for Claude Code agent execution |
| `AUTODEV_TOKEN` | Fine-grained PAT with Contents, Pull requests, Issues (read/write). Used for push/PR operations so events trigger downstream workflows (GITHUB_TOKEN events don't trigger other workflows). |

### Circuit breakers

- **Max concurrency**: Only 1 `autodev` PR open at a time (prevents merge conflicts)
- **Copilot iterations**: Up to 3 fix cycles on Copilot feedback before transitioning to Claude
- **Claude fix**: 1 final fix cycle after Claude review
- **Timeouts**: 60 min for implementation, 45 min for review fixes
- **Max turns**: 100 for implementation, 50 for review fixes (high to allow complex work, prevents infinite loops)
- **Protected files**: Agent cannot modify CLAUDE.md, workflows, or autodev scripts
- **Trusted users**: Only users in `AUTODEV_TRUSTED_USERS` (config.sh) can trigger autodev via `agent-ready` label
- **Scheduled review poll**: Every 4 hours fallback catches reviews from bot actors gated by GitHub's contributor approval

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
- `phase:1`, `phase:2`, `phase:3` — roadmap phase
- `good-first-issue` — approachable for new contributors
- `spec` — has a spec document in `docs/internal/specs/`

Workflow:
1. Features start as GitHub Issues with clear acceptance criteria
2. Complex features get a spec doc in `docs/internal/specs/`
3. When ready to implement, create a branch from the issue
4. Issues reference the spec; PRs reference the issue

## Backlog Curation Skills

Five Claude Code skills form a backlog quality and personality pipeline. All are
manual-invoke only (`disable-model-invocation: true`) since they create/update GitHub
issues or modify user-facing strings.

| Skill | Purpose | Example |
|-------|---------|---------|
| `/brainstorm` | Generate feature ideas through codebase exploration | `/brainstorm todo`, `/brainstorm plugins` |
| `/sweep-issues` | Audit open issues against quality checklist and label gaps | `/sweep-issues`, `/sweep-issues feature` |
| `/refine-issue` | Iteratively improve an existing issue via Q&A (auto-picks `needs-refinement`) | `/refine-issue 35`, `/refine-issue` |
| `/draft-issue` | Turn a rough idea into a structured issue | `/draft-issue recurring todos` |
| `/personality-audit` | Audit CLI output, docs, and site for tone consistency | `/personality-audit cli`, `/personality-audit docs` |

The pipeline flow: `/sweep-issues` labels issues needing work with `needs-refinement` →
`/refine-issue` (no args) auto-picks from that queue → `/personality-audit` ensures
user-facing strings stay consistent with the project voice. `/brainstorm` and `/draft-issue`
feed new issues into the backlog that `/sweep-issues` later evaluates.

All skills target the gold-standard issue template (based on issue #35) defined in
`.claude/skills/shared/issue-quality-checklist.md`. The template includes: summary,
subcommands table, architecture notes, integration points, acceptance criteria, and
documentation requirements.

Key files:
- `.claude/skills/brainstorm/SKILL.md` — feature ideation skill
- `.claude/skills/sweep-issues/SKILL.md` — backlog quality audit skill
- `.claude/skills/refine-issue/SKILL.md` — issue refinement skill (with auto-pick)
- `.claude/skills/draft-issue/SKILL.md` — issue drafting skill
- `.claude/skills/personality-audit/SKILL.md` — tone and voice audit skill
- `.claude/skills/shared/issue-quality-checklist.md` — shared quality template

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

### L-015: Copilot review state is COMMENTED, not changes_requested
GitHub Copilot's pull request reviewer posts reviews with state `COMMENTED`, not
`changes_requested`. A workflow filtering on `review.state == 'changes_requested'`
will never trigger on Copilot reviews. The fix is to check for the reviewer identity
(`copilot-pull-request-reviewer[bot]`) and inspect whether the review has any actionable
comments in either its body or its inline comments, rather than relying solely on the review state.

### L-016: Bot actors trigger GitHub Actions approval gates
When a bot (e.g. `Copilot`) posts a `pull_request_review`, GitHub treats it as a
first-time contributor and requires manual approval before the triggered workflow runs
(conclusion: `action_required`, zero jobs execute). This isn't configurable via API for
public repos. Fix: add a `schedule` trigger as a fallback — a cron that polls for
unprocessed reviews on autodev PRs regardless of who posted them.

### L-017: Label-based triggers need trust verification
The `agent-ready` label triggers the entire autonomous pipeline. Without verification,
anyone who can label an issue could queue arbitrary code generation. Fix: use the issue
timeline API to check who applied the label and only proceed if they're in the trusted
users allowlist (`AUTODEV_TRUSTED_USERS` in config.sh).

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
| `internal/hook/registry.go` | Thread-safe hook registry with glob pattern matching |
| `internal/hook/exec.go` | ExecHandler — runs external hook scripts |
| `cmd/hook.go` | Hook CLI commands (list, create, test) |
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
| `internal/tui/picker.go` | Reusable fuzzy-search picker (Bubbletea model, Run helper) |
| `internal/tui/fuzzy.go` | Fuzzy matching algorithm (subsequence with scoring) |
| `internal/tmux/tmux.go` | Tmux session management (list, new, attach, kill) |
| `internal/tmux/layout.go` | Layout persistence (save/load/list, TOML in XDG config) |
| `cmd/tmux.go` | Tmux CLI commands with TUI picker integration |
| `cmd/env.go` | Env CLI commands (show, set, unset, diff, switch, export, template, inject) |
| `internal/env/env.go` | Env manager: profile CRUD, age encryption/decryption, active profile tracking, diff, export |
| `scripts/autodev/config.sh` | Autodev shared constants, logging, utilities |
| `scripts/autodev/pick-issue.sh` | Issue selection with concurrency guard |
| `scripts/autodev/parse-reviews.sh` | Extract review feedback for agent consumption |
| `scripts/autodev/check-gates.sh` | Quality gate verification (CI, iterations, mergeable) |
| `scripts/autodev/open-pr.sh` | PR creation with auto-merge and iteration tracking |
| `scripts/autodev/agent-exec.sh` | Model-agnostic agent execution abstraction |
| `site/astro.config.mjs` | Astro + Starlight config (sidebar, social links, plugins) |
| `site/src/content/docs/index.mdx` | Landing page (hero, features, quick start) |
| `site/src/content/docs/getting-started/` | Installation and quick start guides |
| `site/src/content/docs/features/` | Feature overview pages (high-level, links to command reference) |
| `site/src/content/docs/commands/` | Command reference pages (full flags, subcommands, error tables) |
| `site/src/content/docs/contributors/` | Architecture and plugin protocol docs |
| `site/src/styles/custom.css` | Gold/amber brand theming |
| `site/vercel.json` | Vercel deployment config (Astro preset, rewrites) |
