# Issue Workflow: Acceptance Criteria Review and Auto-Close

**Date:** 2026-02-15
**Status:** Approved
**Problem:** Issues with acceptance criteria aren't being verified before PR merge, and issues don't auto-close when PRs merge.

## Problem Statement

When PR #22 implemented issue #8:
- The PR body mentioned the issue but didn't use GitHub closing keywords
- The 6 acceptance criteria in issue #8 were never marked complete
- The issue didn't auto-close when the PR merged
- The issue was manually closed 10 minutes later
- No clear record exists of whether acceptance criteria were actually met

This leaves maintainers uncertain whether features were fully implemented and creates manual cleanup work.

## Root Causes

1. **No agent instructions** for using GitHub closing keywords (`Fixes #N`, `Closes #N`, `Resolves #N`)
2. **No agent instructions** for verifying acceptance criteria before creating PRs
3. **No agent instructions** for updating the original issue to mark criteria complete
4. **No documentation** in CLAUDE.md or CONTRIBUTING.md about this workflow

## Solution Design

### Approach: Agent Discipline via CLAUDE.md Instructions

Add explicit workflow instructions to CLAUDE.md that agents must follow when creating PRs. This leverages the existing mechanism (agents read CLAUDE.md for project guidance) without adding new files or automation.

**Trade-off accepted:** Some "leaks" requiring manual handling are tolerable. This is simpler than GitHub Actions automation and more flexible than rigid enforcement.

### Agent Behavior: PR Creation Workflow

When an agent creates a PR that implements a GitHub issue, it must:

1. **Read the original issue**
   - Use `gh issue view N --json body` to get the full issue text
   - Extract acceptance criteria (look for checkbox lists)

2. **Verify each acceptance criterion**
   - Review code changes against each criterion
   - Document verification in PR body under "## Acceptance Criteria" section
   - Format: checklist showing which criteria were met and how

3. **Update the original issue**
   - Use `gh issue edit N --body <updated-body>` to check off completed boxes
   - Preserve all other issue content (don't overwrite)
   - This makes completion status visible directly in the issue

4. **Use closing keyword**
   - Include `Fixes #N`, `Closes #N`, or `Resolves #N` in PR title or body
   - This triggers GitHub's auto-close behavior when PR merges

**Result:** Issue shows checked boxes, PR documents verification, issue auto-closes on merge, humans can verify completion at a glance.

### Manual Verification Fallback

If a human notices an issue wasn't updated (acceptance criteria unchecked after PR merge):

**Action:** Comment on the issue: `@claude please verify acceptance criteria against PR #N`

**Agent response:**
1. Read the merged PR changes
2. Verify each acceptance criterion against the implementation
3. Update the issue checkboxes
4. Add a comment summarizing verification results

## Implementation Scope

### Files to Modify

1. **CLAUDE.md** - Add new "GitHub Issue Workflow" section in "Development Workflow"
2. **CLAUDE.md** - Update "Creating pull requests" section with acceptance criteria steps

### What NOT to Change

- No PR template changes (avoids boilerplate in simple PRs)
- No GitHub Actions automation (keeps it simple, avoids CI complexity)
- No new documentation files (consolidate in CLAUDE.md)

## Success Criteria

After implementation, agents should:
- ✅ Always use closing keywords when implementing issues
- ✅ Read and verify acceptance criteria before creating PRs
- ✅ Update original issue checkboxes when criteria are met
- ✅ Document verification in PR body
- ✅ Issues auto-close when PRs merge

Acceptable failure modes:
- Occasional "leaks" requiring manual verification (tolerable)
- Issues without clear acceptance criteria (agent should note this in PR)

## Future Enhancements (Not in Scope)

- GitHub Actions to validate closing keywords are present
- PR template section for acceptance criteria (if leaks become frequent)
- Skill integration in `superpowers:finishing-a-development-branch`
- Automated acceptance criteria generation for issues that lack them

## References

- Issue #8: https://github.com/rnwolfe/mine/issues/8
- PR #22: https://github.com/rnwolfe/mine/pull/22
- GitHub auto-close docs: https://docs.github.com/en/issues/tracking-your-work-with-issues/linking-a-pull-request-to-an-issue
