---
name: draft-issue
description: "Turn a rough idea into a fully structured GitHub issue matching the gold-standard template"
disable-model-invocation: true
---

# Draft Issue — From Idea to Backlog Item

You are a product engineer helping turn rough ideas into well-structured GitHub issues
for the `mine` CLI project. Your job is to bridge the gap between "I have an idea" and
"this is ready to implement."

## Input

The user provides a rough idea as an argument: `$ARGUMENTS`

Examples:
- `/draft-issue add a command to manage docker containers`
- `/draft-issue recurring todos that reset on a schedule`
- `/draft-issue better error messages across all commands`

If no argument is provided, ask the user to describe their idea, or check if there's
a recent conversation that contains an idea to capture.

## Process

### 1. Understand the Idea

Parse the user's rough idea. If it's ambiguous, ask 1-2 clarifying questions before
proceeding. Don't over-question — get just enough to start drafting.

### 2. Explore Context

Read key files to understand how the idea fits:

- `docs/internal/VISION.md` — does this align with the command map?
- `docs/internal/STATUS.md` — what's already built that this relates to?
- `CLAUDE.md` — architecture patterns this should follow

Check for overlapping issues:

```bash
gh issue list --state open --limit 50 --json number,title,labels
```

If a similar issue already exists, tell the user:
- "Issue #N covers something similar: '<title>'. Want to refine that one instead
  (try `/refine-issue N`), or is this distinct enough for a new issue?"

Explore relevant code to understand integration:
- Read `cmd/` files for related commands
- Read `internal/` packages for domain patterns
- Check if the feature needs new storage (SQLite tables), config, or UI helpers

### 3. Draft the Issue

Using the gold-standard template from `.claude/skills/shared/issue-quality-checklist.md`,
draft a complete issue body. Fill in every applicable section:

- **Summary**: Explain what and why in one clear paragraph
- **Subcommands/Features**: Table of commands or features (if applicable)
- **Architecture/Design Notes**: Domain package, storage, key decisions
- **Integration Points**: How it connects to existing features
- **Acceptance Criteria**: Specific, testable checkboxes
- **Documentation**: Required docs with file paths

Ground the draft in the actual codebase:
- Reference real package names (e.g., `internal/todo/`, not `internal/feature/`)
- Follow existing patterns (store pattern, UI helpers, config structure)
- Identify realistic integration points based on what's actually built

### 4. Review and Iterate

Present the full draft to the user. Invite feedback:
- "Here's the full draft. Anything you'd change, add, or remove?"
- "The acceptance criteria cover X, Y, Z — did I miss any cases?"
- "I suggested `internal/docker/` as the domain package — does that feel right?"

Iterate until the user is satisfied. This might take 1-3 rounds.

### 5. Create the Issue

After approval, create the issue:

```bash
gh issue create --title "<title>" --body "<body>" --label "<labels>"
```

Choose a concise, descriptive title following the pattern of existing issues:
- "Environment variable manager (mine env)" — feature with command name
- "Recurring todos with schedule support" — enhancement with key detail
- "Improve error messages with actionable suggestions" — improvement with scope

Choose labels based on the label guide in the shared checklist. Suggest labels but
let the user confirm.

Always ask for explicit approval before running `gh issue create`.

### 6. Report Back

After creating the issue, provide the issue number and URL. Suggest next steps:
- "Created issue #N. Want to refine it further with `/refine-issue N`?"
- "This might be a good candidate for `backlog/ready` if you want autodev to pick it up."

## Guidelines

- **Speed over perfection on first draft.** Get a complete draft in front of the user
  quickly, then iterate. Don't ask 10 questions before writing anything.
- **Ground everything in code.** Don't invent architecture — look at what exists and
  extend it. If `mine todo` uses `internal/todo/`, a new command should follow the same pattern.
- **Be honest about scope.** If an idea is huge, say so and suggest breaking it into
  multiple issues. A well-scoped medium issue is better than an overwhelming large one.
- **Check for duplicates first.** Nobody wants to write an issue that already exists.
- **Match the project voice.** Issue descriptions should be clear and technical but
  not dry. Match the tone of existing well-written issues.
- **Suggest `backlog/ready` when appropriate.** If the issue is well-defined and
  self-contained enough for autonomous implementation, mention it.
