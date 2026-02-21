---
name: autodev
description: "Pick an issue from the backlog, implement it in a fresh worktree, and open a PR"
disable-model-invocation: true
---

# Autodev — Autonomous Implementation from the CLI

You are autonomously implementing a GitHub issue for the `mine` CLI tool, end-to-end:
pick an issue, create a fresh worktree, implement it, verify it, and open a PR.

## Input

The user may provide an issue number as an argument: `$ARGUMENTS`

- `/autodev` — auto-pick the highest-value `backlog/ready` issue
- `/autodev 42` — implement a specific issue number
- `/autodev 42 "custom branch suffix"` — optional branch name override

---

## Step 1 — Read Project Context

Before touching any issues, read `CLAUDE.md` for architecture patterns, design principles,
key files, and the GitHub Issue Workflow section. This is your operating manual.

---

## Step 2 — Select an Issue

### If an issue number was provided

```bash
gh issue view $ISSUE_NUMBER --repo rnwolfe/mine --json number,title,body,labels,state
```

Validate:
- Issue is open
- Issue has the `backlog/ready` label (warn but proceed if missing — the user is overriding)
- Issue is not already `agent/implementing` (abort if it is and tell the user why)

### If no issue number was provided

**Check concurrency guard first:**

```bash
gh pr list --repo rnwolfe/mine --label "via/autodev" --state open --json number,title
```

If there's already 1 or more open `via/autodev` PRs, **stop and report** what's in flight. Do
not start new work when there are open autodev PRs. Ask the user if they want to override.

**Fetch candidates:**

```bash
gh issue list --repo rnwolfe/mine \
  --label "backlog/ready" \
  --state open \
  --json number,title,body,labels \
  --limit 30
```

Filter out any issues that already have the `agent/implementing` label.

If no `backlog/ready` issues exist, broaden the search:

```bash
gh issue list --repo rnwolfe/mine \
  --state open \
  --label "feature" \
  --json number,title,body,labels \
  --limit 20
```

**Evaluate and pick** the highest-value issue. Consider:
- Concrete acceptance criteria (easier to verify = lower risk)
- Self-contained scope (touches one domain, not multiple systems)
- User-visible impact (commands users actually use daily)
- Dependencies: avoid issues blocked by other open issues
- Avoid issues with `human/blocked`, `blocked`, or `wip` labels

Present your selection with a brief rationale (1-2 sentences) before proceeding.
Give the user a moment to redirect if they disagree — but do not wait for approval
unless this is an interactive session. If non-interactive, proceed.

---

## Step 3 — Mark In-Progress and Prepare Branch

```bash
ISSUE_NUMBER=<picked number>
ISSUE_TITLE=$(gh issue view $ISSUE_NUMBER --repo rnwolfe/mine --json title --jq .title)

# Slugify the title (lowercase, hyphens, max 50 chars)
SLUG=$(echo "$ISSUE_TITLE" \
  | tr '[:upper:]' '[:lower:]' \
  | sed -E 's/[^a-z0-9]+/-/g' \
  | sed -E 's/^-+|-+$//g' \
  | cut -c1-50)

BRANCH="autodev/issue-${ISSUE_NUMBER}-${SLUG}"
```

Mark the issue in-progress:

```bash
gh issue edit $ISSUE_NUMBER --repo rnwolfe/mine --add-label "agent/implementing"
```

---

## Step 4 — Create a Fresh Worktree

Ensure main is up to date:

```bash
git fetch origin main
```

Create a worktree branching from `origin/main` (not local main, which may be stale):

```bash
WORKTREE_PATH=".worktrees/${BRANCH##autodev/}"
git worktree add -b "$BRANCH" "$WORKTREE_PATH" origin/main
```

All implementation work happens in `$WORKTREE_PATH`. Use absolute paths when reading
and writing files. Use `cd <worktree>` prefix for make commands.

---

## Step 5 — Implement the Issue

Read the full issue body:

```bash
gh issue view $ISSUE_NUMBER --repo rnwolfe/mine --json body --jq .body
```

Then implement. Follow all conventions from CLAUDE.md:

- `cmd/` files are thin orchestration — domain logic lives in `internal/`
- Keep files under 500 lines
- Match existing code patterns (look at similar commands before writing new ones)
- Write tests (`_test.go` next to each changed file)
- Use `internal/ui` helpers for all output — never raw `fmt.Println`
- Do NOT modify: `CLAUDE.md`, `.github/workflows/`, `scripts/autodev/`

**Read before writing.** Before creating or editing any file, read the existing file
first to understand the current state. Explore related files to match patterns.

For example, before implementing a new `mine tmux` subcommand, read:
- `cmd/tmux.go` — existing command structure
- `internal/tmux/tmux.go` — domain logic patterns
- A similar recently-added subcommand for style reference

---

## Step 5.5 — Pre-Commit Quality Review

Before running `make test`, review your implementation against these checks:

**Reference implementation**: If the issue cites a reference file (e.g. "follow the
pattern in `cmd/config.go:174`"), verify you read it and matched its patterns. Check:
editor invocation, error message style, stdin/stdout/stderr wiring, temp file cleanup.

**Testing quality**:
- Integration tests exercise the actual `runXxx` handler, not just internal helpers
- External tools (editors, shells) are mocked with fake scripts in `t.TempDir()`
- Error paths are tested end-to-end (handler → error), not just at the helper level
- If the issue says "integration test for X", the test must call the handler function

**Error messages**:
- Command suggestions use `ui.Accent.Render()` (see `cmd/config.go:180`)
- Invalid input errors include the expected format/pattern
- Errors are wrapped with operation context (`"saving profile: %w"`)

**Documentation completeness**:
- CLAUDE.md key files table updated if new files were added
- CLAUDE.md architecture pattern updated if CLI surface changed
- Site docs updated with new commands, flags, and error table entries

---

## Step 6 — Verify

From the worktree directory, run:

```bash
cd $WORKTREE_PATH && make test
cd $WORKTREE_PATH && make build
```

If tests fail: fix them. Do not open a PR with failing tests.
If build fails: fix it. Do not open a PR that doesn't compile.

If you cannot make tests pass after a reasonable attempt (2-3 iterations), add the
`human/blocked` label and stop:

```bash
gh issue edit $ISSUE_NUMBER --repo rnwolfe/mine \
  --add-label "human/blocked" \
  --remove-label "agent/implementing"
```

---

## Step 7 — Commit

Stage and commit all changes. `git add -A` is safe here because the worktree
is an isolated copy containing only intentional changes — no risk of staging
sensitive files or unrelated work.

```bash
cd $WORKTREE_PATH
git add -A
git commit -m "$(cat <<EOF
feat: implement #${ISSUE_NUMBER} — ${ISSUE_TITLE}

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

Use conventional commit types: `feat`, `fix`, `refactor`, `test`, `docs`.
For bugs, use `fix:`. For new features, use `feat:`. Keep the first line under 72 chars.

---

## Step 8 — Write PR Description

Write a detailed PR description to `/tmp/pr-description-${ISSUE_NUMBER}.md`:

```markdown
## Summary

<2-4 sentences: what was implemented and why>

Closes #$ISSUE_NUMBER

## Changes

- **New files**: list each new file and its purpose
- **Modified files**: list each modified file and what changed
- **Architecture**: any design decisions or patterns used

## CLI Surface

<If new commands/flags were added:>
- `mine <command>` — description
- Flags: `--flag` — description

## Test Coverage

- Unit tests for ...
- Edge cases covered: ...

## Acceptance Criteria

<Verify each criterion from the issue:>
- [x] Criterion — how it was met
- [ ] Criterion — why it was not met (note as follow-up issue if significant)
```

---

## Step 9 — Push and Open PR

Push the branch:

```bash
cd $WORKTREE_PATH
git push -u origin "$BRANCH"
```

Create the PR:

```bash
gh pr create \
  --repo rnwolfe/mine \
  --head "$BRANCH" \
  --base main \
  --title "$ISSUE_TITLE" \
  --body "$(cat /tmp/pr-description-${ISSUE_NUMBER}.md)

<!-- autodev-state: {\"phase\": \"copilot\", \"copilot_iterations\": 0} -->" \
  --label "via/autodev"
```

---

## Step 10 — Report

Print a clean summary:

```
✓ Implemented #$ISSUE_NUMBER: $ISSUE_TITLE
  Branch:  $BRANCH
  Worktree: $WORKTREE_PATH
  PR:      <PR URL>

The worktree is at .worktrees/<name>. Run `git worktree list` to see it.
To clean up after merge: git worktree remove .worktrees/<name>
```

---

## Guardrails

- **One PR at a time**: If a `via/autodev` PR is already open, report it and stop.
- **Never force-push main**: Only push to the new feature branch.
- **Never modify protected files**: CLAUDE.md, `.github/workflows/`, `scripts/autodev/`
- **Tests must pass**: Never open a PR with failing `make test` or broken `make build`.
- **Verify acceptance criteria**: Read the issue's acceptance criteria and check each one
  in the PR description — don't just implement and hope.
- **Worktree hygiene**: Always create the worktree inside `.worktrees/`. Never create
  worktrees at the repo root or outside the project directory.
- **Stale branches**: Before creating the branch, check if it already exists remotely:
  ```bash
  git ls-remote --exit-code origin "refs/heads/$BRANCH" 2>/dev/null && \
    git push origin --delete "$BRANCH" || true
  ```

## Error Recovery

| Situation | Action |
|-----------|--------|
| Tests fail after 3 attempts | Add `human/blocked` label, clean up worktree, report to user |
| Issue has no acceptance criteria | Note it in PR, implement based on title/description |
| No `backlog/ready` issues exist | Broaden to open `feature`/`enhancement` issues, pick highest value |
| Worktree already exists | Remove it first: `git worktree remove --force <path>`, then recreate |
| Push fails (auth) | Report the error — never force-push or bypass auth |
