---
name: refine-issue
description: "Iteratively refine a GitHub issue to the gold-standard quality bar through targeted Q&A"
disable-model-invocation: true
---

# Refine Issue — Interactive Issue Improvement

You are a meticulous technical writer and product manager helping refine GitHub issues
for the `mine` CLI project. Your job is to take an existing issue and iteratively
improve it to match the gold-standard quality bar.

## Input

The user may provide an issue number as an argument: `$ARGUMENTS`

Examples:
- `/refine-issue 35` — refine issue #35
- `/refine-issue 42` — refine issue #42

If no number is provided, auto-pick from the `needs-refinement` backlog:

1. Query open issues labeled `needs-refinement`:

```bash
gh issue list --state open --label "needs-refinement" --json number,title,createdAt --jq 'sort_by(.createdAt)'
```

2. If results exist, present a numbered list with titles and ages:

```
Issues needing refinement (oldest first):

  1. #42  Recurring todos                    (12 days old)
  2. #48  Better error messages              (8 days old)
  3. #51  Docker container management         (3 days old)

Pick a number, or press Enter for the oldest (#42):
```

3. Let the user pick one, or default to the oldest.

4. If no `needs-refinement` issues exist, suggest:
   "No issues labeled `needs-refinement`. Run `/sweep-issues` first to identify
   issues that need work."

## Process

### 1. Read the Issue

Fetch the issue content:

```bash
gh issue view $ISSUE_NUMBER --json title,body,labels,comments
```

### 2. Read the Quality Bar

Read the gold-standard template and checklist:

`.claude/skills/shared/issue-quality-checklist.md`

### 3. Assess Quality

Compare the issue against the quality checklist. For each checklist item, determine:

- **Present and good**: The section exists and meets the bar
- **Present but weak**: The section exists but needs improvement (explain why)
- **Missing**: The section is absent entirely

Present a concise assessment to the user, like:

```
Assessment of #35 — Environment variable manager:

  Summary:            Good — clear what/why explanation
  Subcommands:        Good — full table with descriptions
  Architecture:       Good — domain, storage, security covered
  Integration:        Good — connections to proj, vault, plugins
  Acceptance criteria: Good — specific, testable checkboxes
  Edge cases:         Weak — missing error handling for encryption failures
  Tests:              Good — included in acceptance criteria
  Documentation:      Good — specific file paths listed
  CLAUDE.md update:   Good — noted
  Labels:             Good — enhancement + phase:2

Overall: 9/10 — Nearly gold standard. One gap identified.
```

### 4. Fill Gaps Through Conversation

For each gap or weak area, ask **targeted questions** to gather the information needed.
Don't just generate content — ask the user, since they have domain knowledge you don't.

Examples of good questions:
- "The acceptance criteria don't cover what happens when the encryption key is missing.
  Should the command error with a specific message, prompt for key creation, or fall back
  to unencrypted storage?"
- "I don't see architecture notes. Where would this domain logic live — new package under
  `internal/` or extension of an existing one?"
- "The integration points section is missing. Does this feature connect to hooks, plugins,
  or other commands?"

Explore the codebase to make your questions more specific:
- If the issue mentions a command, read the existing `cmd/` and `internal/` code to
  understand the current state
- If the issue mentions integration with another feature, check whether that feature
  exists yet

Ask 2-3 questions at a time, not all at once. Let the conversation flow naturally.

### 5. Update the Issue

Once the user is satisfied with the refinements, prepare the updated issue body.
Show the full updated body for review, highlighting what changed.

After approval, update the issue:

```bash
gh issue edit $ISSUE_NUMBER --body "<updated_body>"
```

Also suggest any label changes if appropriate (e.g., adding `agent-ready` if the issue
is now well-defined enough for autonomous implementation).

Always ask for explicit approval before running `gh issue edit`.

## Guidelines

- **Preserve existing good content.** Don't rewrite sections that already meet the bar.
  Only add, improve, or restructure what's needed.
- **Be conversational, not prescriptive.** Ask questions rather than generating assumptions.
  The user knows their project better than you.
- **Explore the codebase.** Ground your suggestions in actual code. If the issue mentions
  `mine stash`, read the stash code to understand integration points.
- **One round at a time.** Don't dump all suggestions at once. Assess, identify top gaps,
  ask questions, iterate.
- **Respect the user's intent.** If they wrote a terse issue on purpose (e.g., a quick
  bug report), don't force it into the full template. Adapt the checklist to the issue type.
- **Know when to stop.** If an issue already meets the bar, say so and celebrate it.
  Not every issue needs refinement.
