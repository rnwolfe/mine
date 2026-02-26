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
│   ├── proj/        # Project registry + context switching
│   ├── tui/         # Reusable TUI components (fuzzy-search picker)
│   ├── tmux/        # Tmux session management and layout persistence
│   ├── env/         # Encrypted per-project environment profiles
│   ├── analytics/   # Anonymous usage analytics (ping, dedup, install ID)
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
- When adding or modifying a feature/command, update ALL affected documentation:
  - Feature overview: `site/src/content/docs/features/<feature>.md` — update when capabilities change
  - Command reference: `site/src/content/docs/commands/<command>.md` — update when flags/subcommands change
  - Follow the pattern of existing pages (YAML frontmatter with title/description)
  - The sidebar auto-generates from files — no config changes needed
- Landing page and docs surfaces contain hardcoded feature claims and examples that must
  stay accurate. Check and update these when your changes affect a featured command:
  - `site/src/components/FeatureTabs.tsx` — feature descriptions and command examples
  - `site/src/components/TerminalDemo.tsx` — animated terminal demo scripts
  - `site/src/content/docs/index.mdx` — docs landing page
  - `site/src/content/docs/getting-started/quick-start.md` — onboarding examples

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
    CLI: `mine env show/set/unset/diff/switch/export/template/inject/edit`. Shell helper: `menv`.
13. **Project context registry**: `internal/proj` persists project membership in SQLite
    (`projects` table) and project-local settings in `~/.config/mine/projects.toml`.
    Current/previous project pointers are tracked via `kv` keys (`proj.current`,
    `proj.previous`) to support shell helpers (`p`, `pp`) and dashboard context.
14. **Analytics pattern**: `internal/analytics` sends a synchronous ping with a 2-second
    HTTP timeout after each command (via `PersistentPostRun`). Daily dedup via the `kv`
    SQLite table prevents multiple pings per command per day. The config field uses a
    `*bool` tri-state: `nil` = default-on (first run shows a one-time privacy notice),
    `true` = explicitly enabled, `false` = opted out. Always fails silently — never
    blocks or errors. Payload is PII-free: install ID (random UUIDv4), version, OS,
    arch, command name (no arguments), and date. Install ID persisted at
    `$XDG_DATA_HOME/mine/analytics_id`.

## Testing Standards

1. **Unit vs integration tests**: Unit tests target internal helpers (e.g. `parseEnvFile`,
   `slugify`). Integration tests exercise the actual `runXxx` handler function end-to-end
   with real (or faked) dependencies — they live in `cmd/*_test.go`.
2. **External binary mocking**: When testing commands that invoke external tools (`$EDITOR`,
   `tmux`, `git`, etc.), create fake scripts in `t.TempDir()` and put that dir on `$PATH`.
   Reference pattern: `cmd/tmux_test.go:59` (`setupTmuxEnv`).
3. **Test isolation**: Every test uses `t.TempDir()` + `t.Setenv()` for XDG dirs so tests
   never touch the real filesystem. Reference pattern: `cmd/config_test.go:13` (`configTestEnv`).
4. **Error paths end-to-end**: Don't just test that a helper rejects bad input. Test that
   the handler returns the right error when the editor exits non-zero, when the file is
   malformed, when the profile doesn't exist, etc.
5. **Test naming**: Use `Test<Handler>_<scenario>` (e.g. `TestRunEnvEdit_InvalidProfile`).
   Table-driven tests are preferred when testing multiple input variations.

## Error Message Standards

1. **Accent rendering**: Use `ui.Accent.Render()` for command suggestions, file paths, and
   key names in error messages (matches pattern at `cmd/config.go:180`).
2. **Expected format**: When rejecting invalid input, include the expected format in the
   error (e.g. the regex pattern, the list of valid values, or an example).
3. **Error wrapping**: Wrap errors with operation context:
   `fmt.Errorf("saving profile: %w", err)` — not bare `return err`, unless the caller
   already provides context.
4. **User-facing error structure**: Multi-line errors for user-facing problems: what went
   wrong, then what to do about it (matches Personality Guide and `cmd/config.go:177`).

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
For a comprehensive architecture deep-dive with diagrams, see
[docs/internal/autodev-pipeline.md](docs/internal/autodev-pipeline.md).

### How it works

Four workflows form the core loop, plus a weekly audit:

1. **`autodev-dispatch`** — Runs on a 1-hour cron (or manual trigger). Picks the
   highest-priority `backlog/ready` issue (sorted by priority tier, then age within
   tier), excluding any issue already labeled `human/blocked` or `agent/implementing`.
   Labels it `agent/implementing` and triggers the implement workflow.
2. **`autodev-implement`** — Checks out `main`, creates a branch, runs the agent (Claude
   via `claude-code-action@v1`) to implement the issue, pushes, and opens a PR.
   After creating the PR, the workflow **polls for Copilot review** (up to 10 minutes)
   and then **dispatches `autodev-review-fix`** directly via `workflow_dispatch`. This
   bypasses the `pull_request_review` trigger which gets gated by GitHub's first-time
   contributor approval for bot actors on public repos. If the agent produces no changes
   or fails, both `backlog/ready` and `agent/implementing` are removed and `human/blocked`
   is added — the issue is permanently dequeued until a human re-labels it. The agent
   prompt includes a **blocker protocol**: if the agent cannot implement the issue, it
   writes a structured report to `/tmp/agent-blocker.md` explaining why. The workflow
   posts this report as a comment on the issue.
3. **`autodev-review-fix`** — Phased review pipeline that routes fixes based on review phase:
   - **Copilot phase**: Iterates on Copilot feedback up to 3 times
   - **Claude phase**: Triggered after Copilot is satisfied (adds `agent/review-claude` label)
   - **Done**: Agent addresses Claude's feedback, creates follow-up issues for unresolved items
4. **`claude-code-review`** — Only triggers when explicitly requested: via `agent/review-claude`
   label (autodev pipeline) or `@claude` mention in a PR comment (manual request).
5. **`autodev-audit`** — Weekly (Monday 9 AM UTC) or manual. Runs a Claude agent to
   analyze recent autodev PRs, compute health metrics, spot-check code quality, and
   file a structured report as a GitHub issue labeled `report/pipeline-audit`.

### Review pipeline flow

```
implement → push → create PR → wait for Copilot review (poll ≤10 min)
                                        ↓
              dispatch autodev-review-fix (copilot phase)
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

The implement → review-fix chain is the primary trigger path. The `pull_request_review`
trigger is gated by GitHub's first-time contributor approval for bot actors (Copilot)
on public repos, so the direct `workflow_dispatch` bypasses that bottleneck. A 4-hour
scheduled poll remains as a safety-net fallback.

### Phase state tracking

PR body contains an HTML comment tracking pipeline state:
```
<!-- autodev-state: {"phase": "copilot", "copilot_iterations": 0} -->
```
Phases: `copilot` → `claude` → `done`

### Labels

**Pipeline stage labels** (mutually exclusive per issue/PR):

| Label | Meaning |
|-------|---------|
| `backlog/triage` | New issue, needs evaluation |
| `backlog/needs-spec` | Passed evaluation, needs specification |
| `backlog/needs-refinement` | Has spec, needs refinement before implementation |
| `backlog/ready` | Issue is ready for autonomous implementation |
| `agent/implementing` | Issue is currently being implemented by an agent |
| `agent/review-copilot` | Agent is addressing Copilot review feedback |
| `agent/review-claude` | Agent is addressing Claude review feedback |
| `human/blocked` | Agent hit a limit and needs human intervention |
| `agent/auto-merge` | All reviews done, auto-merge enabled — merges when CI passes |
| `human/review-merge` | All reviews done, auto-merge unavailable — needs human merge |

**Origin labels** (persistent, one per PR):

| Label | Meaning |
|-------|---------|
| `via/autodev` | PR created by `/autodev` CLI skill |
| `via/actions` | PR created by GitHub Actions pipeline |
| `via/maestro` | PR created by Maestro (experimental) |

**Priority labels** (dispatch ordering):

| Label | Meaning |
|-------|---------|
| `priority/critical` | Autodev picks first; highest urgency |
| `priority/high` | Autodev prefers over normal; important |
| (no label) | Normal priority; FIFO within tier |

**Report labels**:

| Label | Meaning |
|-------|---------|
| `report/pipeline-audit` | Weekly pipeline health report issue |
| `regression/autodev` | Bug introduced by an autodev-generated PR — query with `gh issue list --label regression/autodev` to surface in audits |

### Secrets required

| Secret | Purpose |
|--------|---------|
| `CLAUDE_CODE_OAUTH_TOKEN` | OAuth token for Claude Code agent execution |
| `AUTODEV_TOKEN` | Fine-grained PAT with Contents, Pull requests, Issues (read/write). Used for push/PR operations so events trigger downstream workflows (GITHUB_TOKEN events don't trigger other workflows). |

### Circuit breakers

- **Max concurrency**: GitHub Actions concurrency groups serialize implementations; multiple PRs can be reviewed in parallel
- **Copilot iterations**: Up to 3 fix cycles on Copilot feedback before transitioning to Claude
- **Claude fix**: 1 final fix cycle after Claude review
- **Timeouts**: 60 min for implementation, 45 min for review fixes
- **Max turns**: 150 for implementation, 50 for review fixes (high to allow complex work, prevents infinite loops)
- **Protected files**: Agent cannot modify CLAUDE.md, workflows, or autodev scripts
- **Trusted users**: Only users in `AUTODEV_TRUSTED_USERS` (config.sh) can trigger autodev via `backlog/ready` label
- **Implement → review-fix chain**: After PR creation, implement polls for Copilot review (≤10 min) and dispatches review-fix directly. Scheduled 4-hour poll is a safety-net fallback.
- **Weekly audit**: Monday 9 AM UTC pipeline health report filed as GitHub issue

### Model-agnostic design

Each workflow has a clearly delimited `AGENT EXECUTION` block. To swap providers:
1. Replace the `claude-code-action@v1` step with the target provider's action
2. Update `scripts/autodev/agent-exec.sh` for local testing
3. Set `AUTODEV_PROVIDER` env var

### Triggering autonomous development

1. Create a GitHub issue with clear acceptance criteria
2. Add the `backlog/ready` label
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
- Use `/release` skill to run the full release flow interactively (see below)

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

## Product Development Lifecycle

The full pipeline — from roadmap to shipped feature — is documented in:

`docs/internal/LIFECYCLE.md`

**8 phases**: Roadmap → Feature Definition → Backlog Quality → Implementation → Review → Merge → Release → Feedback → (repeat)

**3 implementation paths** (Phase 4): Maestro Auto Run (`maestro/Backlog-Loop/`) · `/autodev` skill · GitHub Actions pipeline

**Audit layer** (cross-cutting, cadence-driven): `/sweep-issues` · `/refine-issue` · `/autodev-audit` · `/personality-audit`

Entry point when you don't know where to start: `/product`

---

## Autonomous Implementation Skill

`/autodev` is the CLI counterpart to the GitHub Actions autodev pipeline. It runs the
full implementation loop locally: pick an issue, create a worktree, implement, verify,
and open a PR — all without leaving the terminal.

| Skill | Purpose | Example |
|-------|---------|---------|
| `/autodev` | Pick highest-value `backlog/ready` issue and implement it end-to-end | `/autodev`, `/autodev 42` |

Key behaviors:
- Auto-picks from `backlog/ready` issues; evaluates by value/impact if multiple exist
- Creates a fresh git worktree at `.worktrees/<branch>` off `origin/main`
- Runs `make test` + `make build` before opening a PR — never ships broken code
- Applies the same concurrency guard as the GH Actions pipeline (max 1 open autodev PR)
- Follows the full GitHub Issue Workflow: closes the issue, verifies acceptance criteria

Key file: `.claude/skills/autodev/SKILL.md`

## Release Skill

`/release` is the release manager. It bridges the gap between "features merged to main"
and "users get a binary" — a step that was previously undocumented and manual.

| Skill | Purpose | Example |
|-------|---------|---------|
| `/release` | Full release flow: analyze → version → CHANGELOG → tag → push | `/release`, `/release v0.3.0` |
| `/release check` | Dry-run: show what's unreleased, proposed version, CHANGELOG preview | `/release check` |
| `/release notes` | Draft CHANGELOG entry only, no commits or tags | `/release notes` |

Key behaviors:
- Fetches merged PRs since last tag and categorizes by conventional commit type
- Proposes semver bump (patch/minor/major) with explicit reasoning
- Drafts a user-facing CHANGELOG entry (not just a list of PR titles)
- Runs a pre-release checklist: no blocked PRs, CHANGELOG accuracy, STATUS.md freshness
- Always shows full summary and waits for explicit confirmation before tagging
- Pushes the tag, which triggers GoReleaser → GitHub Release automatically
- Suggests `/product sync` after release to update STATUS.md

Key file: `.claude/skills/release/SKILL.md`

## Strategic Product Skill

`/product` is the roadmap owner and vision guardian. It does not generate feature
ideas — it maintains the strategic coherence of the product over time. Before any
feature gets into the backlog, `/product` asks: does this make `mine` more completely
what it's trying to be?

| Skill | Purpose | Example |
|-------|---------|---------|
| `/product` | Full roadmap health check: phase gaps, vision drift, priorities | `/product` |
| `/product spec` | Draft spec for highest-value unspecced roadmap feature | `/product spec` |
| `/product spec "idea"` | Evaluate a specific idea for fit; draft spec if it passes | `/product spec "focus + todos"` |
| `/product sync` | Update VISION.md and STATUS.md to reflect current reality | `/product sync` |
| `/product eval N` | Score an open issue on vision, phase, and principle fit | `/product eval 42` |

Key behaviors:
- Reads VISION.md, STATUS.md, DECISIONS.md, all open issues, and existing specs before
  any output — never forms opinions without full context
- Applies a four-part vision filter to every idea: identity test, principle test, phase
  test, replacement test — fails any idea that doesn't clear all four
- Says no explicitly and with reasoning when an idea doesn't fit the vision
- Creates spec documents in `docs/internal/specs/` before GitHub issues
- Can update living docs (VISION.md, STATUS.md) and commit the changes

Key file: `.claude/skills/product/SKILL.md`

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
| `/autodev-audit` | Audit autodev pipeline health, PR quality, and improvement opportunities | `/autodev-audit`, `/autodev-audit pipeline`, `/autodev-audit code` |

The pipeline flow is documented in full in `docs/internal/LIFECYCLE.md`.
Short version: `/product` (strategy) → `/product spec` (spec) → `/draft-issue` / issue
creation (backlog entry) → `/sweep-issues` + `/refine-issue` (quality) → `/autodev`
(implementation) → `/product sync` (living docs) → repeat.

All skills target the gold-standard issue template (based on issue #35) defined in
`.claude/skills/shared/issue-quality-checklist.md`. The template includes: summary,
subcommands table, architecture notes, integration points, acceptance criteria, and
documentation requirements.

Key files:
- `docs/internal/LIFECYCLE.md` — full pipeline: how all skills connect
- `.claude/skills/product/SKILL.md` — strategic roadmap ownership skill
- `.claude/skills/brainstorm/SKILL.md` — feature ideation skill
- `.claude/skills/sweep-issues/SKILL.md` — backlog quality audit skill
- `.claude/skills/refine-issue/SKILL.md` — issue refinement skill (with auto-pick)
- `.claude/skills/draft-issue/SKILL.md` — issue drafting skill
- `.claude/skills/personality-audit/SKILL.md` — tone and voice audit skill
- `.claude/skills/autodev-audit/SKILL.md` — pipeline health and code quality audit skill
- `.claude/skills/shared/issue-quality-checklist.md` — shared quality template

## Lessons Learned

See [docs/internal/lessons-learned.md](docs/internal/lessons-learned.md).
New entries should follow the `L-NNN` numbering convention.

## Key Files

See [docs/internal/key-files.md](docs/internal/key-files.md).
