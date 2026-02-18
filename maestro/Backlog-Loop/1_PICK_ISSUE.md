# Pick Next Backlog Issue

## Context

- **Playbook:** Backlog Loop
- **Agent:** {{AGENT_NAME}}
- **Project:** {{AGENT_PATH}}
- **Auto Run Folder:** {{AUTORUN_FOLDER}}
- **Loop:** {{LOOP_NUMBER}}

## Objective

Select the next `agent-ready` issue from the GitHub backlog, verify it was labeled by a trusted user, and prepare a working branch. Output the issue details for downstream documents.

## Tasks

- [x] **Check concurrency**: Run `gh pr list --repo rnwolfe/mine --label autodev --state open --json number --jq 'length'`. If the count is >= 1, write "BLOCKED: concurrency limit reached" to `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_ISSUE.md` and mark this task complete without proceeding further. (Loop 00001: result = 1, blocked)

- [ ] **Find candidate issue**: Run `gh issue list --repo rnwolfe/mine --label agent-ready --state open --json number,title,labels --jq '[.[] | select(.labels | map(.name) | index("in-progress") | not)] | sort_by(.number) | first'`. If no issues found, write "BLOCKED: no agent-ready issues" to `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_ISSUE.md` and mark complete.

- [ ] **Verify trusted labeler**: Use `gh api repos/rnwolfe/mine/issues/ISSUE_NUMBER/timeline --jq '[.[] | select(.event == "labeled" and .label.name == "agent-ready")] | last | .actor.login'` to check who applied the label. Only proceed if the labeler is `rnwolfe`. If untrusted, write "BLOCKED: untrusted labeler" to the issue file and mark complete.

- [ ] **Read issue details**: Run `gh issue view ISSUE_NUMBER --repo rnwolfe/mine --json title,body,labels` and save the full output to `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_ISSUE.md` in this format:

```markdown
# Issue #N: Title

## Status
READY

## Labels
label1, label2

## Body
<full issue body>
```

- [ ] **Mark issue in-progress**: Run `gh issue edit ISSUE_NUMBER --repo rnwolfe/mine --add-label in-progress`.

- [ ] **Create branch**: From the latest `main`, create and checkout a new branch: `git checkout main && git pull origin main && git checkout -b autodev/issue-N-<slugified-title>` where the slug is the title lowercased with non-alphanumeric chars replaced by hyphens, truncated to 50 chars.

## Guidelines

- If the issue file already exists with status `READY`, skip all tasks — the issue was already picked.
- If ANY blocker is hit (concurrency, no issues, untrusted labeler), write the reason to the issue file so `5_PROGRESS.md` can detect it and exit the loop.
