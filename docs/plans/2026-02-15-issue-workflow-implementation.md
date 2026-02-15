# GitHub Issue Workflow Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Update CLAUDE.md with explicit instructions for verifying acceptance criteria and ensuring issues auto-close when PRs merge.

**Architecture:** Documentation-only change. Add new "GitHub Issue Workflow" section to CLAUDE.md with step-by-step agent instructions for PR creation workflow.

**Tech Stack:** Markdown, GitHub CLI (`gh`), git

---

## Task 1: Add GitHub Issue Workflow Section to CLAUDE.md

**Files:**
- Modify: `CLAUDE.md:100-109` (insert new section after "Development Workflow")

**Context:** The "Development Workflow" section currently ends at line 109. We'll insert a new section immediately after it, before "Release Process" (line 111).

**Step 1: Read current CLAUDE.md to confirm insertion point**

Run: `head -120 CLAUDE.md | tail -30`
Expected: Should see "Development Workflow" section ending around line 109, then "Release Process" starting at line 111.

**Step 2: Insert new section after Development Workflow**

Add the following section between lines 109 and 110 (after "Development Workflow", before "Release Process"):

```markdown
## GitHub Issue Workflow

When creating a PR that implements a GitHub issue, follow this workflow to ensure acceptance criteria are verified and issues auto-close on merge.

### Before Creating the PR

1. **Read the original issue**
   ```bash
   gh issue view N --json body,title
   ```
   Extract acceptance criteria (look for checkbox lists in the issue body).

2. **Verify each acceptance criterion**
   - Review your code changes against each criterion
   - Confirm each one is satisfied by the implementation
   - Note any criteria that are NOT met (incomplete scope is OK, document it)

### Creating the PR

1. **Document acceptance criteria in PR body**

   Add an "## Acceptance Criteria" section in your PR body:

   ```markdown
   ## Acceptance Criteria

   Verified against issue #N:
   - [x] Criterion 1 — Met by [specific implementation detail]
   - [x] Criterion 2 — Met by [specific implementation detail]
   - [ ] Criterion 3 — Out of scope for this PR, will address in #M
   ```

2. **Update the original issue checkboxes**

   For completed criteria, check them off in the issue itself:
   ```bash
   # Read current issue body
   gh issue view N --json body -q .body > /tmp/issue-body.txt

   # Edit the file to check boxes (change [ ] to [x])
   # Then update the issue
   gh issue edit N --body "$(cat /tmp/issue-body.txt)"
   ```

   This makes completion status visible directly in the issue.

3. **Use closing keywords**

   Include one of these in your PR title or body:
   - `Fixes #N`
   - `Closes #N`
   - `Resolves #N`

   This triggers GitHub's auto-close behavior when the PR merges.

### What If Issue Has No Acceptance Criteria?

If the issue doesn't have clear acceptance criteria:
- Note this in your PR body: "Issue #N has no formal acceptance criteria"
- List what you implemented anyway for reviewer clarity
- Consider adding a comment to the issue suggesting criteria for future similar issues

### Manual Verification Fallback

If a human asks you to verify acceptance criteria for an already-merged PR:

**Command:** Comment tag like `@claude please verify acceptance criteria against PR #N`

**Your response:**
1. Read the merged PR: `gh pr view N --json files,additions,deletions`
2. Read the issue: `gh issue view M --json body`
3. Verify each criterion against the code changes
4. Update the issue checkboxes if not already done
5. Add a comment to the issue summarizing verification results

**Example verification comment:**
```markdown
Verified acceptance criteria against PR #22:

- [x] Criterion 1 — ✅ Met (implemented in commit abc123)
- [x] Criterion 2 — ✅ Met (tests added in commit def456)
- [ ] Criterion 3 — ⚠️ Not addressed in this PR

Overall: 2/3 criteria met. Criterion 3 should be tracked separately.
```
```

**Step 3: Verify the section renders correctly**

Run: `grep -A 5 "## GitHub Issue Workflow" CLAUDE.md`
Expected: Should show the new section header and first few lines of content.

**Step 4: Commit the changes**

```bash
git add CLAUDE.md
git commit -m "docs: add GitHub issue workflow for acceptance criteria

Adds explicit agent instructions for verifying acceptance criteria and
using closing keywords to ensure issues auto-close when PRs merge.

Implements design from docs/plans/2026-02-15-issue-workflow-design.md

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

Expected: Clean commit with CLAUDE.md as the only changed file.

---

## Task 2: Add Lesson Learned Entry

**Files:**
- Modify: `CLAUDE.md:133-174` (add new lesson after L-008)

**Context:** Document this as a lesson learned so future agents understand the context.

**Step 1: Add L-009 entry after L-008**

Add this entry after the L-008 section (after line 173):

```markdown
### L-009: Acceptance criteria must be explicitly verified
Issue #8 was implemented in PR #22, but the acceptance criteria were never verified
and the issue didn't auto-close because the PR didn't use closing keywords. Agents
must read the issue, verify each criterion, update issue checkboxes, and use
`Fixes #N` / `Closes #N` / `Resolves #N` in the PR body. See "GitHub Issue Workflow"
section for the full workflow.
```

**Step 2: Verify the entry is added**

Run: `grep -A 4 "### L-009" CLAUDE.md`
Expected: Should show the new lesson with full text.

**Step 3: Commit the addition**

```bash
git add CLAUDE.md
git commit -m "docs: add lesson learned about acceptance criteria verification

Documents L-009 based on issue #8 / PR #22 where acceptance criteria
were not verified and issue didn't auto-close.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

Expected: Clean commit.

---

## Task 3: Validation Test

**Files:**
- None (test only)

**Context:** Verify that the new instructions are clear and actionable by simulating the workflow.

**Step 1: Test reading instructions**

Run: `grep -A 50 "## GitHub Issue Workflow" CLAUDE.md | head -60`
Expected: Should see complete workflow with all subsections clearly formatted.

**Step 2: Test gh commands are valid**

Run the example commands against a real issue (use issue #8 for testing):

```bash
# This should work (read-only)
gh issue view 8 --json body,title

# This should work (read-only)
gh pr view 22 --json files
```

Expected: Both commands return valid JSON data.

**Step 3: Manual review checklist**

Verify the documentation includes:
- [ ] How to read an issue
- [ ] How to extract acceptance criteria
- [ ] How to document verification in PR
- [ ] How to update issue checkboxes
- [ ] Which closing keywords to use
- [ ] What to do if issue has no criteria
- [ ] How to handle manual verification requests

Expected: All items present and clear.

**Step 4: No commit needed**

This is a validation-only task.

---

## Success Criteria

After completing this plan:

✅ CLAUDE.md has new "GitHub Issue Workflow" section with complete agent instructions
✅ Lesson L-009 documents the context and solution
✅ All `gh` command examples are syntactically correct
✅ Instructions are actionable (specific commands, not vague guidance)
✅ Both commits are pushed to the current branch

## Notes

- This is a documentation-only change, no code modifications
- The workflow relies on agent discipline, not automation (per design decision)
- Future enhancement could add this to `superpowers:finishing-a-development-branch` skill
- PR template changes are explicitly out of scope (per design decision)
