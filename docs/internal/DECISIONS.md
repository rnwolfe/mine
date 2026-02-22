# Architecture Decision Log

## ADR-001: Go over Rust for Implementation Language

**Date**: 2026-02-14
**Status**: Accepted

**Context**: Need a language that produces fast single binaries with excellent CLI UX library support.

**Decision**: Go 1.25+ with the Charm ecosystem (bubbletea, lipgloss, huh, bubbles).

**Rationale**:
- Rust toolchain not available in current environment; Go 1.25 is
- Charm ecosystem is the gold standard for terminal UX (used by GitHub CLI, etc.)
- Single binary output with trivial cross-compilation
- Fast compilation enables rapid iteration
- Pure Go SQLite avoids CGo complexity
- Still extremely fast for CLI workloads (<50ms target easily achievable)

**Tradeoffs**: Slightly larger binary than Rust equivalent. Acceptable.

---

## ADR-002: SQLite for Local Data Storage

**Date**: 2026-02-14
**Status**: Accepted

**Context**: Need persistent storage for todos, growth tracking, session data, etc.

**Decision**: SQLite via `modernc.org/sqlite` (pure Go, no CGo).

**Rationale**:
- Single file database, easy to backup/sync
- ACID compliant, handles concurrent reads
- No server process needed
- Pure Go implementation avoids CGo build complexity
- Well-understood, battle-tested technology

---

## ADR-003: TOML for Configuration Format

**Date**: 2026-02-14
**Status**: Accepted

**Context**: Need a human-editable configuration format.

**Decision**: TOML.

**Rationale**:
- Developer-native format (Cargo.toml, pyproject.toml, etc.)
- More readable than YAML for configuration (no indentation sensitivity)
- Strong typing support
- Well-supported in Go ecosystem
- Familiar to the target audience

---

## ADR-004: XDG Base Directory Compliance

**Date**: 2026-02-14
**Status**: Accepted

**Context**: Where to store config, data, cache.

**Decision**: Follow XDG Base Directory Specification.

**Rationale**:
- `~/.config/mine/` for config — portable, backed up
- `~/.local/share/mine/` for data — SQLite DB lives here
- `~/.cache/mine/` for cache — safe to delete
- Respects `$XDG_*` environment variables when set
- Clean, predictable, respects user's system organization

---

## ADR-005: Mining Metaphor — Light Touch

**Date**: 2026-02-14
**Status**: Superseded by ADR-006

**Context**: The name "mine" invites a mining/crafting metaphor.

**Decision**: Use the metaphor where it fits naturally, don't force it.

**Rationale**:
- `mine stash`, `mine craft`, `mine dig`, `mine vault` — these map naturally
- Don't rename "config" to "pickaxe" or "help" to "lantern"
- Whimsy should enhance UX, not obscure it
- New users should understand commands without knowing the metaphor

---

## ADR-006: Brand Identity — Personal Ownership

**Date**: 2026-02-22
**Status**: Accepted
**Supersedes**: ADR-005

**Context**: The name "mine" was interpreted as inviting a mining/crafting metaphor.
The original intent was possessive — "this is mine, for me." The metaphor added
confusion about brand identity and misdirected contributor decisions.

**Decision**: The brand identity is personal ownership, not mining. The name means
"yours." Command names (dig, craft, vault, stash) are developer vocabulary that
stands alone without a unifying metaphor.

**Rationale**:
- "Personal developer supercharger" is the product identity
- Command names are self-explanatory without metaphor context
- The possessive framing ("yours, your way") is stronger messaging
- Removes a layer of indirection between the name and what the tool does
