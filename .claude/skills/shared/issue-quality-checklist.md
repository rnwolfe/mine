# Gold-Standard Issue Template

This defines the quality bar for `mine` GitHub issues. Based on the format established
by issue #35 (Environment variable manager). All backlog curation skills target this
template as the output format.

## Template

```markdown
## Summary

One paragraph explaining what this feature/enhancement does and why it matters.
Should answer: What problem does this solve? Who benefits? How does it fit into mine's vision?

## Subcommands / Features

| Command | Description |
|---------|-------------|
| `mine <cmd> <sub>` | What it does |
| `mine <cmd> <sub> --flag` | What the flag changes |

> Omit this section if the issue is a bug fix, refactor, or single-behavior enhancement.

## Architecture / Design Notes

- **Domain package**: `internal/<pkg>/` — where the core logic lives
- **Storage**: SQLite table design, new migrations, or "no storage needed"
- **Key technical decisions**: Libraries, algorithms, protocols
- **Security considerations**: Input validation, encryption, access control
- **Performance**: Anything that could affect the <50ms budget

## Integration Points

How this connects to existing mine features:
- Which existing commands are affected?
- Does it fire or consume hook events?
- Does it interact with config, store, or plugins?
- Are there dependencies on other planned features?

## Acceptance Criteria

- [ ] Specific, testable criterion that maps to one verifiable behavior
- [ ] Another criterion — include happy path AND edge cases
- [ ] Error handling: what happens when X fails?
- [ ] Test requirements: unit tests for domain logic, integration tests if needed
- [ ] Performance: command completes within the <50ms budget (if applicable)

> Each checkbox should be independently verifiable. Avoid vague criteria like
> "works correctly" — instead say "returns exit code 0 and prints confirmation message".

## Documentation

- [ ] User-facing docs: `docs/commands/<cmd>.md`
- [ ] Internal specs (if complex): `docs/internal/specs/<feature>.md`
- [ ] CLAUDE.md updates: new key files, architecture patterns, or lessons learned
```

## Quality Checklist

When evaluating an issue, check each item:

- [ ] **Summary**: Clear one-paragraph explanation of what and why
- [ ] **Scope**: Features/subcommands enumerated (if applicable)
- [ ] **Architecture**: Domain package, storage approach, key decisions documented
- [ ] **Integration**: Connections to existing features identified
- [ ] **Acceptance criteria**: Specific, testable checkboxes (not vague)
- [ ] **Edge cases**: Error handling and failure modes covered in criteria
- [ ] **Tests**: Test requirements included in acceptance criteria
- [ ] **Documentation**: Doc requirements listed with specific file paths
- [ ] **CLAUDE.md**: Update requirement noted for new patterns/key files
- [ ] **Labels**: Appropriate labels applied (feature/enhancement, phase, etc.)

## Label Guide

| Label | When to use |
|-------|-------------|
| `feature` | Brand new capability (new command, new domain) |
| `enhancement` | Improvement to existing feature |
| `phase:1` | Foundation features (core CLI, config, store) |
| `phase:2` | Growth features (craft, dig, shell, ai) |
| `phase:3` | Advanced features (vault, grow, dash, plugins) |
| `good-first-issue` | Clear scope, isolated domain, good for new contributors |
| `spec` | Has a spec document in `docs/internal/specs/` |
| `backlog/ready` | Issue is well-defined enough for autonomous implementation |
