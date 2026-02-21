# mine — Product Development Lifecycle

> The complete skill pipeline from strategic direction to shipped feature.

Every skill in `.claude/skills/` maps to a stage in this lifecycle. This document shows
how they connect, when to use each one, and what the hand-offs look like.

---

## The Cycle

```
┌─────────────────────────────────────────────────────────────────────┐
│                         STRATEGY LAYER                              │
│                                                                     │
│   /product                         /product sync                   │
│   Health check: phase gaps,        Reconcile VISION.md +           │
│   vision drift, synergy map,       STATUS.md with what's           │
│   ranked priorities                actually shipped                 │
│                │                              ▲                    │
└────────────────┼──────────────────────────────┼────────────────────┘
                 │ priorities + gaps             │ merged PRs
                 ▼                              │
┌─────────────────────────────────────────────────────────────────────┐
│                         DISCOVERY LAYER                             │
│                                                                     │
│   /brainstorm [area]               /product spec "idea"            │
│   Generate ideas in a focus        Vision filter: identity,        │
│   area; conversational, no         principle, phase, replacement.   │
│   vision filter enforced           Hard no if it fails.            │
│                │                              │                    │
└────────────────┼──────────────────────────────┼────────────────────┘
                 │ ideas                         │ passes filter
                 ▼                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       SPECIFICATION LAYER                           │
│                                                                     │
│   /product spec                                                     │
│   Auto-picks highest-value unspecced roadmap feature.              │
│   Writes docs/internal/specs/<name>.md with strategic rationale,  │
│   architecture, dependencies, acceptance criteria, out-of-scope.   │
│                │                                                   │
└────────────────┼────────────────────────────────────────────────────┘
                 │ spec document
                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       BACKLOG ENTRY LAYER                           │
│                                                                     │
│   /product spec (creates issue)    /draft-issue "rough idea"       │
│   Full spec → issue with spec      Ad-hoc path: no prior spec,    │
│   link, all sections populated,    conversational drafting,        │
│   spec + phase labels              iterate until ready             │
│                │                              │                    │
└────────────────┼──────────────────────────────┼────────────────────┘
                 └──────────────┬───────────────┘
                                │ open GitHub issue
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      BACKLOG QUALITY LAYER                          │
│                                                                     │
│   /sweep-issues [label]            /product eval N                 │
│   Score all open issues on         Score one issue on vision,      │
│   10-item quality checklist.       phase, and design principles.   │
│   Labels: needs-refinement,        Output: READY / REFINE /        │
│   agent-ready suggestions          DECLINE with reasoning          │
│                │                              │                    │
│                ▼                              │                    │
│   /refine-issue [N]                           │                    │
│   Fill gaps via targeted Q&A.                 │                    │
│   Updates issue body. Suggests                │                    │
│   agent-ready when bar is met.                │                    │
│                │                             │                     │
└────────────────┼─────────────────────────────┼────────────────────┘
                 └──────────────┬──────────────┘
                                │ agent-ready label
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      IMPLEMENTATION LAYER                           │
│                                                                     │
│   /autodev [N]                     GitHub Actions autodev          │
│   CLI loop: pick issue,            Scheduled loop: same steps      │
│   fresh worktree off main,         running autonomously on push    │
│   implement, make test + build,    or 4-hour cron. Phased review   │
│   open PR with acceptance          pipeline: Copilot → Claude →    │
│   criteria verified.               done.                           │
│                │                              │                    │
└────────────────┼──────────────────────────────┼────────────────────┘
                 └──────────────┬───────────────┘
                                │ PR merged
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    QUALITY ASSURANCE LAYER                          │
│                                                                     │
│   /autodev-audit                   /personality-audit              │
│   Pipeline health: PR quality,     Tone consistency: CLI output,   │
│   code pattern compliance,         docs, site copy. Flags drift    │
│   iteration stats, improvement     from the "whimsical but         │
│   opportunities.                   competent" voice.               │
│                │                              │                    │
└────────────────┼──────────────────────────────┼────────────────────┘
                 └──────────────┬───────────────┘
                                │ feedback
                                ▼
                       (back to STRATEGY)
```

---

## Layer Reference

### Strategy Layer — *Where should we go?*

| Skill | When | Output |
|-------|------|--------|
| `/product` | Before any major work batch; when direction feels unclear | Roadmap health report, ranked priorities |
| `/product sync` | After a release or significant merge batch | Updated VISION.md + STATUS.md committed |

**Cadence**: Run `/product` at the start of each work session or sprint. Run `/product sync`
after every merge batch or release.

**Rule**: No implementation work should start without a current `/product` health check.
If you don't know what the priorities are, run it.

---

### Discovery Layer — *What specifically should we build?*

| Skill | When | Output |
|-------|------|--------|
| `/brainstorm [area]` | Backlog thin in an area; exploring new territory | 3-5 ideas with scope and fit analysis |
| `/product spec "idea"` | You have a specific idea and want to know if it belongs | Vision filter result + spec if it passes |

**Hand-off**: Ideas that pass the vision filter in `/product spec "idea"` go directly to
Specification. `/brainstorm` outputs are more exploratory — good ideas still need to pass
the vision filter before becoming specs.

**Rule**: `/brainstorm` is exploratory and permissive. `/product spec "idea"` is the
gatekeeping step. Any idea that becomes a spec should have passed the vision filter.

---

### Specification Layer — *What exactly are we building and why?*

| Skill | When | Output |
|-------|------|--------|
| `/product spec` | Picking up a new feature area; need spec before implementation | `docs/internal/specs/<name>.md` |

**What a spec contains**:
- Strategic rationale (why `mine` needs this, not just why it's useful)
- Command surface
- Architecture and domain package
- Integration points with existing features
- Dependencies (internal and external)
- Acceptance criteria
- Explicit out-of-scope

**Rule**: Complex features need a spec before a GitHub issue. The spec is the design
artifact; the issue is the implementation contract. Don't skip the spec for medium-or-larger
features — vague issues produce vague implementations.

---

### Backlog Entry Layer — *Get it into the queue*

| Skill | When | Output |
|-------|------|--------|
| `/product spec` (issue creation step) | After spec is approved | GitHub issue with spec link and full template |
| `/draft-issue "rough idea"` | Ad-hoc capture of a self-contained idea that doesn't need a spec | GitHub issue, iteratively drafted |

**When to use `/draft-issue` vs `/product spec`**:
- Small enhancement to an existing command → `/draft-issue` is fine
- New command, new domain, or cross-feature integration → spec first, then issue

---

### Backlog Quality Layer — *Is it ready to build?*

| Skill | When | Output |
|-------|------|--------|
| `/sweep-issues [label]` | Backlog has grown; preparing for a work batch | Issues labeled `needs-refinement` or `agent-ready` |
| `/refine-issue [N]` | An issue is labeled `needs-refinement` | Updated issue body, possible `agent-ready` label |
| `/product eval N` | Spot-check: is this specific issue vision-aligned and ready? | READY / REFINE / DECLINE with reasoning |

**The quality gate**: `agent-ready` means the issue is specific enough to implement
autonomously without human clarification. Don't label an issue `agent-ready` unless it
has clear acceptance criteria, architecture notes, and defined scope.

**Cadence**: Run `/sweep-issues` before a work batch. Use `/refine-issue` to process
the `needs-refinement` queue. Use `/product eval N` for spot-checks on individual issues.

---

### Implementation Layer — *Build it*

| Skill | When | Output |
|-------|------|--------|
| `/autodev [N]` | `agent-ready` issues in the backlog; want to implement now | PR with tests passing, acceptance criteria verified |
| GitHub Actions autodev | Ongoing; runs on schedule or push trigger | Same as above, but autonomous |

**Prerequisites**: Issue must be `agent-ready`. If it isn't, go back to Backlog Quality.

**Rule**: Never open a PR with failing `make test` or broken `make build`. Verify
acceptance criteria explicitly in the PR description.

---

### Quality Assurance Layer — *Is what shipped good?*

| Skill | When | Output |
|-------|------|--------|
| `/autodev-audit` | After several implementation cycles; pipeline feels slow or inconsistent | Audit report with specific improvement recommendations |
| `/personality-audit [area]` | Before a release; after bulk doc/CLI changes | Tone report with specific flagged strings |

**Cadence**: Run `/autodev-audit` monthly or after every 5-10 autodev PRs. Run
`/personality-audit` before any release.

---

## Entry Points

Starting from different states, here's where to enter the cycle:

| Your situation | Start here |
|----------------|------------|
| New work session, unclear what to do | `/product` |
| Vision/status docs feel stale | `/product sync` |
| Have a vague idea, want to explore | `/brainstorm` |
| Have a specific idea, want to know if it fits | `/product spec "idea"` |
| Ready to spec a known roadmap gap | `/product spec` |
| Have a quick idea to capture | `/draft-issue "idea"` |
| Backlog has grown, messy | `/sweep-issues` |
| Specific issue needs improvement | `/refine-issue N` |
| Want to check if issue is vision-ready | `/product eval N` |
| Issue is `agent-ready`, want to ship it | `/autodev N` |
| Autodev pipeline feels off | `/autodev-audit` |
| CLI output/docs feel inconsistent | `/personality-audit` |

---

## The Complete Pipeline (Linear View)

When working through a full feature from scratch:

```
/product          → identify the gap
/product spec     → write the spec (docs/internal/specs/)
                  → create the GitHub issue
/product eval N   → confirm it passes the vision filter
/sweep-issues     → confirm quality score
/refine-issue N   → fill any gaps
                  → label agent-ready
/autodev N        → implement in worktree
                  → make test + make build
                  → open PR
                  → human merges
/product sync     → update STATUS.md with what shipped
/product          → next health check
```

---

## Skill Ownership Map

| Layer | Primary | Supporting |
|-------|---------|------------|
| Strategy | `/product` | `/product sync` |
| Discovery | `/brainstorm` | `/product spec "idea"` |
| Specification | `/product spec` | — |
| Backlog Entry | `/product spec`, `/draft-issue` | — |
| Backlog Quality | `/sweep-issues`, `/refine-issue` | `/product eval N` |
| Implementation | `/autodev` | GitHub Actions autodev |
| Quality Assurance | `/autodev-audit`, `/personality-audit` | — |
