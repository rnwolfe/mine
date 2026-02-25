---
name: code-sweep
description: "Audit codebase for architectural drift, pattern violations, complexity creep, dead code, and test gaps from continuous autonomous development"
disable-model-invocation: true
---

# Code Sweep — Codebase Health Audit

You are a codebase health auditor for the `mine` CLI tool. Your job is to detect
**architectural drift** — the kind of quality degradation that passes individual PR
review but accumulates across dozens of agent-implemented features.

This is distinct from `go vet` (syntax-level issues) and `/autodev-audit` (pipeline
health). This checks **structural health**: pattern compliance, complexity, duplication,
dead code, and test gaps.

## Input

The user may scope the audit: `$ARGUMENTS`

| Invocation | Scope |
|-----------|-------|
| `/code-sweep` | Full sweep — all scopes |
| `/code-sweep patterns` | Pattern violations only |
| `/code-sweep complexity` | File/function size and nesting |
| `/code-sweep dead-code` | Unused exports, stale helpers |
| `/code-sweep tests` | Test coverage gaps |

## Process

### Step 1 — Read Project Conventions

Read `CLAUDE.md` for the authoritative list of patterns and rules:
- Architecture patterns (store pattern, UI consistency, domain separation)
- File size limits (500 lines per file, 50 lines per function target)
- Test standards (unit vs integration, isolation, naming)
- Error message standards

### Step 2 — Scope-Specific Checks

---

#### Scope: `patterns`

**Raw fmt usage** (should use `internal/ui` helpers):
```bash
grep -rn "fmt\.Print" cmd/ --include="*.go" | grep -v "_test.go"
grep -rn "fmt\.Print" internal/ --include="*.go" | grep -v "_test.go"
```
For each hit: is there an appropriate `ui.*` helper? If yes, flag as **violation**.

**Direct SQL outside store** (all SQL should live in `internal/store/` or domain packages' own store files):
```bash
grep -rn "\.Query\|\.Exec\|\.QueryRow" cmd/ --include="*.go"
```

**os.Exit in non-main context**:
```bash
grep -rn "os\.Exit" cmd/ --include="*.go" | grep -v "main\.go"
grep -rn "os\.Exit" internal/ --include="*.go"
```

**Cross-package imports that violate domain separation**:
- `cmd/` importing `internal/` is OK
- `internal/A` importing `internal/B` is a yellow flag — check if it's necessary
- Any import cycle risk: `internal/A → internal/B → internal/A`

**Missing error wrapping** (bare `return err` where context would help):
```bash
grep -rn "return err$" cmd/ --include="*.go" | grep -v "_test.go" | head -20
```
Evaluate if the caller provides context. Bare `return err` at the cmd handler level is usually fine; inside internal helpers it often loses context.

**Hardcoded XDG paths** (should use config helpers):
```bash
grep -rn '\.config/mine\|\.local/share/mine\|\.cache/mine' --include="*.go"
```
Except in tests where temp dirs are used.

---

#### Scope: `complexity`

**Files over 500 lines**:
```bash
find . -name "*.go" -not -path "*/vendor/*" -exec wc -l {} + | awk '$1 > 500 {print $2, $1}' | sort -k2 -rn
```

**Functions over 50 lines** (heuristic — use wc on function bodies):
```bash
# Look for long function bodies using pattern matching
grep -n "^func " cmd/*.go internal/**/*.go 2>/dev/null | head -5
```
Flag any file with a function the auditor judges to be significantly over 50 lines after reading.

**Deeply nested conditionals** (more than 3 levels):
Read files flagged for complexity and look for patterns like:
```go
if x {
    if y {
        if z {
            // 4+ levels
        }
    }
}
```

---

#### Scope: `dead-code`

**Unexported functions not called within their package**:
Read each `internal/` package. For each unexported function, check if it's called from the same package's `.go` files (excluding `_test.go`). If not called from anywhere, it's likely dead.

```bash
# List all unexported functions
grep -rn "^func [a-z]" internal/ --include="*.go" | grep -v "_test.go"
```

**Exported functions/types not used outside their package**:
```bash
grep -rn "^func [A-Z]\|^type [A-Z]" internal/ --include="*.go" | grep -v "_test.go"
```
Cross-reference: is it imported and used by `cmd/` or another `internal/` package? If not, it may be dead (exported for testing is OK if tests use it).

**Stale TODO/FIXME comments** (note: check git blame age if possible):
```bash
grep -rn "TODO\|FIXME\|HACK\|XXX" --include="*.go" | grep -v "_test.go"
```
Flag any that reference features or issues that have since shipped.

---

#### Scope: `tests`

**Exported functions without test coverage**:
```bash
grep -rn "^func [A-Z]" internal/ --include="*.go" | grep -v "_test.go"
```
For each exported function, check if a corresponding test exists in `*_test.go` files in the same package.

**`cmd/` handlers without integration tests**:
```bash
grep -rn "^func run" cmd/ --include="*.go" | grep -v "_test.go"
```
Each `runXxx` handler should have a `TestRunXxx_*` test. List handlers missing tests.

**Test isolation violations** (tests touching real filesystem):
```bash
grep -rn "os\.Getenv\|os\.MkdirAll\|os\.WriteFile" cmd/ --include="*_test.go" | grep -v "t\.TempDir\|t\.Setenv"
```
Tests should use `t.TempDir()` and `t.Setenv()` — never touch real XDG dirs.

**Coverage from `docs/internal/coverage.json`**:
Read `docs/internal/coverage.json` if it exists. Highlight packages below 40% as **priority** test targets.

### Step 3 — Categorize Findings

| Severity | Action |
|---------|--------|
| **Fix** | Clear violation of CLAUDE.md rules — apply directly (PR) |
| **Refactor** | Improvement opportunity — file an issue labeled `backlog/ready` |
| **Investigate** | Possible issue needing human judgment — file an issue labeled `backlog/needs-refinement` |

### Step 4 — Report

```
Code Sweep Report — [scope]
Generated: YYYY-MM-DD

## Fix (clear violations — PR eligible)
  cmd/todo.go:45    raw fmt.Printf — should use ui.Success()
  internal/ai/ai.go:12  os.Exit(1) in non-main context

## Refactor (improvement opportunities — issues filed)
  internal/hook/pipeline.go  712 lines — exceeds 500-line limit, candidate for split
  cmd/env.go  runEnvEdit function is 89 lines — candidate for extraction

## Investigate (needs human judgment — issues filed)
  internal/plugin/ → internal/hook/  cross-package import — intentional?
  internal/store/kv.go:88  exported KVGet not used outside package — dead?

## Test Gaps
  internal/contrib/  ContribList exported, no test coverage
  cmd/agents.go  runAgentsDiff — no integration test

## Summary
  Fix: N  Refactor: N  Investigate: N  Test gaps: N
  Files scanned: N
```

### Step 5 — Apply Fixes and File Issues

**In skill (interactive) mode**:
- Present report
- Ask which categories to fix
- Show each proposed change before applying

**In workflow mode** (`code-sweep.yml`):
- Apply **Fix** items directly (string/call changes, not logic changes)
- File GitHub issues for **Refactor** and **Investigate** items
- File GitHub issues for **Test Gap** items labeled `backlog/ready` (these are well-scoped)

## Issue Template for Filed Items

```bash
gh issue create \
  --repo rnwolfe/mine \
  --title "refactor: [brief description from finding]" \
  --body "## Summary

  [Finding from code sweep]

  **File**: path/to/file.go:line
  **Category**: [Refactor|Investigate|Test Gap]
  **Detected by**: code-sweep workflow — [DATE]

  ## What to do

  [Specific action: extract function, add test, remove dead code, etc.]

  ## Acceptance Criteria

  - [ ] [specific criterion]" \
  --label "enhancement,backlog/ready"
```

## Guidelines

- **Read the code, don't guess**: Always read the actual file before flagging. Don't flag based on pattern name alone.
- **Skip generated code**: Any file with a `// Code generated` header is off-limits.
- **Test files are different**: `_test.go` files have different rules — `fmt.Println` in tests is OK, longer functions are OK.
- **Context matters for cross-package imports**: Some cross-package imports are correct by design (e.g., `internal/store` is used by all domain packages). Check CLAUDE.md's domain separation rules before flagging.
- **Don't flag style as dead code**: An exported function used only in tests is not dead.
