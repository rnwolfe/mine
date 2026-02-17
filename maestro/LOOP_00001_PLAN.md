# Implementation Plan: Issue #19 — Anonymous usage analytics (opt-out)

## Approach

Add an `internal/analytics/` package that provides a fire-and-forget `Ping(command)` function called at the end of command execution via the existing `hook.Wrap` pipeline. Analytics data is minimal (installation ID, version, OS/arch, command name, date) and sent via a non-blocking HTTP POST. Opt-out is managed through a new `analytics` field in the TOML config, with a `mine config set` subcommand for toggling. Daily deduplication uses the existing `kv` table in SQLite. A one-time privacy notice is shown on first run after the feature lands.

## Files to Create

- `internal/analytics/analytics.go` — Core analytics package: `Ping()`, payload construction, dedup logic, HTTP send, opt-out check
- `internal/analytics/analytics_test.go` — Tests for payload construction, opt-out behavior, dedup logic, HTTP send (using httptest)
- `internal/analytics/id.go` — Installation ID management: generate UUIDv4, read/write from XDG data dir (`~/.local/share/mine/analytics_id`)
- `internal/analytics/id_test.go` — Tests for ID generation and persistence

## Files to Modify

- `internal/config/config.go` — Add `AnalyticsConfig` struct with `Enabled bool` field (default: `true`) to `Config`
- `cmd/config.go` — Add `mine config set <key> <value>` subcommand for toggling config values (starting with `analytics`)
- `cmd/root.go` — Import analytics package; add a `PersistentPostRunE` on rootCmd that calls `go analytics.Ping(cmd.Name())` after every command
- `cmd/init.go` — Generate installation ID during `mine init`; display one-time privacy notice
- `internal/store/store.go` — No changes needed — the `kv` table already exists and is used by `dig` for state tracking

## Architecture Decisions

- **Package boundary**: `internal/analytics/` is a self-contained package. It imports `internal/config` (for opt-out check), `internal/version` (for version string), and `internal/store` (for kv-based dedup). It does NOT import `internal/ui` — all output happens in `cmd/` layer.
- **Installation ID storage**: Plain text file at `~/.local/share/mine/analytics_id` (XDG data dir, already created by `config.Paths.EnsureDirs()`). Not in SQLite — the ID must be readable without opening the database, and survives DB resets.
- **Daily dedup**: Uses the existing `kv` table with keys like `analytics:last_ping:todo = 2026-02-17`. Before sending, check if today's date matches; if so, skip. After successful send, write today's date.
- **Config default**: `analytics.enabled = true` in TOML. When the field is missing (existing installs), Go's zero value is `false` — so we need a tri-state or explicit default. Solution: use a `*bool` pointer in the config struct, and treat `nil` as `true` (opt-out, not opt-in). Alternatively, use `defaultConfig()` to set `true`.
- **Backend endpoint**: The analytics endpoint URL will be a constant in the package. For the initial implementation, use a configurable endpoint URL with a sensible default. The actual backend (PostHog, custom, etc.) is an infrastructure decision independent of the client code. The package will POST a JSON payload and not care what receives it.
- **No external SDK**: Use stdlib `net/http` for the POST. No PostHog Go SDK dependency — keeps the binary lean and avoids coupling to a specific vendor. The JSON payload format can be compatible with PostHog's `/capture` API if desired.
- **`config set` subcommand**: Implement a generic `mine config set <key> <value>` that supports dotted keys (e.g., `analytics.enabled`). This is useful beyond analytics and follows the issue's suggested UX (`mine config set analytics false`). Keep it simple: only support the known config keys, reject unknown ones.

## CLI Surface

- `mine config set analytics false` — Opt out of analytics
- `mine config set analytics true` — Opt back in
- `mine config set <key> <value>` — Generic config setter (extensible)
- `mine config get <key>` — Read a single config value (useful companion)

No new top-level commands. Analytics runs silently in the background — no user-facing analytics commands needed.

## Test Strategy

### Unit tests for:
- **Payload construction** (`analytics_test.go`): Verify JSON payload contains expected fields (install ID, version, OS, arch, command, date) and no unexpected fields (no args, no file paths)
- **Opt-out behavior** (`analytics_test.go`): When `analytics.enabled = false` in config, `Ping()` returns immediately without HTTP call
- **Dedup logic** (`analytics_test.go`): First call for a command today sends; second call same day skips; different command same day sends; same command next day sends
- **HTTP send** (`analytics_test.go`): Use `httptest.Server` to verify correct HTTP method, content-type, and payload format
- **Installation ID** (`id_test.go`): Generate creates valid UUIDv4; subsequent reads return same ID; missing file triggers generation
- **Config set** (in `cmd/` or `config_test.go`): Verify `analytics` key toggles the config field

### Edge cases:
- Network timeout/failure — verify `Ping()` does not block or error visibly
- Missing installation ID file — generates one on first ping
- Corrupt/empty ID file — regenerates
- Config file missing analytics section — defaults to enabled

### Integration points:
- `PersistentPostRunE` on rootCmd — verify it fires for subcommands
- `mine init` — verify ID generation and privacy notice display
- `mine config set analytics false` → subsequent `Ping()` is no-op

## Risks & Considerations

- **`PersistentPostRunE` conflicts**: Cobra's `PersistentPostRunE` on rootCmd runs after ALL subcommands. Need to verify no existing commands set their own `PersistentPostRunE` that would be overridden. Alternative: use a wrapper pattern in `hook.Wrap` to inject the analytics ping at the notify stage. This is actually cleaner and aligns with the existing architecture (notify stage = fire-and-forget async).
- **Better approach — hook.Wrap integration**: Instead of `PersistentPostRunE`, register a global notify-stage hook that calls `analytics.Ping()`. This is more consistent with the existing plugin/hook architecture and avoids Cobra lifecycle conflicts. The hook pipeline already has a fire-and-forget notify stage (`go runNotifyStage(...)`) which is the perfect place for analytics.
- **Binary size**: stdlib `net/http` is already likely pulled in by other dependencies. No additional bloat.
- **Privacy compliance**: The data schema must be documented and auditable. No PII by design — command name only (not args), UUIDv4 (not tied to identity), day-granularity timestamp.
- **First-run notice timing**: The one-time notice should appear after `mine init` completes, not during. For existing users who upgrade, the notice should appear on the first command after the upgrade. Use a kv flag (`analytics:notice_shown = true`) to track.
- **Config pointer semantics**: Using `*bool` for the `Enabled` field means TOML encoding/decoding needs care. The `BurntSushi/toml` library handles pointer fields correctly. `defaultConfig()` should set `Enabled` to a `true` pointer.

## Acceptance Criteria Mapping

- [x] Analytics enabled by default, opt-out via config → `AnalyticsConfig.Enabled` defaults to `true`; `mine config set analytics false` sets it to `false`
- [x] One-time notice on first run explaining what's collected and how to opt out → kv flag `analytics:notice_shown`; notice displayed in dashboard or init
- [x] No PII collected — audit and document data schema → Payload struct is explicit: ID, version, OS, arch, command, date. Tests verify no extra fields
- [x] Non-blocking: commands remain < 50ms regardless of network → `go analytics.Ping(cmd)` in fire-and-forget goroutine (via notify hook stage)
- [x] Fails silently when offline or endpoint unreachable → HTTP client with short timeout (2s), errors logged at debug level only
- [x] `internal/analytics/` package with clean interface → `analytics.Ping(command string)` is the only public API needed
- [x] Tests for payload construction, opt-out behavior, dedup logic → Comprehensive test file with table-driven tests
- [x] Privacy policy section added to docs/site → Add page at `site/src/content/docs/docs/privacy.md`
- [x] Installation ID stored in `~/.local/share/mine/` (XDG data dir) → File at `{DataDir}/analytics_id`
