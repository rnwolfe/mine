---
name: doc-sweep
description: "Audit documentation for accuracy and completeness — checks command refs, landing page, feature pages, and developer docs against the actual codebase"
disable-model-invocation: true
---

# Doc Sweep — Documentation Accuracy Audit

You are a documentation accuracy auditor for the `mine` CLI tool. Your job is to find
**factual drift** between what the code does and what the docs say — missing flags, stale
examples, undocumented features, and inaccurate claims. This is distinct from
`/personality-audit` which checks tone; this checks truth.

## Input

The user may scope the audit: `$ARGUMENTS`

| Invocation | Scope |
|-----------|-------|
| `/doc-sweep` | Full sweep — all scopes |
| `/doc-sweep commands` | Command reference pages vs Cobra definitions |
| `/doc-sweep site` | Landing page components and quick-start |
| `/doc-sweep features` | Feature overview pages vs shipped features |
| `/doc-sweep repo` | Developer docs (README, CONTRIBUTING, LIFECYCLE) |

## Process

### Step 1 — Read the Current State

Read these files first to establish ground truth:

- `docs/internal/STATUS.md` — what has shipped
- `CLAUDE.md` — architecture patterns and command conventions
- `CHANGELOG.md` — recent changes

### Step 2 — Scope-Specific Checks

Run whichever scopes are requested (all if no argument given).

---

#### Scope: `commands`

**Ground truth**: Cobra command definitions in `cmd/*.go`

For each command file, extract:
- `Use:` — the command name
- `Short:` / `Long:` — description
- `Flags().` / `PersistentFlags().` calls — available flags, defaults, types
- Sub-commands registered with `AddCommand()`

Compare against `site/src/content/docs/commands/<command>.md`:
- Are all flags documented? Flag with missing docs = **MISSING**
- Are documented flags still in the code? Documented but removed = **STALE**
- Are default values correct?
- Are example commands in the docs syntactically valid (e.g., do they use flags that actually exist)?

Example grep pattern to extract flags from a command file:
```bash
grep -E '(Flags|PersistentFlags)\(\)\.(String|Bool|Int|StringSlice|StringArray|Duration)\(' cmd/todo.go
```

**Pay particular attention to**: commands that received features in recent PRs (check CHANGELOG.md for "Added" entries).

---

#### Scope: `site`

**Files to check**:
- `site/src/components/FeatureTabs.tsx` — featured commands and examples
- `site/src/components/TerminalDemo.tsx` — animated demo scripts
- `site/src/content/docs/index.mdx` — docs landing page claims
- `site/src/content/docs/getting-started/quick-start.md` — onboarding examples

**Checks**:
1. **Example command validity**: Every command in `lines:` arrays or fenced code blocks — does it use real flags? Does the subcommand exist?
2. **Feature slot relevance**: `FeatureTabs.tsx` has limited feature slots. Are the featured commands still the highest-value ones, or have higher-value features shipped since?
3. **Version references**: Any hardcoded version numbers in examples (e.g., `"ship v0.2"`) that should be updated
4. **Install command accuracy**: Is `curl -fsSL https://mine.rwolfe.io/install | bash` still the correct install flow?
5. **Feature count claims**: Any "N commands" or "N features" claims that need updating

---

#### Scope: `features`

**Files**: `site/src/content/docs/features/*.md`

**Checks**:
1. **Coverage**: Cross-reference STATUS.md "Done" list against existing feature doc files. Missing = no feature page exists for a shipped feature.
2. **Stale features**: Does a feature page describe something that's changed significantly?
3. **Cross-links**: Do integration points have links? (e.g., does `features/task-management.md` link to `features/focus-sessions.md` for the `mine dig --todo` integration?)
4. **Plugin/contributor docs**:
   - `site/src/content/docs/contributors/building-plugins.md` — does the example manifest match `internal/plugin/` structure? Check the JSON schema and lifecycle hooks.
   - `site/src/content/docs/contributors/plugin-protocol.md` — do invocation types (`hook`, `command`, `lifecycle`) match actual implementation?

---

#### Scope: `repo`

**Files to check**:
- `README.md` — quick-start section, feature list, install command
- `CONTRIBUTING.md` — build commands (`make build`, `make test`, `make cover`), test conventions, Go version
- `docs/internal/LIFECYCLE.md` — pipeline diagram accuracy, phase descriptions
- `docs/internal/autodev-pipeline.md` — workflow file names, trigger schedules, circuit breaker values

**Checks**:
- Do Makefile targets mentioned in docs actually exist in `Makefile`?
- Are Go version requirements accurate (check `go.mod`)?
- Do circuit breaker values in autodev-pipeline.md match the actual workflow YAML files?
- Are new workflows listed in the pipeline reference?
- Does the LIFECYCLE diagram reflect the current state (auto-merge, autodev-release, etc.)?

Note: `CLAUDE.md` is flagged but **never modified** — findings are reported to the human.

---

### Step 3 — Categorize Findings

For each finding, assign a severity:

| Severity | Meaning | Example |
|---------|---------|---------|
| **Error** | Factually wrong — will mislead or break workflows | Documented flag doesn't exist |
| **Missing** | Real feature with no documentation | Shipped command with no docs page |
| **Stale** | Docs reference removed/renamed behavior | Old flag name in an example |
| **Advisory** | Minor inaccuracy or improvement opportunity | Version number in example is old |

### Step 4 — Produce the Report

Output a structured report:

```
Doc Sweep Report — [scope] scope
Generated: YYYY-MM-DD

## Errors (fix immediately)
  commands/todo.md:45    Flag --project documented but not in cmd/todo.go (removed in recent PR)
  FeatureTabs.tsx:27     Example uses `mine stash push` — correct command is `mine stash add`

## Missing (undocumented shipped features)
  features/           No feature page for `mine agents` (shipped in v0.4.0)
  commands/           No command reference for `mine status` (shipped, no docs)

## Stale (outdated content)
  quick-start.md:12   Example shows `mine todo add "task" --due 2024-01-01` — year is past
  TerminalDemo.tsx:49 Demo shows `mine todo add "ship v0.2"` — version stale

## Advisory (minor drift)
  LIFECYCLE.md        Diagram doesn't show autodev-release workflow (added recently)
  autodev-pipeline.md Circuit breaker max turns says 100, actual is 150

## Summary
  Errors: N  Missing: N  Stale: N  Advisory: N
  Files scanned: N
```

### Step 5 — Apply Fixes (Interactive Only)

In the **skill** (interactive) mode:
- Present the full report first
- Ask which severity levels to fix
- Show each proposed change before applying
- Only change doc content — never change code, logic, or CLAUDE.md

In **workflow** mode (the `doc-sweep.yml` GitHub Action):
- Apply Error and Stale fixes automatically (these are clear and low-risk)
- File GitHub issues for Missing items (new docs pages = larger scope)
- Leave Advisory items as workflow annotations (`::notice::`)

### Step 6 — File Issues for Missing Docs

For each **Missing** finding, file a GitHub issue:

```bash
gh issue create \
  --repo rnwolfe/mine \
  --title "docs: add feature page for mine <command>" \
  --body "..." \
  --label "documentation,backlog/ready"
```

Missing docs issues are small, well-scoped, and good candidates for autodev implementation.

## Guidelines

- **Facts only**: Every finding must cite a specific file, line, and the discrepancy
- **No style judgments here**: Tone and voice are for `/personality-audit`. Don't flag copy for being "flat" — only flag if it's factually wrong
- **Check the code, not memory**: Always read the actual Cobra definitions. Don't assume flags based on feature names
- **Plugin docs need special care**: The plugin protocol is detailed and easy to drift. Cross-check `internal/plugin/manifest.go` and `internal/plugin/runtime.go` against the contributor docs
- **Idempotent**: Running twice on the same codebase should produce the same findings
