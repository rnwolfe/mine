# Push Changes and Check Progress

## Context

- **Playbook:** Review Fix
- **Agent:** {{AGENT_NAME}}
- **Project:** {{AGENT_PATH}}
- **Auto Run Folder:** {{AUTORUN_FOLDER}}
- **Loop:** {{LOOP_NUMBER}}
- **PR Number:** PR_NUMBER

## Objective

Commit and push the fixes, then check if there's likely to be more feedback. If the feedback file was empty (NO_FEEDBACK), exit the loop.

## Tasks

- [ ] **Check for exit condition**: Read `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_FEEDBACK.md`. If the content is "NO_FEEDBACK", do NOT reset documents 1-2 — the loop should exit. Mark this task complete.

- [ ] **Check for changes**: Run `git diff --stat`. If there are no changes, do NOT reset — nothing to push. Mark complete.

- [ ] **Commit and push**: Stage and commit all changes, then push:
  ```
  git add -A
  git commit -m "fix: address review feedback (iteration {{LOOP_NUMBER}})"
  git push
  ```

- [ ] **Reset for next loop**: Reset all tasks in documents 1 and 2 by unchecking all checkboxes in:
  - `{{AUTORUN_FOLDER}}/1_EXTRACT_FEEDBACK.md`
  - `{{AUTORUN_FOLDER}}/2_FIX_AND_REPLY.md`

  Change every `- [x]` back to `- [ ]` in those files. This allows the next loop to extract fresh feedback after the reviewer re-reviews.

## Exit Conditions (do NOT reset)

- Feedback file contains "NO_FEEDBACK" — no more review comments to address
- No code changes were made — nothing to push
- Loop {{LOOP_NUMBER}} >= 3 — max iterations reached, needs human attention

## Guidelines

- This document has **Reset on Completion** enabled
- Documents 1-2 only reset when this document explicitly unchecks their tasks
- After pushing, the reviewer (Copilot or human) will re-review, and the next loop iteration will pick up any new feedback
