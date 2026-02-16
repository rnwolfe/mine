# Progress Check — Loop Control

## Context

- **Playbook:** Backlog Loop
- **Agent:** {{AGENT_NAME}}
- **Project:** {{AGENT_PATH}}
- **Auto Run Folder:** {{AUTORUN_FOLDER}}
- **Loop:** {{LOOP_NUMBER}}

## Objective

Determine whether to continue the loop or exit. This document controls the pipeline:
- If more `agent-ready` issues exist and no blockers, reset documents 1-4 to trigger another loop.
- If blocked or backlog is empty, leave documents 1-4 completed so the pipeline exits.

## Tasks

- [ ] **Check for blockers**: Read `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_ISSUE.md`. If the status contains "BLOCKED" (concurrency limit, no issues, untrusted labeler), do NOT reset any documents — the pipeline should exit. Append the blocker reason to `{{AUTORUN_FOLDER}}/BACKLOG_LOG_{{DATE}}.md` and mark this task complete.

- [ ] **Check backlog**: Run `gh issue list --repo rnwolfe/mine --label agent-ready --state open --json number,labels --jq '[.[] | select(.labels | map(.name) | index("in-progress") | not)] | length'`. If the count is 0, do NOT reset documents — the backlog is empty. Append "Backlog empty — pipeline exiting" to the log and mark complete.

- [ ] **Check concurrency**: Run `gh pr list --repo rnwolfe/mine --label autodev --state open --json number --jq 'length'`. If >= 1, do NOT reset — wait for the current PR to be merged before processing more. Append "Concurrency limit — waiting for PR merge" to the log and mark complete.

- [ ] **Return to main**: Run `git checkout main && git pull origin main` to prepare for the next iteration.

- [ ] **Reset for next loop**: If none of the above blockers apply, reset all tasks in documents 1 through 4 by unchecking all checkboxes in:
  - `{{AUTORUN_FOLDER}}/1_PICK_ISSUE.md`
  - `{{AUTORUN_FOLDER}}/2_PLAN.md`
  - `{{AUTORUN_FOLDER}}/3_IMPLEMENT.md`
  - `{{AUTORUN_FOLDER}}/4_OPEN_PR.md`

  Change every `- [x]` back to `- [ ]` in those files. This signals Auto Run to process them again in the next loop iteration.

  Append to `{{AUTORUN_FOLDER}}/BACKLOG_LOG_{{DATE}}.md`:
  ```markdown
  ---
  Loop {{LOOP_NUMBER}} complete. More issues available — continuing to next loop.
  ```

## Exit Conditions (do NOT reset)

The pipeline exits when ANY of these are true:
1. Issue file contains "BLOCKED"
2. No `agent-ready` issues remain (backlog empty)
3. An autodev PR is already open (concurrency limit)

When exiting, leave documents 1-4 with their tasks checked. Auto Run will see no unchecked tasks and stop.

## Guidelines

- This document has **Reset on Completion** enabled in the playbook config
- Documents 1-4 do NOT have Reset on Completion — they only reset when this document explicitly unchecks their tasks
- Always append to the log, never overwrite it
