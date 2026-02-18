# Progress Check — Loop Control

## Context

- **Playbook:** Backlog Loop
- **Agent:** {{AGENT_NAME}}
- **Project:** {{AGENT_PATH}}
- **Auto Run Folder:** {{AUTORUN_FOLDER}}
- **Loop:** {{LOOP_NUMBER}}

## Objective

Determine whether to continue the loop or exit. This document controls the pipeline:
- If more `agent-ready` issues exist and no blockers, reset documents 1-8 to trigger another loop.
- If blocked or backlog is empty, leave documents 1-8 completed so the pipeline exits.

## Tasks

- [ ] **Check for blockers**: Read `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_ISSUE.md`. If the status contains "BLOCKED" (concurrency limit, no issues, untrusted labeler), do NOT reset any documents — the pipeline should exit. Append the blocker reason to `{{AUTORUN_FOLDER}}/BACKLOG_LOG_{{DATE}}.md` and mark this task complete.

- [ ] **Check backlog**: Run `gh issue list --repo rnwolfe/mine --label agent-ready --state open --json number,labels --jq '[.[] | select(.labels | map(.name) | (index("in-progress") | not) and (index("maestro") | not))] | length'`. If the count is 0, do NOT reset documents — the backlog is empty. Append "Backlog empty — pipeline exiting" to the log and mark complete.

- [ ] **Check concurrency**: Run `gh pr list --repo rnwolfe/mine --label maestro --state open --json number --jq 'length'`. If >= 3, do NOT reset — concurrency limit reached. Append "Concurrency limit (3 maestro PRs open) — pipeline pausing" to the log and mark complete.

- [ ] **Reset for next loop**: If none of the above blockers apply, reset all tasks in documents 1 through 8 by unchecking all checkboxes in:
  - `{{AUTORUN_FOLDER}}/1_PICK_ISSUE.md`
  - `{{AUTORUN_FOLDER}}/2_PLAN.md`
  - `{{AUTORUN_FOLDER}}/3_IMPLEMENT.md`
  - `{{AUTORUN_FOLDER}}/4_OPEN_PR.md`
  - `{{AUTORUN_FOLDER}}/5_COPILOT_REVIEW.md`
  - `{{AUTORUN_FOLDER}}/6_SELF_REVIEW.md`
  - `{{AUTORUN_FOLDER}}/7_DOCS_FOLLOWUP.md`
  - `{{AUTORUN_FOLDER}}/8_FINALIZE.md`

  Change every `- [x]` back to `- [ ]` in those files. This signals Auto Run to process them again in the next loop iteration.

  Append to `{{AUTORUN_FOLDER}}/BACKLOG_LOG_{{DATE}}.md`:
  ```markdown
  ---
  Loop {{LOOP_NUMBER}} complete. More issues available — continuing to next loop.
  ```

## Exit Conditions (do NOT reset)

The pipeline exits when ANY of these are true:
1. Issue file contains "BLOCKED"
2. No `agent-ready` issues remain (excluding those labeled `in-progress` or `maestro`)
3. Concurrency limit reached (3 open maestro PRs)

When exiting, leave documents 1-8 with their tasks checked. Auto Run will see no unchecked tasks and stop.

## Guidelines

- This document has **Reset on Completion** enabled in the playbook config
- Documents 1-8 do NOT have Reset on Completion — they only reset when this document explicitly unchecks their tasks
- Always append to the log, never overwrite it
- The concurrency check uses the `maestro` label, not `autodev` — this playbook's PRs are independent of the GitHub Actions autodev pipeline
