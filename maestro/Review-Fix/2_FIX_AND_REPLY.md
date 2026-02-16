# Fix Issues and Reply to Comments

## Context

- **Playbook:** Review Fix
- **Agent:** {{AGENT_NAME}}
- **Project:** {{AGENT_PATH}}
- **Auto Run Folder:** {{AUTORUN_FOLDER}}
- **Loop:** {{LOOP_NUMBER}}
- **PR Number:** PR_NUMBER

## Objective

Read the review feedback, fix each issue in the code, and reply directly to each inline comment on GitHub explaining what was changed.

## Tasks

- [ ] **Read feedback**: Read `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_FEEDBACK.md`. If the content is "NO_FEEDBACK", mark this task complete without proceeding.

- [ ] **Read project conventions**: Read `CLAUDE.md` for project patterns, architecture rules, and coding standards.

- [ ] **Address each comment**: For each piece of review feedback:
  1. Understand what the reviewer is asking for
  2. Make the code change (or determine why no change is needed)
  3. If you made a change, verify it doesn't break tests (`make test`)
  4. Reply directly to inline comments using:
     ```
     gh api "repos/rnwolfe/mine/pulls/PR_NUMBER/comments/COMMENT_ID/replies" -f body="<explanation of what you changed>"
     ```
  5. If you chose not to change something, reply explaining why

- [ ] **Verify build**: Run `make test && make build` to confirm everything passes.

- [ ] **Check protected files**: Run `git diff --name-only` and verify no changes to CLAUDE.md, `.github/workflows/`, or `scripts/autodev/`. Revert any protected file changes with `git checkout -- <file>`.

## Guidelines

- Do NOT modify CLAUDE.md, any files in `.github/workflows/`, or `scripts/autodev/`
- Address ALL comments — don't skip any
- Keep replies concise but informative
- If a comment can't be fully resolved, explain what was done and what remains
- Run tests after fixing each group of related changes, not just at the end
