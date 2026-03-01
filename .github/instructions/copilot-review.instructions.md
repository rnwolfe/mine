# Copilot Review Instructions for mine

## Project Context

mine is a Go CLI tool built with Cobra, Lipgloss/Bubbletea TUI, and SQLite (modernc.org/sqlite, pure Go). Single binary, no CGo.

## File Size Limit

All non-test Go files under `cmd/` and `internal/` must stay under 500 lines. Flag any file exceeding this limit. Known exceptions are tracked in `.github/filelen-exceptions.txt`.

## Error Handling (Priority: High)

This is the most frequent review feedback category. Flag these patterns:

- **Silent error ignoring**: Any `err` from a database query, I/O operation, or HTTP call that is discarded (assigned to `_` or unchecked). Errors from `sql.Query`, `sql.QueryRow().Scan()`, `sql.Exec`, `os.Open`, `os.ReadFile`, `json.Marshal/Unmarshal`, etc. must be checked.
- **Catch-all error masking**: Using any `err != nil` as "no rows found". The correct pattern is to check `errors.Is(err, sql.ErrNoRows)` specifically and return/wrap all other errors.
- **Partial failure coupling**: When a function performs multiple independent operations (e.g., querying streak AND querying total minutes), one failure should not prevent the other from succeeding. Flag code where a single `if err != nil { return }` blocks unrelated data.
- **Missing error wrapping**: Bare `return err` without context. Prefer `fmt.Errorf("operation context: %w", err)`.

## Store Pattern

Each feature domain has a package under `internal/` with a `Store` type that owns all SQL. The `cmd/` layer should never contain raw SQL queries — it calls store methods. Flag any `db.Query`, `db.Exec`, or `db.QueryRow` in `cmd/*.go` files.

## UI Output

All user-facing output must use helpers from `internal/ui` (`ui.Ok`, `ui.Warn`, `ui.Err`, `ui.Tip`, `ui.Kv`, `ui.Accent.Render()`). Flag any raw `fmt.Println` or `fmt.Printf` that produces user-facing output in `cmd/` files. Error messages should include what went wrong AND what to do about it.

## Testing Standards

- Tests use `t.TempDir()` + `t.Setenv()` for isolation — never touch the real filesystem.
- Integration tests should call the actual `runXxx` handler, not just internal helpers.
- External tool tests (tmux, git, editors) should mock via fake scripts in `t.TempDir()` on `$PATH`.
- When tests modify package-level global variables (e.g., Cobra flag vars), they should restore them via `t.Cleanup`.

## Hook Wrapping

All command handlers should be wrapped with `hook.Wrap("command.name", handlerFunc)` to participate in the plugin pipeline. Flag any `RunE` that directly assigns a handler without `hook.Wrap`.

## Code Style

- `cmd/` files are thin orchestration: parse args, call domain logic, format output.
- `internal/` packages own their domain and don't import each other unnecessarily.
- Emoji in output must use icon constants from `internal/ui/theme.go` — no raw emoji literals.
