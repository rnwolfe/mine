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
