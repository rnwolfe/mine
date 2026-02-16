# Contributing

## Development Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/rnwolfe/mine.git
   cd mine
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Build and test:
   ```bash
   make build
   make test
   ```

## Workflow

1. **Branch from main**: `git checkout -b feat/my-feature`
2. **Make changes**: follow existing patterns in the codebase
3. **Test**: `make test` must pass
4. **Build**: `make build` must succeed
5. **Push and PR**: push your branch and open a PR

### Branch Naming

| Prefix | Purpose |
|--------|---------|
| `feat/` | New features |
| `fix/` | Bug fixes |
| `chore/` | Maintenance, cleanup |
| `docs/` | Documentation |

### PR Requirements

- CI must pass (vet, test with coverage, build, smoke test)
- Coverage must not drop below threshold
- Copilot code review runs automatically
- Human merges after reviewing

## Code Standards

### Architecture Rules

- `cmd/` files are **thin** — parse args, call domain logic, format output
- Business logic lives in `internal/` packages
- All terminal output goes through `internal/ui` helpers
- Tests live next to the code they test (`_test.go` suffix)
- Keep files under 500 lines

### Testing

- Write tests for all domain logic
- Use in-memory SQLite (`:memory:`) for database tests
- Run with race detector: `go test -race ./...`
- Aim for meaningful coverage, not 100%

### Style

- Follow Go conventions (`gofmt`, `go vet`)
- Error messages should say what went wrong AND what to do
- Use the mining metaphor naturally, don't force it
- Be whimsical in user-facing text, precise in code

## Adding a New Command

1. Create a domain package: `internal/myfeature/myfeature.go`
2. Write tests: `internal/myfeature/myfeature_test.go`
3. Create the command: `cmd/myfeature.go`
4. Register in `cmd/root.go`: `rootCmd.AddCommand(myFeatureCmd)`
5. Add documentation to `site/src/content/docs/commands/myfeature.md`
6. Update `CHANGELOG.md`

## Documentation

User-facing documentation lives in `site/src/content/docs/`. To update docs:

1. Navigate to `site/src/content/docs/`
2. Edit the relevant markdown file
3. Test locally: `cd site && npm install && npm run dev`
4. Build to verify: `npm run build`

The documentation site is built with Astro Starlight and auto-deploys to [mine.rwolfe.io](https://mine.rwolfe.io) via Vercel.
