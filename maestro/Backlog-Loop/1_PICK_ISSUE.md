# Pick Next Backlog Issue

## Context

- **Playbook:** Backlog Loop
- **Agent:** Mine CLI
- **Project:** /home/rnwolfe/dev/mine
- **Auto Run Folder:** /home/rnwolfe/dev/mine/maestro
- **Loop:** 00002

## Objective

Select the next `agent-ready` issue from the GitHub backlog, verify it was labeled by a trusted user, create a fresh git worktree for implementation, and output the issue details for downstream documents.

## Tasks

- [x] **Check concurrency**: Run `gh pr list --repo rnwolfe/mine --label maestro --state open --json number --jq 'length'`. If the count is >= 3, write "BLOCKED: concurrency limit reached (3 maestro PRs open)" to `/home/rnwolfe/dev/mine/maestro/LOOP_00002_ISSUE.md` and mark this task complete without proceeding further.
  > Result: 2 open maestro PRs — under limit, proceeding.

- [x] **Find candidate issue**: Run `gh issue list --repo rnwolfe/mine --label agent-ready --state open --json number,title,labels --jq '[.[] | select(.labels | map(.name) | (index("in-progress") | not) and (index("maestro") | not))] | sort_by(.number) | first'`. This excludes issues already labeled `in-progress` or `maestro` (being worked by another instance). If no issues found, write "BLOCKED: no agent-ready issues available" to `/home/rnwolfe/dev/mine/maestro/LOOP_00002_ISSUE.md` and mark complete.
  > Result: Found issue #33 "Git workflow supercharger (mine git)"

- [x] **Verify trusted labeler**: Use `gh api repos/rnwolfe/mine/issues/ISSUE_NUMBER/timeline --jq '[.[] | select(.event == "labeled" and .label.name == "agent-ready")] | last | .actor.login'` to check who applied the label. Only proceed if the labeler is `rnwolfe`. If untrusted, write "BLOCKED: untrusted labeler" to the issue file and mark complete.
  > Result: Labeler is `rnwolfe` — trusted, proceeding.

- [ ] **Label issue**: Apply both `maestro` and `in-progress` labels to claim the issue:
  ```
  gh issue edit 28 --repo rnwolfe/mine --add-label maestro --add-label in-progress
  ```
  This prevents other parallel instances from picking the same issue.
  > Result: Labels applied successfully.

- [ ] **Read issue details**: Run `gh issue view 28 --repo rnwolfe/mine --json title,body,labels` and save the full output to `/home/rnwolfe/dev/mine/maestro/LOOP_00002_ISSUE.md` in this format:

```markdown
# Issue #N: Title

## Status
READY

## Labels
label1, label2

## Worktree
/home/rnwolfe/dev/mine-worktrees/issue-N

## Body
<full issue body>
```
  > Result: Saved to `/home/rnwolfe/dev/mine/maestro/LOOP_00002_ISSUE.md`.

- [ ] **Create worktree**: Create a fresh git worktree for this issue so multiple instances can work in parallel without conflicts:
  > Result: Worktree created at `/home/rnwolfe/dev/mine-worktrees/issue-28` on branch `maestro/issue-28-user-local-hooks` tracking `origin/main` (HEAD at f537e82).
  ```
  mkdir -p /home/rnwolfe/dev/mine-worktrees
  git -C /home/rnwolfe/dev/mine fetch origin main
  git -C /home/rnwolfe/dev/mine worktree add /home/rnwolfe/dev/mine-worktrees/issue-28 -b maestro/issue-28-user-local-hooks origin/main
  ```
  Where the slug is the title lowercased with non-alphanumeric chars replaced by hyphens, truncated to 50 chars.

## Guidelines

- If the issue file already exists with status `READY`, skip all tasks — the issue was already picked.
- If ANY blocker is hit (concurrency, no issues, untrusted labeler), write the reason to the issue file so `9_PROGRESS.md` can detect it and exit the loop.
- The worktree isolates this implementation from the main checkout and other parallel runs. All subsequent documents operate inside the worktree directory.
- Labels are applied immediately after selection to prevent race conditions with parallel instances.
