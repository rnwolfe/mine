# Issue #19: Anonymous usage analytics (opt-out)

## Status
READY

## Labels
enhancement, phase:2, agent-ready, in-progress, maestro

## Worktree
/home/rnwolfe/dev/mine-worktrees/issue-19

## Body
## Summary

Add lightweight, anonymous usage analytics to track growth and active users. Enabled by default with easy opt-out. Privacy-first: no PII, no tracking IDs tied to identity, no command arguments or user data.

## Motivation

Understanding adoption and usage patterns is critical for prioritizing features and measuring growth. Without analytics, we're flying blind on which commands are used, how many active users exist, and whether releases are being adopted.

## Scope

### What gets collected
- Random installation ID (UUIDv4, generated on `mine init`, stored locally)
- mine version and OS/arch
- Command name invoked (e.g., `todo`, `dig`, `craft` — NOT arguments or data)
- Timestamp (day granularity, not exact time)
- Session count (daily active, not per-invocation)

### What is NEVER collected
- Command arguments, flags, or values
- Todo content, file paths, config values, secrets
- IP addresses (use privacy-respecting ingest that strips IPs)
- Any form of PII

### Commands
- `mine config set analytics false` — Opt out
- `mine config set analytics true` — Opt back in
- First run after install shows one-time notice explaining analytics + how to opt out

### Implementation
- Lightweight HTTP POST to analytics endpoint (fire-and-forget, non-blocking)
- Zero impact on command latency (async, no waiting for response)
- Fail silently on network errors (offline-first)
- Backend: consider PostHog (self-hostable, generous free tier), Plausible, or simple custom endpoint
- Daily dedup: only send one ping per day per command category (not per invocation)

## Acceptance Criteria

- [ ] Analytics enabled by default, opt-out via config
- [ ] One-time notice on first run explaining what's collected and how to opt out
- [ ] No PII collected — audit and document data schema
- [ ] Non-blocking: commands remain < 50ms regardless of network
- [ ] Fails silently when offline or endpoint unreachable
- [ ] `internal/analytics/` package with clean interface
- [ ] Tests for payload construction, opt-out behavior, dedup logic
- [ ] Privacy policy section added to docs/site
- [ ] Installation ID stored in `~/.local/share/mine/` (XDG data dir)

## Design Notes

- Fire-and-forget goroutine: `go analytics.Ping(cmd)` at end of command execution
- Daily dedup via kv table: `analytics:last_ping:todo = 2026-02-14`
- Backend recommendation: PostHog — open source, self-hostable, has Go SDK, free up to 1M events/month
- Privacy notice text should be friendly, not legalese:
  ```
  mine sends anonymous usage stats (command names, version, OS) to help
  improve the tool. No personal data is ever collected.
  Opt out anytime: mine config set analytics false
  ```

## Priority

Medium — valuable for growth tracking but not a user-facing feature. Best implemented early in Phase 2 so we start collecting data as user base grows.

## PR
- **Number:** 107
- **URL:** https://github.com/rnwolfe/mine/pull/107
- **Branch:** maestro/issue-19-anonymous-usage-analytics-opt-out

## Agentic Knowledge Base

- [ ] Update `CLAUDE.md` (symlinked as `agents.md`) with any new architecture patterns, key files, domain packages, or lessons learned introduced by this work
