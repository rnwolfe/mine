# Pick Next Backlog Issue

## Context

- **Playbook:** Backlog Loop
- **Agent:** {{AGENT_NAME}}
- **Project:** {{AGENT_PATH}}
- **Auto Run Folder:** {{AUTORUN_FOLDER}}
- **Loop:** {{LOOP_NUMBER}}

## Objective

Select the next `agent-ready` issue from the GitHub backlog, verify it was labeled by a trusted user, create a fresh git worktree for implementation, and output the issue details for downstream documents.

## Tasks

- [ ] **Check concurrency**: Run `gh pr list --repo rnwolfe/mine --label maestro --state open --json number --jq 'length'`. If the count is >= 3, write "BLOCKED: concurrency limit reached (3 maestro PRs open)" to `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_ISSUE.md` and mark this task complete without proceeding further.

- [ ] **Find candidate issue**: Run `gh issue list --repo rnwolfe/mine --label agent-ready --state open --json number,title,labels --jq '[.[] | select(.labels | map(.name) | (index("in-progress") | not) and (index("maestro") | not))] | sort_by(.number) | first'`. This excludes issues already labeled `in-progress` or `maestro` (being worked by another instance). If no issues found, write "BLOCKED: no agent-ready issues available" to `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_ISSUE.md` and mark complete.

- [ ] **Verify trusted labeler**: Use `gh api repos/rnwolfe/mine/issues/ISSUE_NUMBER/timeline --paginate --jq '[.[] | select(.event == "labeled" and .label.name == "agent-ready")] | last | .actor.login // empty'` to check who applied the label. Only proceed if the labeler is `rnwolfe`. If the result is empty or the labeler is untrusted, write "BLOCKED: untrusted labeler" to the issue file and mark complete.

- [ ] **Label issue**: Apply both `maestro` and `in-progress` labels to claim the issue:
  ```
  gh issue edit ISSUE_NUMBER --repo rnwolfe/mine --add-label maestro --add-label in-progress
  ```
  This prevents other parallel instances from picking the same issue.

- [ ] **Read issue details**: Run `gh issue view ISSUE_NUMBER --repo rnwolfe/mine --json title,body,labels` and save the full output to `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_ISSUE.md` in this format:

```markdown
# Issue #N: Title

## Status
READY

## Labels
label1, label2

## Worktree
{{AGENT_PATH}}-worktrees/issue-ISSUE_NUMBER

## Body
<full issue body>
```

- [ ] **Create worktree**: Create a fresh git worktree for this issue so multiple instances can work in parallel without conflicts:
  ```
  mkdir -p {{AGENT_PATH}}-worktrees
  git -C {{AGENT_PATH}} fetch origin main
  git -C {{AGENT_PATH}} worktree add {{AGENT_PATH}}-worktrees/issue-ISSUE_NUMBER -b maestro/issue-ISSUE_NUMBER-<slugified-title> origin/main
  ```
  Where the slug is the title lowercased with non-alphanumeric chars replaced by hyphens, truncated to 50 chars.

## Guidelines

- If the issue file already exists with status `READY`, skip all tasks — the issue was already picked.
- If ANY blocker is hit (concurrency, no issues, untrusted labeler), write the reason to the issue file so `9_PROGRESS.md` can detect it and exit the loop.
- The worktree isolates this implementation from the main checkout and other parallel runs. All subsequent documents operate inside the worktree directory.
- Labels are applied immediately after selection to prevent race conditions with parallel instances.
