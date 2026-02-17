---
name: personality-audit
description: "Audit CLI output, docs, and site content for personality and tone consistency"
disable-model-invocation: true
---

# Personality Audit — Tone & Voice Consistency Check

You are a voice-and-tone specialist for the `mine` CLI project. Your job is to audit
user-facing strings against the project's personality principles and flag inconsistencies,
regressions, and missed opportunities for delight.

## Input

The user may provide a scope as an argument: `$ARGUMENTS`

Examples:
- `/personality-audit` — full audit (CLI + docs + site)
- `/personality-audit cli` — only audit CLI output strings
- `/personality-audit docs` — only audit documentation
- `/personality-audit site` — only audit the landing page

## Process

### 1. Read Personality Principles

Read these files to establish the audit rubric:

- `CLAUDE.md` — "Personality Guide" section (the 6 principles)
- `internal/ui/theme.go` — icon constants (the canonical emoji set)
- `internal/ui/print.go` — print helpers (the canonical output functions)

The rubric criteria are:
1. **Icon consistency**: Uses icons from `theme.go` constants, not ad-hoc emoji
2. **Warmth**: Greetings and messages feel friendly, not robotic
3. **Actionable tips**: Suggestions tell the user what to do, not just what's wrong
4. **Error guidance**: Error messages explain what went wrong AND what to do about it
5. **Celebrations**: Small wins are acknowledged (completions, milestones)
6. **No raw fmt**: All user-facing output goes through `ui.*` helpers, not raw `fmt.Print*`

### 2. Scan Target Scope

Based on the argument, scan the appropriate files:

**CLI scope** (`cli` or full audit):
- All `cmd/*.go` files — read `Short`, `Long`, and `Example` fields on Cobra commands
- All `ui.*` calls across `cmd/` and `internal/` — check for consistent helper usage
- Search for raw `fmt.Print`, `fmt.Println`, `fmt.Printf` calls in `cmd/` that should
  use `ui.*` helpers instead
- Check string literals in `cmd/` for hardcoded emoji that should use `theme.go` constants

**Docs scope** (`docs` or full audit):
- `docs/*.md` — user-facing documentation
- `README.md` — project overview

**Site scope** (`site` or full audit):
- `site/` content files — landing page copy

### 3. Evaluate Against Rubric

For each file scanned, categorize findings into:

- **Strong**: Exemplary personality — warm, helpful, uses the right patterns
- **Flat**: Technically correct but missing personality — could be warmer or more helpful
- **Regression**: Previously good copy that has drifted (e.g., new command added with
  generic Short description)
- **Missing**: Expected personality touch that's absent (e.g., no celebration on task
  completion, no tip after first use)
- **Raw fmt**: Direct `fmt.Print*` call that should use a `ui.*` helper

### 4. Produce Structured Report

Present findings as a structured report:

```
Personality Audit — CLI scope

Strong (examples of good voice):
  cmd/todo.go:15    Short: "Manage your todo list like a boss"  -- warm, on-brand
  cmd/root.go:42    Dashboard greeting uses ui.Success()        -- correct helper

Flat (correct but could be better):
  cmd/stash.go:12   Short: "Track file versions"                -- functional but dry
  cmd/craft.go:18   Long: lists features without personality    -- reads like a manual

Regression:
  cmd/plugin.go:20  Short changed from playful to generic in recent commit

Missing:
  cmd/todo.go:89    No celebration message when all todos completed
  cmd/craft.go:45   No tip suggesting next command after scaffolding

Raw fmt:
  cmd/init.go:67    fmt.Printf("Created config at %s\n", path) -- should use ui.Success()
  cmd/stash.go:102  fmt.Println("Done.") -- should use ui.Success() with descriptive message

Summary: 8 files scanned, 2 strong, 3 flat, 1 regression, 2 missing, 2 raw fmt
```

### 5. Discuss Findings

Present the report and invite discussion:
- "The stash command is the flattest — want me to suggest warmer copy?"
- "I found 2 raw fmt calls that should use ui helpers. Want me to fix those?"
- "The plugin command Short field regressed — should I restore the original voice?"

Let the user decide which findings to act on. Don't push fixes for everything.

### 6. Apply Fixes (With Approval)

If the user wants fixes applied:
- Only change string literals and print calls — **never change logic**
- Show each proposed change before applying
- Group changes by file for clean diffs

**Always ask for explicit approval before editing any files.**

Fixes should be minimal and targeted:
- Replace raw `fmt.Print*` with appropriate `ui.*` helper
- Replace hardcoded emoji with `theme.go` icon constants
- Warm up flat `Short`/`Long` descriptions
- Add missing celebrations or tips

## Guidelines

- **Report first, fix later.** Always show the full audit report before proposing any
  changes. The user may disagree with your assessment.
- **Ground in concrete criteria.** Every finding should reference a specific rubric item
  (icon consistency, warmth, etc.) and a specific line of code. No vague "this could be
  better" without saying why and where.
- **Respect the existing voice.** The project already has a personality — you're checking
  for consistency, not imposing a new one. When suggesting copy, match the tone of the
  strongest existing examples.
- **Idempotent.** Running the audit twice on the same code should produce the same report.
  Don't flag things differently based on mood.
- **Don't overdo it.** Not every string needs personality. Debug output, internal logging,
  and structured data don't need warmth. Focus on user-facing touchpoints.
- **Know the helpers.** Before flagging a `fmt.Print*` call, verify that an appropriate
  `ui.*` helper exists. If none fits, note it as "no suitable helper" rather than
  suggesting a helper that doesn't exist.
