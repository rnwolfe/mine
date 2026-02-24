package agents

import (
	"os"
	"path/filepath"
	"testing"
)

// ── parseFrontmatterDescription ───────────────────────────────────────────────

func TestParseFrontmatterDescription_SingleLine(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "SKILL.md")
	content := `---
name: my-skill
description: Reviews code changes and provides feedback
---

## Instructions

Do stuff.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got := parseFrontmatterDescription(path)
	want := "Reviews code changes and provides feedback"
	if got != want {
		t.Errorf("parseFrontmatterDescription() = %q, want %q", got, want)
	}
}

func TestParseFrontmatterDescription_FoldedMultiLine(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "SKILL.md")
	content := `---
name: my-skill
description: >
  TODO: Describe what this skill does and when to use it.
---

## Instructions
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got := parseFrontmatterDescription(path)
	want := "TODO: Describe what this skill does and when to use it."
	if got != want {
		t.Errorf("parseFrontmatterDescription() = %q, want %q", got, want)
	}
}

func TestParseFrontmatterDescription_NoFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "SKILL.md")
	content := `# My Skill

Just a regular markdown file.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got := parseFrontmatterDescription(path)
	if got != "" {
		t.Errorf("parseFrontmatterDescription() = %q, want empty string", got)
	}
}

func TestParseFrontmatterDescription_MissingFile(t *testing.T) {
	got := parseFrontmatterDescription("/nonexistent/SKILL.md")
	if got != "" {
		t.Errorf("parseFrontmatterDescription() = %q, want empty string for missing file", got)
	}
}

// ── parseMarkdownDescription ──────────────────────────────────────────────────

func TestParseMarkdownDescription_FirstNonHeadingLine(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "cmd.md")
	content := `# deploy

Deploy to production.

## Steps
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got := parseMarkdownDescription(path)
	want := "Deploy to production."
	if got != want {
		t.Errorf("parseMarkdownDescription() = %q, want %q", got, want)
	}
}

func TestParseMarkdownDescription_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "cmd.md")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	got := parseMarkdownDescription(path)
	if got != "" {
		t.Errorf("parseMarkdownDescription() = %q, want empty string", got)
	}
}

func TestParseMarkdownDescription_OnlyHeadings(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "cmd.md")
	content := `# Title

## Section
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got := parseMarkdownDescription(path)
	if got != "" {
		t.Errorf("parseMarkdownDescription() = %q, want empty string for headings-only file", got)
	}
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestList_EmptyStore(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	result, err := List(ListOptions{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(result.Skills) != 0 {
		t.Errorf("Skills = %d, want 0", len(result.Skills))
	}
	if len(result.Commands) != 0 {
		t.Errorf("Commands = %d, want 0", len(result.Commands))
	}
}

func TestList_AfterAddSkill(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	if _, err := AddSkill("code-review"); err != nil {
		t.Fatalf("AddSkill() error: %v", err)
	}

	result, err := List(ListOptions{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(result.Skills) != 1 {
		t.Fatalf("Skills = %d, want 1", len(result.Skills))
	}
	if result.Skills[0].Name != "code-review" {
		t.Errorf("Skills[0].Name = %q, want %q", result.Skills[0].Name, "code-review")
	}
}

func TestList_AfterAddCommand(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	if _, err := AddCommand("deploy"); err != nil {
		t.Fatalf("AddCommand() error: %v", err)
	}

	result, err := List(ListOptions{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(result.Commands) != 1 {
		t.Fatalf("Commands = %d, want 1", len(result.Commands))
	}
	if result.Commands[0].Name != "deploy" {
		t.Errorf("Commands[0].Name = %q, want %q", result.Commands[0].Name, "deploy")
	}
}

func TestList_TypeFilter_SkillsOnly(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	if _, err := AddSkill("my-skill"); err != nil {
		t.Fatal(err)
	}
	if _, err := AddCommand("deploy"); err != nil {
		t.Fatal(err)
	}

	result, err := List(ListOptions{Type: "skills"})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(result.Skills) != 1 {
		t.Errorf("Skills = %d, want 1", len(result.Skills))
	}
	// Commands should be nil when filtered to skills.
	if len(result.Commands) != 0 {
		t.Errorf("Commands = %d, want 0 when type=skills", len(result.Commands))
	}
}

func TestList_TypeFilter_CommandsOnly(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	if _, err := AddCommand("deploy"); err != nil {
		t.Fatal(err)
	}
	if _, err := AddSkill("my-skill"); err != nil {
		t.Fatal(err)
	}

	result, err := List(ListOptions{Type: "commands"})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(result.Commands) != 1 {
		t.Errorf("Commands = %d, want 1", len(result.Commands))
	}
	if len(result.Skills) != 0 {
		t.Errorf("Skills = %d, want 0 when type=commands", len(result.Skills))
	}
}

func TestList_SkillDescriptionFromFrontmatter(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	// Scaffold a skill and overwrite SKILL.md with a known description.
	result, err := AddSkill("code-review")
	if err != nil {
		t.Fatalf("AddSkill() error: %v", err)
	}

	custom := `---
name: code-review
description: Reviews code changes and provides feedback
---

## Instructions

Do stuff.
`
	if err := os.WriteFile(result.SKILLMD, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	listResult, err := List(ListOptions{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(listResult.Skills) != 1 {
		t.Fatalf("Skills = %d, want 1", len(listResult.Skills))
	}
	wantDesc := "Reviews code changes and provides feedback"
	if listResult.Skills[0].Description != wantDesc {
		t.Errorf("Skills[0].Description = %q, want %q", listResult.Skills[0].Description, wantDesc)
	}
}

func TestList_MultipleSkills(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	names := []string{"alpha", "beta", "gamma"}
	for _, name := range names {
		if _, err := AddSkill(name); err != nil {
			t.Fatalf("AddSkill(%q) error: %v", name, err)
		}
	}

	result, err := List(ListOptions{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(result.Skills) != 3 {
		t.Errorf("Skills = %d, want 3", len(result.Skills))
	}
}

// ── parseFrontmatterDescription: edge cases ───────────────────────────────────

func TestParseFrontmatterDescription_EmptyValue(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "SKILL.md")
	// Bare "description:" with no value is a null scalar, not a block scalar.
	// Subsequent lines should NOT be treated as multi-line description.
	content := `---
name: my-skill
description:
name2: should-not-be-description
---

## Instructions
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got := parseFrontmatterDescription(path)
	if got != "" {
		t.Errorf("parseFrontmatterDescription() = %q, want empty string for bare 'description:'", got)
	}
}

func TestParseFrontmatterDescription_LiteralBlockScalar(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "SKILL.md")
	content := `---
name: my-skill
description: |
  Literal block description.
  Second line.
---

## Instructions
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got := parseFrontmatterDescription(path)
	want := "Literal block description. Second line."
	if got != want {
		t.Errorf("parseFrontmatterDescription() = %q, want %q", got, want)
	}
}

// ── parseMarkdownDescription: frontmatter tracking ────────────────────────────

func TestParseMarkdownDescription_SkipsFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "cmd.md")
	// YAML frontmatter followed by real content — frontmatter keys must be skipped.
	content := `---
name: deploy
---

Deploy to production.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got := parseMarkdownDescription(path)
	want := "Deploy to production."
	if got != want {
		t.Errorf("parseMarkdownDescription() = %q, want %q", got, want)
	}
}

func TestParseMarkdownDescription_ThematicBreakAfterFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "cmd.md")
	// A thematic break (---) inside the body must NOT re-enter frontmatter mode.
	content := `---
name: deploy
---

Deploy to production.

---

More content here.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got := parseMarkdownDescription(path)
	want := "Deploy to production."
	if got != want {
		t.Errorf("parseMarkdownDescription() = %q, want %q", got, want)
	}
}

// ── List: uninitialized store ─────────────────────────────────────────────────

func TestList_UninitializedStore(t *testing.T) {
	setupEnv(t) // sets XDG dirs but does NOT call Init()

	result, err := List(ListOptions{})
	if err != nil {
		t.Fatalf("List() error on uninitialized store: %v", err)
	}

	// All slices should be nil/empty — no directories exist.
	if len(result.Skills) != 0 {
		t.Errorf("Skills = %d, want 0", len(result.Skills))
	}
	if len(result.Commands) != 0 {
		t.Errorf("Commands = %d, want 0", len(result.Commands))
	}
	if len(result.Agents) != 0 {
		t.Errorf("Agents = %d, want 0", len(result.Agents))
	}
	if len(result.Rules) != 0 {
		t.Errorf("Rules = %d, want 0", len(result.Rules))
	}
	if len(result.Instructions) != 0 {
		t.Errorf("Instructions = %d, want 0", len(result.Instructions))
	}
	if len(result.Settings) != 0 {
		t.Errorf("Settings = %d, want 0", len(result.Settings))
	}
}

func TestList_UnknownTypeReturnsError(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	_, err := List(ListOptions{Type: "bogus"})
	if err == nil {
		t.Error("List() with unknown type = nil, want error")
	}
}

// ── Integration: add skill → list → verify ────────────────────────────────────

func TestIntegration_AddSkillThenList(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	// 1. Add skill.
	addResult, err := AddSkill("test-runner")
	if err != nil {
		t.Fatalf("AddSkill() error: %v", err)
	}

	// Verify structure exists.
	for _, sub := range []string{"", "scripts", "references", "assets"} {
		p := filepath.Join(addResult.Dir, sub)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected path %q to exist: %v", p, err)
		}
	}

	// 2. List — skill should appear.
	listResult, err := List(ListOptions{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(listResult.Skills) != 1 {
		t.Fatalf("Skills = %d, want 1", len(listResult.Skills))
	}
	if listResult.Skills[0].Name != "test-runner" {
		t.Errorf("Skills[0].Name = %q, want %q", listResult.Skills[0].Name, "test-runner")
	}
}
