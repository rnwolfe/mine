# Extract Review Feedback

## Context

- **Playbook:** Review Fix
- **Agent:** {{AGENT_NAME}}
- **Project:** {{AGENT_PATH}}
- **Auto Run Folder:** {{AUTORUN_FOLDER}}
- **Loop:** {{LOOP_NUMBER}}
- **PR Number:** PR_NUMBER

## Objective

Extract all actionable review feedback from the PR and write it to a structured file for the fix step.

## Tasks

- [ ] **Fetch top-level reviews**: Run `gh api repos/rnwolfe/mine/pulls/PR_NUMBER/reviews --jq '[.[] | select((.state == "CHANGES_REQUESTED" or .state == "COMMENTED") and (.body != null and .body != "") and (.user.login != "github-actions[bot]"))] | sort_by(.submitted_at) | reverse | .[0:5]'`. Extract reviewer, state, body, and review ID for each.

- [ ] **Fetch inline comments**: Run `gh api repos/rnwolfe/mine/pulls/PR_NUMBER/comments --jq '[.[] | select(.user.login != "github-actions[bot]")] | sort_by(.created_at) | reverse | .[0:20]'`. Extract author, file path, line number, body, and comment ID for each.

- [ ] **Write feedback file**: Save structured feedback to `{{AUTORUN_FOLDER}}/LOOP_{{LOOP_NUMBER}}_FEEDBACK.md`:

```markdown
# Review Feedback — Loop {{LOOP_NUMBER}}

## Review Comments

### reviewer (STATE) [review_id: ID]
<body>

## Inline Comments

### path/to/file.go:42 (author) [comment_id: ID]
<body>
```

If no feedback is found (no reviews and no inline comments), write "NO_FEEDBACK" as the only content. This signals the loop to exit.

## Guidelines

- Include ALL comments, not just the latest — the agent should see full context
- Always include the `comment_id` and `review_id` tags — they're needed for replies
- Filter out `github-actions[bot]` comments (those are our own bot)
