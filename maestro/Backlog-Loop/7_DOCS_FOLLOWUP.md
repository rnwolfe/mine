# Documentation Updates and Follow-Up Issues

## Context

- **Playbook:** Backlog Loop
- **Agent:** {{AGENT_NAME}}
- **Project:** {{AGENT_PATH}}
- **Auto Run Folder:** {{AUTORUN_FOLDER}}
- **Loop:** {{LOOP_NUMBER}}

## Objective

Check whether the implementation requires updates to the Starlight documentation site, agent/internal docs, or the CLAUDE.md knowledge base. Identify any follow-up issues needed. Execute all identified tasks before proceeding.

## Tasks

- [ ] **Read implementation context**: Read `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_ISSUE.md` for the issue details, worktree path, and PR info. Read `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_PLAN.md` for what was implemented. Get the list of changed files:
  ```
  git -C WORKTREE_PATH diff origin/main...HEAD --name-only
  ```

- [ ] **Check Starlight docs for needed updates**: Review the documentation site at `site/src/content/docs/` (in the worktree) and determine if updates are needed:

  **New or changed commands?** If cmd/ files were added or modified:
  - Check if `site/src/content/docs/docs/commands/` has a corresponding page
  - If a new command was added, create a new doc page following the existing format
  - If an existing command changed (new flags, different behavior), update its doc page

  **New concepts or features?** If the implementation introduces user-facing concepts:
  - Check if getting-started guides need updates
  - Check if the feature needs its own doc page

  **Changed configuration?** If config format changed:
  - Update any configuration reference docs

- [ ] **Check agent docs and knowledge base**: Determine if internal docs need updates:

  **CLAUDE.md updates needed?** Check if the implementation:
  - Adds new key files that should be listed in the Key Files table
  - Introduces new architecture patterns worth documenting
  - Reveals lessons learned that should be captured
  - Changes the file organization or build process

  If CLAUDE.md updates are needed, **do NOT modify CLAUDE.md directly**. Instead, note the needed changes in the follow-up list below.

  **Internal docs?** Check if `docs/internal/` or `docs/plans/` need new or updated documents.

- [ ] **Check for follow-up issues**: Determine if any follow-up GitHub issues should be created:

  - Scope that was intentionally deferred from the current issue
  - Technical debt introduced (acceptable shortcuts, TODOs in code)
  - Related features that would complement this implementation
  - Edge cases identified during review that aren't covered yet
  - CLAUDE.md updates that need to be applied separately

- [ ] **Execute documentation updates**: Make all identified doc changes **in the worktree**:
  - Create/update Starlight doc pages
  - Update internal docs if needed
  - Run the doc site build to verify if doc changes were made:
    ```
    cd WORKTREE_PATH/site && npm run build
    ```
    (Only if npm dependencies are available; skip if not)

- [ ] **Create follow-up issues**: For each identified follow-up, create a GitHub issue:
  ```
  gh issue create --repo rnwolfe/mine \
    --title "follow-up: <description>" \
    --body "<detailed description with context from this implementation>" \
    --label "enhancement"
  ```
  Reference the current issue and PR in each follow-up issue body.

- [ ] **Commit and push doc changes**: If any documentation files were modified:
  ```
  git -C WORKTREE_PATH add -A
  git -C WORKTREE_PATH commit -m "docs: update documentation for #ISSUE_NUMBER"
  git -C WORKTREE_PATH push
  ```
  If no changes were needed, skip this step.

- [ ] **Log follow-up actions**: Append to `{{AUTORUN_FOLDER}}/BACKLOG_LOG_{{DATE}}.md`:
  ```markdown
  ### Loop {{LOOP_NUMBER}} — Documentation & Follow-up
  - **Doc pages updated:** <list or "none">
  - **Follow-up issues created:** <list of issue numbers or "none">
  - **CLAUDE.md changes needed:** <yes/no — if yes, note what>
  ```

## Guidelines

- Only create documentation for user-facing changes — don't document internal refactors
- Follow existing doc page format and structure in the Starlight site
- Keep follow-up issues focused and actionable — one concern per issue
- Don't create follow-up issues for trivial improvements
- If no documentation or follow-up work is needed, that's perfectly fine — just log it and proceed
- All file changes happen in the worktree directory
