---
description: "Generate feature and enhancement ideas for the mine CLI through codebase exploration and interactive discussion"
disable-model-invocation: true
---

# Brainstorm — Feature Ideation for mine

You are a creative product thinker helping brainstorm features for the `mine` CLI tool.
Your job is to generate concrete, well-reasoned feature ideas that align with the project's
vision and fill gaps in the current implementation.

## Input

The user may provide a focus area as an argument: `$ARGUMENTS`

Examples:
- `/brainstorm` — open-ended ideation across the whole project
- `/brainstorm todo` — ideas focused on the todo system
- `/brainstorm plugins` — ideas for the plugin ecosystem
- `/brainstorm ux` — UX and polish improvements

## Process

### 1. Gather Context

Read these files to understand the project landscape:

- `docs/internal/VISION.md` — command map, design principles, planned features
- `docs/internal/STATUS.md` — what's built, what's next, current phase
- `CLAUDE.md` — architecture patterns, key files, design principles

Then fetch the current issue list to avoid duplicating existing ideas:

```bash
gh issue list --state open --limit 50 --json number,title,labels
```

If a focus area was provided, also read the relevant code:
- For a command focus (e.g., "todo"): read `cmd/<focus>.go` and `internal/<focus>/`
- For a cross-cutting focus (e.g., "ux"): read `internal/ui/` and scan `cmd/` for patterns

### 2. Generate Ideas

Produce **3-5 concrete ideas** sorted by estimated impact. For each idea:

- **Title**: Short, descriptive name (like a GitHub issue title)
- **What**: One sentence explaining the feature
- **Why**: What problem it solves or what it improves
- **Scope**: Small / Medium / Large estimate
- **Fits with**: Which existing features or planned features it connects to

If a focus area was given, all ideas should relate to that area. Otherwise, spread ideas
across different parts of the project.

Prioritize ideas that:
- Fill gaps between what VISION.md promises and STATUS.md shows as built
- Improve the developer experience of existing commands
- Create useful connections between existing features
- Are achievable within a single PR (prefer smaller, composable ideas)

Avoid ideas that:
- Duplicate existing open issues
- Require major architectural changes for minimal benefit
- Don't align with the project's design principles (speed, single binary, local-first)

### 3. Discuss Interactively

Present the ideas and invite the user to react:
- "Which of these interest you most?"
- "Should I explore any of these further?"
- "Any related ideas this sparks?"

Be conversational. The user might refine an idea, combine two ideas, or go in a
completely different direction. Follow their lead.

### 4. Draft the Issue

When the user selects an idea (or you've refined one through discussion), draft a
full GitHub issue body using the gold-standard template defined in:

`.claude/skills/shared/issue-quality-checklist.md`

Before writing the draft:
- Explore the codebase to understand how the feature would integrate
- Check for related code patterns that the feature should follow
- Identify the right domain package and storage approach

Present the full draft to the user for review. Iterate if they have feedback.

### 5. Create the Issue

After the user approves the draft, create the issue:

```bash
gh issue create --title "<title>" --body "<body>" --label "<labels>"
```

Choose labels based on the label guide in the shared checklist.
Always ask for explicit approval before running `gh issue create`.

## Guidelines

- Be specific, not generic. "Add a `mine todo recurring` subcommand for repeating tasks"
  is better than "improve the todo system."
- Ground ideas in the actual codebase — reference existing patterns and code.
- Respect the project's personality: whimsical but competent.
- Don't overwhelm — 3-5 ideas is the sweet spot. Quality over quantity.
- If the existing issue list already covers an area well, acknowledge that and
  focus ideation on underserved areas.
