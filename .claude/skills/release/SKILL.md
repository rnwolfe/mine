---
name: release
description: "Cut a release: analyze unreleased work, propose semver bump, draft CHANGELOG, tag, and push"
disable-model-invocation: true
---

# Release — Ship It

You are the release manager for the `mine` CLI tool. Your job is to take everything
that's been merged to `main` since the last tag and turn it into a clean, versioned
release: the right semver bump, a well-written CHANGELOG entry, and a tagged commit
that triggers GoReleaser.

You are the last gate before users see new features. Be deliberate.

---

## Input

```
/release           — Full interactive release flow: analyze → propose → draft → confirm → tag
/release check     — Dry-run only: show what's unreleased, proposed version, CHANGELOG preview
/release notes     — Draft CHANGELOG entry only (no tagging, no commits)
```

The user may also provide a version override: `/release v0.3.0`

---

## Step 1 — Read Context

Read these files before doing anything else:

- `CHANGELOG.md` — current release history and `[Unreleased]` section
- `CLAUDE.md` — release process section and project conventions
- `.goreleaser.yaml` — what GoReleaser does (to describe it accurately)

Then collect live data:

```bash
# Last tag (current released version)
# If this is the first release and no tags exist yet, fall back to the initial commit.
if [ -n "$(git tag --list)" ]; then
  LAST_TAG=$(git describe --tags --abbrev=0)
else
  echo "No Git tags found; assuming first release. Using initial commit as LAST_TAG."
  LAST_TAG=$(git rev-list --max-parents=0 HEAD)
fi

# All commits since last tag (or since initial commit for first release)
git log ${LAST_TAG}..HEAD --oneline --no-merges

# Merged PRs since last tag — the authoritative source
gh pr list --repo rnwolfe/mine \
  --state merged \
  --json number,title,body,mergedAt,labels \
  --limit 50 | \
  jq --arg since "$(git log ${LAST_TAG} -1 --format=%aI)" \
  '[.[] | select(.mergedAt > $since)]'

# Any open PRs that might affect release readiness
gh pr list --repo rnwolfe/mine --state open \
  --json number,title,labels \
  --label "human/blocked"

# Check if there's anything already in [Unreleased] in CHANGELOG.md
# (read the file — already done above)
```

---

## Step 2 — Categorize the Changes

Go through each merged PR since the last tag. Classify by conventional commit type
(read the PR title prefix or infer from the content):

| Conventional type | CHANGELOG category |
|-------------------|--------------------|
| `feat:` | **Added** |
| `fix:` | **Fixed** |
| `refactor:` | **Changed** |
| `docs:` | **Documentation** (omit if trivial) |
| `chore:`, `ci:`, `test:` | Omit unless user-visible |
| `perf:` | **Changed** |
| Breaking change (`!` suffix or `BREAKING CHANGE:` in body) | **Breaking** |

For each PR, extract the user-facing impact — not the implementation detail. "feat: add
`mine release` skill" is fine for internal tooling, but "feat: implement store migration"
should be described as the user-visible effect, e.g., "Automatic database schema
migrations on upgrade."

---

## Step 3 — Propose a Semver Bump

Apply semver rules strictly:

- **Patch** (`v0.2.0` → `v0.2.1`): Only bug fixes. No new commands, no new flags, no
  behavior changes. Purely `fix:` PRs.

- **Minor** (`v0.2.0` → `v0.3.0`): New features that are backwards-compatible. New
  commands, new subcommands, new flags. Any `feat:` PR.

- **Major** (`v0.2.0` → `v1.0.0`): Breaking changes. Removed commands, changed flag
  names, incompatible config format changes, breaking protocol changes.

Since we're pre-1.0, `v0.x.y`: breaking changes bump minor (not major), new features
bump minor, fixes bump patch. When in doubt, bump minor — underversioning (calling a
minor a patch) is the only real mistake.

**Pre-release suffixes**: If the changes are experimental or the feature set is
incomplete, propose a pre-release tag (e.g., `v0.3.0-alpha.1`). GoReleaser marks these
as pre-release automatically.

State your proposed version and reasoning clearly before proceeding.

---

## Step 4 — Run the Pre-Release Checklist

Work through this checklist and report the result of each item:

```
[ ] No open PRs with human/blocked label
[ ] All merged PRs in scope have conventional commit titles (can infer type)
[ ] [Unreleased] section in CHANGELOG.md is accurate (matches what's actually merged)
[ ] STATUS.md is not severely stale (last sync within ~5 PRs)
[ ] make test passes on current HEAD (recommend running, but don't block on it if CI is green)
```

For each failing item, note it but do not abort. The human decides what's a blocker.

**Advisory checks** (warn but never block):
- Suggest `/personality-audit cli` if more than 3 new commands were added
- Suggest `/product sync` if STATUS.md wasn't updated recently

---

## Step 5 — Draft the CHANGELOG Entry

Draft the new version section following the existing format in `CHANGELOG.md`:

```markdown
## [X.Y.Z] - YYYY-MM-DD

### Added

- **Feature Name** (`mine cmd sub`) — One sentence describing what users can now do.
  Keep it user-facing, not implementation-focused.

### Fixed

- Bug description — what was wrong and what it does now instead.

### Changed

- Changed behavior — old behavior → new behavior.

### Breaking

- **Command name** — What changed and what users need to update.
```

Rules for the draft:
- Group by category (Added / Fixed / Changed / Breaking)
- Within each category, lead with the highest user-impact items
- Omit `chore:`, `ci:`, `test:` PRs unless they change observable behavior
- If `[Unreleased]` already has content in CHANGELOG.md, merge it with what you found
  from the PR list (de-duplicate)
- Bold the feature name for scanability
- Include the command surface in backticks when applicable

---

## Step 6 — Present Summary and Request Confirmation

Show the user everything before touching any files:

```
Release Summary
───────────────
Current version:  v0.2.0-alpha.1
Proposed version: v0.3.0
PRs in scope:     12 (10 feat, 2 fix)
Last tag date:    2026-02-18

Pre-release checklist:
  ✓ No human/blocked PRs
  ✓ All PRs have conventional commit titles
  ✗ STATUS.md may be stale — last sync was 8 PRs ago (advisory)
  ~ make test not run (CI was green on last merge)

Proposed CHANGELOG entry:
<draft entry>

Proposed tag: v0.3.0
Tag command:  git tag v0.3.0 && git push origin v0.3.0

Proceed? (y to continue, or specify a different version)
```

**Wait for explicit confirmation before proceeding.** Do not tag or commit without it.

If the user provides a different version, use that instead. If they say the CHANGELOG
needs changes, make them and re-present before continuing.

---

## Step 7 — Update CHANGELOG.md

Move the drafted section into CHANGELOG.md:

1. Replace the `## [Unreleased]` section content with an empty `[Unreleased]` stub
2. Insert the new versioned section immediately after `[Unreleased]`
3. Update the comparison link at the bottom if CHANGELOG.md uses them

The resulting top of CHANGELOG.md should look like:

```markdown
## [Unreleased]

## [X.Y.Z] - YYYY-MM-DD

### Added
...
```

---

## Step 8 — Commit the CHANGELOG

```bash
git add CHANGELOG.md
git commit -m "chore: release v${VERSION}

Co-Authored-By: Claude <noreply@anthropic.com>"
```

Do not include other files in this commit. The release commit should only be the
CHANGELOG update — this keeps the git history clean and makes release archaeology easy.

---

## Step 9 — Tag and Push

```bash
git tag "v${VERSION}"
git push origin main
git push origin "v${VERSION}"
```

The tag push triggers the `release.yml` GitHub Actions workflow, which runs GoReleaser:
- Compiles 4 binaries (linux/darwin × amd64/arm64)
- Creates `tar.gz` archives and `checksums.txt`
- Publishes a GitHub Release with GoReleaser's changelog

---

## Step 10 — Report

Print a clean summary:

```
✓ Released mine v${VERSION}

  Tag:       v${VERSION}
  Changelog: CHANGELOG.md updated
  Pipeline:  https://github.com/rnwolfe/mine/actions (GoReleaser running)
  Release:   https://github.com/rnwolfe/mine/releases/tag/v${VERSION} (available in ~2 min)

Next steps:
  • /product sync — update STATUS.md to reflect what shipped
  • Monitor https://github.com/rnwolfe/mine/actions for GoReleaser completion
  • Verify install: curl -fsSL https://mine.rwolfe.io/install | bash
```

---

## Mode: Dry-Run Check (`/release check`)

Run Steps 1–5 only. After the summary in Step 6, **stop**. Do not modify any files,
do not commit, do not tag. Present the full picture — version proposal, checklist,
CHANGELOG draft — and exit. This mode is safe to run at any time.

---

## Mode: Draft Notes Only (`/release notes`)

Run Steps 1–5. Write the CHANGELOG draft to `/tmp/release-notes-draft.md` and display
it. Do not update CHANGELOG.md. Do not commit. Do not tag. Useful for reviewing what
the entry would look like before committing to a release.

---

## Guardrails

- **Never tag without explicit confirmation.** The tag push triggers GoReleaser and
  is not easily undone. Always present the full summary and wait for user approval.
- **Never push to main without confirmation.** Same rule as the tag.
- **Never force-push.** If a tag already exists for the proposed version, stop and
  ask the user what to do. Do not delete or move tags.
- **Never version-bump without reasoning.** Always explain why patch vs. minor vs. major.
- **Respect pre-release tags.** If the last tag was `v0.2.0-alpha.1`, the next release
  could be `v0.2.0-alpha.2` (another pre-release) or `v0.2.0` (stable). Ask the user
  which they intend unless it's obvious from context.
- **CHANGELOG is the source of truth.** If `[Unreleased]` already has accurate content,
  use it. Don't discard it in favor of auto-generated content from PR titles.

## Error Recovery

| Situation | Action |
|-----------|--------|
| Tag already exists | Stop. Show existing tag. Ask if user wants a different version. |
| No PRs since last tag | Report "nothing to release" and exit |
| Push fails (auth) | Show exact error. Never retry with --force. |
| GoReleaser fails | Link to the Actions run. Don't attempt manual recovery. |
| User wants to undo | Provide the exact commands to delete the tag locally and remotely, but do not run them — tagging is reversible but GoReleaser may have already published a release |
