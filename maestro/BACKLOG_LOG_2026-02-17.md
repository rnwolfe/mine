# Backlog Log — 2026-02-17

## Loop 00001 — Issue #19: Anonymous usage analytics (opt-out)
- **Branch:** maestro/issue-19-anonymous-usage-analytics-opt-out
- **PR:** https://github.com/rnwolfe/mine/pull/107
- **Status:** PR opened
- **Files changed:** 14

### Loop 00001 — Documentation & Follow-up
- **Doc pages updated:** `site/src/content/docs/commands/config.md` (added set/get subcommands), `site/src/content/docs/docs/privacy.md` (fixed inaccurate goroutine description)
- **Follow-up issues created:** #109 (CLAUDE.md updates for analytics package), #110 (deploy analytics ingest backend)
- **CLAUDE.md changes needed:** yes — add `internal/analytics/analytics.go`, `internal/analytics/id.go` to Key Files table; add `internal/analytics/` to File Organization; document analytics pattern and `*bool` config pattern in Architecture Patterns

### Loop 00001 — Finalized
- **Issue:** #19
- **PR:** #107
- **Status:** maestro/review-ready — awaiting human review
- **Worktree:** cleaned up

---
Loop 00001 complete. More issues available — continuing to next loop.

## Loop 00002 — Issue #28: User-local hooks (~/.config/mine/hooks/)
- **Branch:** maestro/issue-28-user-local-hooks
- **PR:** https://github.com/rnwolfe/mine/pull/111
- **Status:** PR opened
- **Files changed:** 5

### Loop 00002 — Documentation & Follow-up
- **Doc pages updated:** `site/src/content/docs/commands/hook.md` (new command reference page for `mine hook`), 3 example hook scripts in `docs/examples/hooks/`
- **Follow-up issues created:** none — all scope covered, no technical debt
- **CLAUDE.md changes needed:** no — already updated in the PR (architecture pattern #11, key files for `registry.go` and `cmd/hook.go`)

### Loop 00002 — Finalized
- **Issue:** #28
- **PR:** #111
- **Status:** maestro/review-ready — awaiting human review
- **Worktree:** cleaned up
