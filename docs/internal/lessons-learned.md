# Lessons Learned

> Numbering convention: `L-NNN` — sequential, never reused. When adding a new entry,
> increment from the highest existing number.

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
now enforces: notify stage <-> notify mode, everything else <-> transform mode.

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

### L-018: Interactive picker output must handle command substitution
Shell helpers like `p` use command substitution to capture `--print-path`, which pipes
stdout. Bubbletea picker rendering must target a real TTY output stream (stderr) in
that mode, or the picker can become invisible/hang despite stdin being interactive.

### L-019: Platform keychain integration via CLI tools avoids CGo
For OS keychain integration (macOS Keychain, GNOME Keyring), shell out to CLI tools
(`security` on macOS, `secret-tool` on Linux) rather than importing native bindings.
This keeps the binary CGo-free and dependency-free. Use build tags for platform files
(`//go:build darwin`, `//go:build linux`, `//go:build !darwin && !linux`) and put
shared types (interfaces, no-op fallback) in an untagged file. For injectable testing,
expose the store as a package-level `var` in the cmd package so tests can swap it out
without modifying production code.

### L-020: Bot review events cause duplicate review-fix runs
When a bot (Copilot, Claude) posts a PR review, it fires a `pull_request_review` event.
`autodev-review-fix` has both a `pull_request_review` trigger AND a `workflow_run`/
`workflow_dispatch` trigger. If both fire for the same review cycle, `claude-fix` or
`copilot-fix` runs twice — wasting API calls and potentially causing state conflicts.
Fix: in the route script, skip `pull_request_review` events when the reviewer is a bot
(`[bot]` login suffix or login `claude`). Bot phases have dedicated trigger paths:
copilot phase via `workflow_dispatch` (from implement), claude phase via `workflow_run`
(after Claude Code Review completes). The `pull_request_review` path should only handle
human-submitted reviews.

### L-021: `allowed_bots` must include `claude` for `workflow_run`-triggered steps
`claude-code-action@v1` has an `allowed_bots` parameter that controls which non-human
actors can invoke the action. When `autodev-review-fix` is triggered by `workflow_run`
(i.e., the "Claude Code Review" workflow completed), `github.actor` is `claude` — the
bot identity of the Claude Code action. If `allowed_bots` only lists `github-actions[bot]`,
the action rejects execution with: "Workflow initiated by non-human actor: claude (type: Bot)."
Fix: set `allowed_bots: "github-actions[bot],claude"` on any agent step that can be
reached via a `workflow_run` whose triggering workflow itself ran a Claude agent.

### L-022: Adopt workflow — reverse-import pattern for existing configs
When importing existing agent configs into a canonical store (adopt), the safe pattern
is: (1) scan for adoptable content and detect conflicts BEFORE making any changes,
(2) copy files into the store, (3) replace originals with symlinks via the existing
Link() function with --force. This separation means the Link() function can be reused
without modification. Conflict detection uses byte-equal comparison for shared files
(instruction files, MCP config) since these map multiple-agent-sources to a single
canonical destination. Per-agent files (settings/claude.json) can never conflict.
Always check `isAlreadyManagedByStore` first to skip files that are already symlinks
into the store — prevents double-adoption. Auto-commit to git history after adoption
to provide a recovery point.

### L-024: Batch-merge conflicts from spec-driven sub-issue bursts
When a spec or refinement breaks a large feature into multiple sub-issues and all
are labeled `backlog/ready`, autodev dispatch can queue them in rapid succession.
Each branches from `main`, implements a different slice, and opens a PR. The first
PR to merge updates `main`; subsequent PRs have merge conflicts. Fix: gate dispatch
on open autodev PR count — if any non-blocked `via/*` PR is open, skip dispatch and
retry next cycle. This serializes the pipeline: one PR open at a time, zero conflict
opportunity. See `scripts/autodev/pick-issue.sh`.

### L-025: Auto-merge requires all-green CI as the quality gate
Enabling `gh pr merge --squash --auto` on a PR queues it for automatic squash merge
once all required branch-protection checks pass. This removes the human-as-bottleneck
step (35–110 min window in measured data) while keeping CI as the hard gate. Use
`--delete-branch` to clean up automatically. Falls back gracefully if auto-merge is
disabled on the repo or the token lacks merge permissions.

### L-023: Implementing a dependent issue without its dependencies merged
When issue dependencies haven't been merged yet, implement the minimum foundation
needed by the current issue alongside the primary feature. Split into logical files
by concern (agents.go, git.go, sync.go) to stay under the 500-line limit and make
each concern independently testable. Use a bare local git repo in integration tests
to mock a real remote — `git init --bare` creates a functional remote without network
access. For sync pull redistribution, use `SyncPullWithResult()` as the primary entry
point (returns a summary), with `SyncPull()` delegating to it — keeps callers simple
while allowing detailed output in the cmd layer.

### L-026: Project-level vs global agent config disambiguation

When implementing project-level scaffolding alongside global agent management, use a
separate spec registry (`buildProjectSpecRegistry`) rather than extending the global
`buildLinkRegistry`. Project-level agent dirs differ from global ones (e.g., codex uses
`.agents/` at project level, but `~/.codex/` globally). This separation avoids coupling
between global and project link behavior.

When the agents store is initialized, the manifest is authoritative for detecting which
agents are active — fall back to live detection ONLY when the store is not yet initialized.
If the manifest says 0 detected agents, return 0 (don't silently fall back to a live scan
that could produce unexpected scaffolding).

For `project init` + `project link` workflows: init creates the directory structure
(including empty skill dirs), and link replaces those dirs with symlinks to the canonical
store. Since init already created the dirs, link requires `--force` to proceed. Document
this in the command help text so users understand the two-step workflow.

### L-027: Copilot fix loop — missing transition to Claude after committed changes
After a successful `copilot-fix` run where Claude commits changes, the autodev pipeline
got stuck: the "Transition to Claude phase" step only fired when `has_changes == 'false'`
(agent ran but made no changes). With `has_changes == 'true'` (fix committed), the step
was skipped, leaving the PR in `copilot` phase indefinitely. The `pull_request_review`
trigger from Copilot's second review is filtered out (bot filter, L-020), and the 4-hour
fallback sees "latest commit is newer than review" and skips. Fix: also trigger the
Claude transition when copilot-fix succeeds with committed changes — one Copilot pass +
one Claude fix is sufficient; Copilot re-review is not required when all comments are addressed.

### L-028: Agent config store pattern

The canonical agents store (`~/.local/share/mine/agents/`) uses a git-backed directory
as its persistence layer — not SQLite. This is intentional: the store's primary value
is portability (push to a git remote, pull on another machine). The `.mine-agents`
manifest JSON tracks detected agents and link mappings. Key design decisions:

- **`buildRegistry`** and **`buildLinkRegistry`** are separate: detection (binary + dir)
  is independent from linking (which files go where). Keep them decoupled.
- **Detection signals**: an agent is "detected" if either its binary is in PATH OR its
  config directory exists — either signal is sufficient.
- **Idempotent init**: `Init()` is always safe to re-run. It skips existing dirs and
  files, creates git repo only once, creates manifest only once.
- **Manifest as contract**: All link operations consistently read and write the manifest
  via `ReadManifest`/`WriteManifest`. Never cache manifest state across operations.

### L-029: Link distribution engine — safety-first symlink management

The link engine (`internal/agents/link.go`) manages symlinks from the canonical store
to each agent's expected config location. Key safety invariants:

- **Never overwrite regular files** without explicit `--force`. Existing regular files
  must go through `adopt` first (imports to store) before linking.
- **Already-correct symlinks are silently updated** (manifest entry upserted, no filesystem
  change). This makes repeated `mine agents link` calls idempotent.
- **Symlinks pointing elsewhere** require `--force` to overwrite — may be an existing
  installation managed by a different tool.
- **Empty source dirs are skipped**: skills/, commands/ are only linked if non-empty.
  This prevents creating confusing empty symlinks.
- The `upsertManifestLink` helper ensures the manifest reflects the last-written state —
  existing entries for the same (source, target, agent) triple are replaced, not duplicated.


### L-029: schedule concurrency group races with workflow_run — use idempotency guard
The `schedule` trigger uses `'scheduled'` as its concurrency group (PR number is unknown
at trigger time). This means it runs concurrently with `workflow_run` and `workflow_dispatch`
runs for the same PR (those use `autodev-review-fix-<PR_NUMBER>`). When both reach the
`Finalize — mark phase done` step simultaneously, they both post completion comments and
try to enable auto-merge. Fix: add an idempotency guard at the start of the finalize step
— check if `human/review-merge` label is already present; if so, skip. This handles the
race without requiring a per-PR concurrency group for schedule (which would require knowing
the PR number before the job starts).

### L-029: CHANGES_REQUESTED review blocks merge even after comments addressed
`claude-code-action` submits PR reviews with state `CHANGES_REQUESTED`. When the
`claude-fix` agent subsequently commits fixes, it posts new `COMMENTED` reviews but does
NOT dismiss the original `CHANGES_REQUESTED` review. GitHub treats an unresolved
`CHANGES_REQUESTED` review as a hard merge blocker even after all inline threads are
resolved. Fix: add a "Dismiss CHANGES_REQUESTED reviews" step after the claude-fix commit
that queries all reviews from `claude[bot]` with state `CHANGES_REQUESTED` and dismisses
them via `PUT /pulls/{pr}/reviews/{review_id}/dismissals`. Requires a token with
`pull-requests: write` permission (`AUTODEV_TOKEN`).
