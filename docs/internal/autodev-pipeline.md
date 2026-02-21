# Autonomous Development Pipeline — Deep Dive

The autodev pipeline is an event-driven GitHub Actions system that autonomously implements
GitHub issues end-to-end. A human creates an issue with acceptance criteria and labels it
`agent-ready`; the pipeline picks it up, creates a branch, implements the feature, opens a
PR, iterates on reviewer feedback (Copilot then Claude), and stops at the final gate: human
merge. A weekly audit monitors pipeline health and files a report.

## Architecture Overview

Five workflows, six shell scripts, and one Claude Code skill form the system.

```mermaid
flowchart TB
    subgraph Triggers
        cron4h["Cron (every 4h)"]
        manual["Manual dispatch"]
        cronMon["Cron (Monday 9 AM)"]
    end

    subgraph Dispatch["autodev-dispatch.yml"]
        pick["pick-issue.sh<br/>Select oldest agent-ready issue"]
    end

    subgraph Implement["autodev-implement.yml"]
        agent1["Claude Agent<br/>(Sonnet 4.6, 100 turns)"]
        openpr["open-pr.sh<br/>Create PR + state tracker"]
    end

    subgraph ReviewFix["autodev-review-fix.yml"]
        route["Route by phase"]
        copilotFix["Copilot fix path<br/>(up to 3 iterations)"]
        claudeFix["Claude fix path<br/>(1 final pass)"]
        done["Phase: done"]
    end

    subgraph CodeReview["claude-code-review.yml"]
        review["Claude Code Review plugin"]
    end

    subgraph Audit["autodev-audit.yml"]
        auditAgent["Claude Agent<br/>(Sonnet 4.6, 30 turns)"]
        auditIssue["Create GitHub issue<br/>with health report"]
    end

    cron4h --> pick
    manual --> pick
    pick -->|"issue_number"| agent1
    agent1 --> openpr
    openpr -->|"PR triggers Copilot review"| route
    route -->|"copilot phase + feedback"| copilotFix
    route -->|"no feedback / 3 iterations"| review
    copilotFix -->|"push triggers new review"| route
    review -->|"workflow_run completed"| claudeFix
    claudeFix --> done
    done -->|"Human merges"| merged["Merged"]

    cronMon --> auditAgent
    auditAgent --> auditIssue
```

## Pipeline Lifecycle

A complete journey from issue creation to merged PR:

```mermaid
sequenceDiagram
    participant H as Human
    participant D as Dispatch
    participant I as Implement
    participant G as GitHub / CI
    participant C as Copilot
    participant RF as Review-Fix
    participant CR as Claude Review

    H->>G: Create issue + label agent-ready
    D->>G: pick-issue.sh (cron or manual)
    G-->>D: Issue #N selected
    D->>I: Trigger with issue_number

    I->>I: Checkout, create branch
    I->>I: Claude agent implements feature
    I->>G: git push + open-pr.sh
    Note over G: PR created with<br/>autodev-state: copilot, iter=0

    G->>C: Copilot auto-reviews PR
    C->>G: Posts review comments
    G->>RF: pull_request_review event

    loop Copilot Phase (up to 3x)
        RF->>RF: parse-reviews.sh
        RF->>RF: Claude agent fixes feedback
        RF->>G: git push (iteration N)
        RF->>G: Update copilot_iterations
        G->>C: Copilot re-reviews
        C->>G: Posts new comments
        G->>RF: pull_request_review event
    end

    RF->>G: Transition to claude phase
    RF->>G: Add label claude-review-requested
    G->>CR: Label triggers review workflow
    CR->>G: Posts review comments

    G->>RF: workflow_run completed
    RF->>RF: parse-reviews.sh
    RF->>RF: Claude agent final fix
    RF->>G: git push (final pass)
    RF->>G: Phase → done

    H->>G: Reviews + merges PR
```

## Workflow Reference

### 1. autodev-dispatch

**File:** `.github/workflows/autodev-dispatch.yml`

| Property | Value |
|----------|-------|
| Triggers | Cron `0 */4 * * *` (every 4h) + manual dispatch |
| Timeout | Default |
| Concurrency | `autodev-dispatch` (serialized) |
| Permissions | contents:read, issues:write, actions:write |

**What it does:** Runs `pick-issue.sh` to find the oldest `agent-ready` issue from a
trusted user, labels it `in-progress`, and dispatches `autodev-implement` with the issue
number. Exits cleanly if no issues are ready.

### 2. autodev-implement

**File:** `.github/workflows/autodev-implement.yml`

| Property | Value |
|----------|-------|
| Triggers | Workflow dispatch (from autodev-dispatch) |
| Input | `issue_number` (required) |
| Timeout | 60 minutes |
| Concurrency | `autodev-implement` (serialized globally) |
| Agent model | Claude Sonnet 4.6, 100 max turns |

**What it does:**
1. Reads the issue title and body via `gh issue view`
2. Creates branch: `autodev/issue-{N}-{slug}` (deletes stale remote if exists)
3. Runs Claude agent with implementation prompt (includes CLAUDE.md rules, doc instructions,
   PR description and title requirements)
4. Reverts any changes to protected files (CLAUDE.md, workflows, autodev scripts)
5. Commits, pushes, and calls `open-pr.sh` to create the PR
6. If no changes produced: comments on issue, adds `needs-human` label

**Agent output files:**
- `/tmp/pr-title.txt` — conventional commit PR title (`type: description`)
- `/tmp/pr-description.md` — full PR body with summary, changes, criteria

### 3. autodev-review-fix

**File:** `.github/workflows/autodev-review-fix.yml`

| Property | Value |
|----------|-------|
| Triggers | `pull_request_review`, `workflow_run` (Claude review), cron `30 */4 * * *` |
| Timeout | 45 minutes |
| Concurrency | Per-PR group (parallel review of different PRs) |
| Agent model | Claude Sonnet 4.6, 50 max turns |

**What it does:** Routes based on the phase stored in the PR body HTML comment.

```mermaid
flowchart TB
    start["Review event received"]
    start --> isAutodev{"Has autodev label?"}
    isAutodev -->|No| skip["Skip"]
    isAutodev -->|Yes| readPhase["Read phase from PR body"]

    readPhase --> phase{"Phase?"}
    phase -->|done| skip
    phase -->|copilot| copilotCheck{"Copilot review?<br/>Has feedback?<br/>Iterations < 3?"}
    copilotCheck -->|"Yes to all"| copilotFix["Copilot fix path"]
    copilotCheck -->|"No feedback or >= 3 iters"| triggerClaude["Transition to claude phase<br/>Add claude-review-requested label"]

    phase -->|claude| claudeCheck{"Claude review<br/>completed?"}
    claudeCheck -->|Yes| claudeFix["Claude fix path"]

    copilotFix --> reconcile1["git pull --rebase"]
    reconcile1 --> parse1["parse-reviews.sh"]
    parse1 --> agent1["Claude agent fixes feedback"]
    agent1 -->|success| commit1["Commit + push + bump iteration"]
    agent1 -->|failure| error1["Add needs-human label<br/>No changes committed"]

    claudeFix --> reconcile2["git pull --rebase"]
    reconcile2 --> parse2["parse-reviews.sh"]
    parse2 --> agent2["Claude agent final fix"]
    agent2 -->|success| commit2["Commit + push"]
    agent2 -->|failure| error2["Add needs-human label<br/>No changes committed"]
    commit2 --> markDone["Phase → done<br/>Post completion comment"]
```

**Key safety features:**
- Branch reconciliation (`git pull --rebase`) before each agent run
- Post-agent steps gated on `steps.<agent>.outcome == 'success'`
- Agent failure adds `needs-human` label; no partial changes committed
- Protected files reverted after successful agent runs

### 4. claude-code-review

**File:** `.github/workflows/claude-code-review.yml`

| Property | Value |
|----------|-------|
| Triggers | `claude-review-requested` label, `@claude` PR comment |
| Agent | Claude with `code-review` plugin |

**What it does:** Runs the Claude Code Review plugin which posts review comments on the PR.
When completed, the `workflow_run` event triggers `autodev-review-fix` to enter the claude
fix path.

### 5. autodev-audit

**File:** `.github/workflows/autodev-audit.yml`

| Property | Value |
|----------|-------|
| Triggers | Cron `0 9 * * 1` (Monday 9 AM UTC) + manual dispatch |
| Input | `limit` (default: 10 PRs to analyze) |
| Timeout | 30 minutes |
| Agent model | Claude Sonnet 4.6, 30 max turns |

**What it does:** Runs a Claude agent that analyzes recent autodev PRs (metrics, code
quality, review themes, stale state) and writes a report to `/tmp/audit-report.md`. The
workflow then creates a GitHub issue titled "Autodev Pipeline Audit — YYYY-MM-DD" with
label `autodev-audit`. If the agent fails, a fallback issue links to the workflow logs.

## Phase State Machine

Review progress is tracked via an HTML comment in the PR body that survives edits and
is invisible to readers:

```html
<!-- autodev-state: {"phase": "copilot", "copilot_iterations": 0} -->
```

```mermaid
stateDiagram-v2
    [*] --> copilot: PR created by open-pr.sh

    copilot --> copilot: Copilot feedback + iteration < 3\n(increment copilot_iterations)
    copilot --> claude: No feedback OR iterations >= 3\n(add claude-review-requested label)

    claude --> done: Claude fix applied\n(remove label, post comment)

    done --> [*]: Human merges PR

    note right of copilot
        Max 3 iterations.
        Agent fixes each round.
        Counter tracked in HTML comment.
    end note

    note right of claude
        Single fix pass.
        Creates follow-up issues
        for unresolved items.
    end note
```

## Scripts & Helpers

All scripts live in `scripts/autodev/` and source `config.sh` for shared constants.

| Script | Purpose | Called by |
|--------|---------|----------|
| `config.sh` | Shared constants (repo, labels, limits, trusted users) + logging + `autodev_slugify()` | All scripts |
| `pick-issue.sh` | Select next `agent-ready` issue; verify trusted labeler via timeline API; label `in-progress` | `autodev-dispatch` |
| `open-pr.sh` | Read agent-generated title/description; create PR with `autodev` label + state tracker | `autodev-implement` |
| `parse-reviews.sh` | Extract review bodies + inline comments with `[comment_id: N]` tags for agent replies | `autodev-review-fix` |
| `check-gates.sh` | Verify quality gates (CI status, iteration count, no pending reviews, mergeable) | Available for local testing |
| `agent-exec.sh` | Local testing abstraction; routes to configured provider (`AUTODEV_PROVIDER`) | Local dev only |

## Security & Trust Model

### Trust verification

The `agent-ready` label triggers the entire pipeline. Without verification, anyone who can
label an issue could queue arbitrary code generation. `pick-issue.sh` verifies the labeler:

1. Fetches the issue timeline via `gh api repos/{owner}/{repo}/issues/{N}/timeline`
2. Finds the last `labeled` event where `label.name == "agent-ready"`
3. Checks `actor.login` against `AUTODEV_TRUSTED_USERS` in `config.sh`
4. Only proceeds if the labeler is trusted (see [L-017](lessons-learned.md#l-017))

### Protected files

Three revert steps (in `autodev-implement` and both paths of `autodev-review-fix`) prevent
the agent from modifying governance files:

```bash
git diff --name-only | grep -E '(CLAUDE\.md|\.github/workflows/|scripts/autodev/)' | xargs git checkout --
```

Files in `docs/internal/` (like `lessons-learned.md` and `key-files.md`) are intentionally
NOT protected — agents are encouraged to update them.

### Secret separation

| Secret | Purpose | Why not GITHUB_TOKEN? |
|--------|---------|----------------------|
| `GITHUB_TOKEN` | Read operations, issue comments, label edits | Default, sufficient for reads |
| `AUTODEV_TOKEN` (PAT) | Push branches, create PRs, edit PRs that trigger workflows | GITHUB_TOKEN events cannot trigger other workflows ([L-014](lessons-learned.md#l-014)) |
| `CLAUDE_CODE_OAUTH_TOKEN` | Authenticate Claude agent executions | Separate credential for AI provider |

## Configuration

### Labels

| Label | Applied by | Meaning |
|-------|-----------|---------|
| `agent-ready` | Human | Issue is ready for autonomous implementation |
| `in-progress` | `pick-issue.sh` | Issue is being worked on |
| `autodev` | `open-pr.sh` | PR was created by the pipeline |
| `needs-human` | Workflow (on failure) | Pipeline hit a limit or error |
| `claude-review-requested` | `autodev-review-fix` | Copilot phase done, triggers Claude review |
| `autodev-audit` | `autodev-audit` workflow | Weekly health report issue |

### Circuit breakers

| Breaker | Value | Purpose |
|---------|-------|---------|
| Implementation concurrency | Serialized via Actions group | One branch created at a time |
| Review-fix concurrency | Per-PR group | Multiple PRs reviewed in parallel |
| Copilot iterations | Max 3 | Prevents infinite fix loops |
| Claude fix passes | 1 | Final pass, creates follow-up issues for remainder |
| Implementation timeout | 60 min | Prevents runaway agent |
| Review-fix timeout | 45 min | Prevents runaway agent |
| Audit timeout | 30 min | Analysis only, no code changes |
| Implementation max turns | 100 | Prevents infinite agent loops |
| Review-fix max turns | 50 | Tighter limit for focused fixes |
| Audit max turns | 30 | Read-only analysis |
| Weekly audit | Monday 9 AM UTC | Pipeline health feedback loop |
| Scheduled review poll | Every 4h (offset 30m) | Catches bot reviews gated by approval ([L-016](lessons-learned.md#l-016)) |

## Debugging Guide

### Agent failed, no changes committed

**Symptoms:** PR gets `needs-human` label, comment says "No changes were committed."

**Diagnosis:**
1. Click the workflow logs link in the PR comment
2. Check the agent step output for error messages
3. Common causes: compilation error the agent couldn't fix, test flakiness, context limit

**Recovery:** Push a manual fix to the branch, or close the PR and re-label the issue.

### Branch divergence / rebase conflicts

**Symptoms:** "Branch reconciliation failed" error in review-fix.

**Diagnosis:** Another process pushed to the branch between checkout and agent execution.

**Recovery:** Manually rebase the branch: `git pull --rebase origin <branch> && git push --force-with-lease`.

### Stale in-progress issue with no PR

**Symptoms:** Issue stuck with `in-progress` label, no open PR.

**Diagnosis:** Implementation workflow failed before creating the PR (agent produced no
changes, or push failed).

**Recovery:** Remove `in-progress` label. If the issue is still valid, re-add `agent-ready`.

### Copilot reviews not triggering review-fix

**Symptoms:** Copilot posts a review but `autodev-review-fix` never runs.

**Diagnosis:** Bot actors trigger GitHub's first-time contributor approval gate
([L-016](lessons-learned.md#l-016)). The scheduled fallback should catch it within 4 hours.

**Recovery:** Manually approve the workflow run in the Actions tab, or wait for the
scheduled poll at `:30` past the next 4-hour mark.

### Wrong model or high costs

**Symptoms:** Pipeline costs are higher than expected.

**Diagnosis:** Check `claude_args` in each workflow for the `--model` flag.

**Current model:** `claude-sonnet-4-6` across all agent steps (implement: 100 turns,
review-fix: 50 turns, audit: 30 turns).

## Related Lessons Learned

These entries in [lessons-learned.md](lessons-learned.md) document hard-won pipeline knowledge:

| Entry | Topic |
|-------|-------|
| [L-013](lessons-learned.md) | Iteration tracking via HTML comments |
| [L-014](lessons-learned.md) | GITHUB_TOKEN cannot trigger downstream workflows |
| [L-015](lessons-learned.md) | Copilot review state is COMMENTED, not changes_requested |
| [L-016](lessons-learned.md) | Bot actors trigger GitHub Actions approval gates |
| [L-017](lessons-learned.md) | Label-based triggers need trust verification |

## Related Files

- [CLAUDE.md — Autonomous Development Workflow](../../CLAUDE.md#autonomous-development-workflow) — reference-level summary
- [lessons-learned.md](lessons-learned.md) — operational lessons (L-013 through L-017)
- [key-files.md](key-files.md) — file purpose index
- [autodev-audit skill](../../.claude/skills/autodev-audit/SKILL.md) — manual audit invocation
