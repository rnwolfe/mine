# Key Files

> Agents should keep this table updated when adding new files or changing file purposes.

| File | Purpose |
|------|---------|
| `cmd/root.go` | Dashboard, command registration |
| `cmd/todo.go` | Todo CRUD commands |
| `cmd/proj.go` | Project CLI commands (add, rm, list, open, scan, config) |
| `cmd/plugin.go` | Plugin CLI commands (install, remove, search, info) |
| `internal/ui/theme.go` | Colors, icons, style constants |
| `internal/store/store.go` | DB connection, migrations |
| `internal/proj/proj.go` | Project domain logic — registry CRUD, scan, open state, settings |
| `internal/todo/todo.go` | Todo domain logic + queries |
| `internal/config/config.go` | Config load/save, XDG paths |
| `internal/hook/hook.go` | Hook types, Context, Handler interface |
| `internal/hook/pipeline.go` | Hook pipeline (Wrap, stage execution, flag rewrites) |
| `internal/hook/discover.go` | User hook discovery, script creation, testing |
| `internal/hook/registry.go` | Thread-safe hook registry with glob pattern matching |
| `internal/hook/exec.go` | ExecHandler — runs external hook scripts |
| `cmd/hook.go` | Hook CLI commands (list, create, test) |
| `internal/plugin/manifest.go` | Plugin manifest parsing and validation |
| `internal/plugin/lifecycle.go` | Plugin install, remove, list, registry management |
| `internal/plugin/runtime.go` | Plugin invocation (hooks, commands, lifecycle events) |
| `internal/plugin/permissions.go` | Permission sandboxing, env builder, audit log |
| `internal/plugin/search.go` | GitHub search for mine plugins |
| `cmd/stash.go` | Stash CLI commands (track, commit, log, restore, sync) |
| `internal/stash/stash.go` | Stash domain logic — git-backed versioning, manifest, sync |
| `cmd/agents.go` | Agents CLI commands (init, commit, log, restore) |
| `internal/agents/agents.go` | Agents domain — canonical store, git versioning, manifest (JSON), restore with copy-mode link re-sync |
| `cmd/craft.go` | Craft CLI commands (dev, ci, git, list) |
| `internal/craft/recipe.go` | Recipe engine, registry, template execution |
| `internal/craft/builtins.go` | Built-in recipe definitions (go, node, python, rust, docker, github CI) |
| `internal/tui/picker.go` | Reusable fuzzy-search picker (Bubbletea model, Run helper) |
| `internal/tui/fuzzy.go` | Fuzzy matching algorithm (subsequence with scoring) |
| `internal/tmux/tmux.go` | Tmux session management (list, new, attach, kill) |
| `internal/tmux/layout.go` | Layout persistence (save/load/list, TOML in XDG config) |
| `cmd/tmux.go` | Tmux CLI commands with TUI picker integration |
| `cmd/config.go` | Config CLI commands (show, path, list, get, set, unset, edit) |
| `cmd/env.go` | Env CLI commands (show, set, unset, diff, switch, export, template, inject) |
| `internal/env/env.go` | Env manager: profile CRUD, age encryption/decryption, active profile tracking, diff, export |
| `internal/vault/vault.go` | Age-encrypted secret store (set, get, delete, list, export, import) |
| `internal/vault/keychain.go` | `PassphraseStore` interface, `noopKeychain`, `ErrNotSupported`, `IsKeychainMiss` helper |
| `internal/vault/keychain_darwin.go` | macOS keychain implementation via `security` CLI |
| `internal/vault/keychain_linux.go` | Linux keychain implementation via `secret-tool`; falls back to noop if not installed |
| `internal/vault/keychain_other.go` | No-op `NewPlatformStore()` for unsupported platforms |
| `cmd/vault.go` | Vault CLI commands; `vaultKeychainStore` var (injectable in tests); passphrase resolution |
| `internal/analytics/analytics.go` | Analytics ping, daily dedup via SQLite, privacy-safe payload construction |
| `internal/analytics/id.go` | Installation ID (UUIDv4) generation and persistence |
| `scripts/autodev/config.sh` | Autodev shared constants, logging, utilities |
| `scripts/autodev/pick-issue.sh` | Issue selection with trust verification |
| `scripts/autodev/parse-reviews.sh` | Extract review feedback for agent consumption |
| `scripts/autodev/check-gates.sh` | Quality gate verification (CI, iterations, mergeable) |
| `scripts/autodev/open-pr.sh` | PR creation with auto-merge and iteration tracking |
| `scripts/autodev/agent-exec.sh` | Model-agnostic agent execution abstraction |
| `.github/workflows/autodev-audit.yml` | Weekly pipeline health audit (files issue) |
| `docs/internal/autodev-pipeline.md` | Autodev pipeline architecture deep-dive |
| `site/astro.config.mjs` | Astro + Starlight config (sidebar, social links, plugins) |
| `site/src/content/docs/index.mdx` | Landing page (hero, features, quick start) |
| `site/src/content/docs/getting-started/` | Installation and quick start guides |
| `site/src/content/docs/features/` | Feature overview pages (high-level, links to command reference) |
| `site/src/content/docs/commands/` | Command reference pages (full flags, subcommands, error tables) |
| `site/src/content/docs/contributors/` | Architecture and plugin protocol docs |
| `site/src/styles/custom.css` | Gold/amber brand theming |
| `site/vercel.json` | Vercel deployment config (Astro preset, rewrites) |
