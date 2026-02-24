package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── ValidateName ──────────────────────────────────────────────────────────────

func TestValidateName_Valid(t *testing.T) {
	valid := []string{
		"a",
		"my-skill",
		"code-review",
		"test-runner-123",
		"abc",
		"skill1",
		strings.Repeat("a", 64),
	}
	for _, name := range valid {
		t.Run(name, func(t *testing.T) {
			if err := ValidateName(name); err != nil {
				t.Errorf("ValidateName(%q) = %v, want nil", name, err)
			}
		})
	}
}

func TestValidateName_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"-bad",
		"bad-",
		"Bad",
		"BAD",
		"bad name",
		"bad.name",
		"bad/name",
		strings.Repeat("a", 65),
		"123abc",
		"_bad",
	}
	for _, name := range invalid {
		t.Run(name, func(t *testing.T) {
			if err := ValidateName(name); err == nil {
				t.Errorf("ValidateName(%q) = nil, want error", name)
			}
		})
	}
}

// ── AddSkill ──────────────────────────────────────────────────────────────────

func TestAddSkill_CreatesDirectoryStructure(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	result, err := AddSkill("my-skill")
	if err != nil {
		t.Fatalf("AddSkill() error: %v", err)
	}

	// Verify directory structure.
	for _, sub := range []string{"", "scripts", "references", "assets"} {
		p := filepath.Join(result.Dir, sub)
		info, statErr := os.Stat(p)
		if statErr != nil {
			t.Errorf("missing path %q: %v", p, statErr)
			continue
		}
		if !info.IsDir() {
			t.Errorf("path %q is not a directory", p)
		}
	}
}

func TestAddSkill_CreatesSKILLMD(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	result, err := AddSkill("my-skill")
	if err != nil {
		t.Fatalf("AddSkill() error: %v", err)
	}

	data, err := os.ReadFile(result.SKILLMD)
	if err != nil {
		t.Fatalf("SKILL.md missing: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "name: my-skill") {
		t.Errorf("SKILL.md does not contain 'name: my-skill', got:\n%s", content)
	}
	if !strings.Contains(content, "description:") {
		t.Errorf("SKILL.md does not contain 'description:', got:\n%s", content)
	}
	if !strings.Contains(content, "## Instructions") {
		t.Errorf("SKILL.md does not contain '## Instructions', got:\n%s", content)
	}
}

func TestAddSkill_UnderSkillsDir(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	result, err := AddSkill("my-skill")
	if err != nil {
		t.Fatalf("AddSkill() error: %v", err)
	}

	expectedDir := filepath.Join(Dir(), "skills", "my-skill")
	if result.Dir != expectedDir {
		t.Errorf("Dir = %q, want %q", result.Dir, expectedDir)
	}
}

func TestAddSkill_DuplicateRejected(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	if _, err := AddSkill("my-skill"); err != nil {
		t.Fatalf("first AddSkill() error: %v", err)
	}

	_, err := AddSkill("my-skill")
	if err == nil {
		t.Error("second AddSkill() = nil, want error for duplicate")
	}
}

func TestAddSkill_InvalidNameRejected(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	_, err := AddSkill("Bad Name")
	if err == nil {
		t.Error("AddSkill('Bad Name') = nil, want error")
	}
}

// ── AddCommand ────────────────────────────────────────────────────────────────

func TestAddCommand_CreatesFile(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	result, err := AddCommand("deploy")
	if err != nil {
		t.Fatalf("AddCommand() error: %v", err)
	}

	if _, err := os.Stat(result.File); err != nil {
		t.Errorf("command file missing: %v", err)
	}
}

func TestAddCommand_FileLocation(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	result, err := AddCommand("deploy")
	if err != nil {
		t.Fatalf("AddCommand() error: %v", err)
	}

	expected := filepath.Join(Dir(), "commands", "deploy.md")
	if result.File != expected {
		t.Errorf("File = %q, want %q", result.File, expected)
	}
}

func TestAddCommand_FileContent(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	result, err := AddCommand("deploy")
	if err != nil {
		t.Fatalf("AddCommand() error: %v", err)
	}

	data, err := os.ReadFile(result.File)
	if err != nil {
		t.Fatalf("reading command file: %v", err)
	}

	if !strings.Contains(string(data), "deploy") {
		t.Errorf("command file does not mention 'deploy', got:\n%s", string(data))
	}
}

func TestAddCommand_DuplicateRejected(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	if _, err := AddCommand("deploy"); err != nil {
		t.Fatalf("first AddCommand() error: %v", err)
	}

	_, err := AddCommand("deploy")
	if err == nil {
		t.Error("second AddCommand() = nil, want error for duplicate")
	}
}

// ── AddAgent ──────────────────────────────────────────────────────────────────

func TestAddAgent_CreatesFile(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	result, err := AddAgent("reviewer")
	if err != nil {
		t.Fatalf("AddAgent() error: %v", err)
	}

	expected := filepath.Join(Dir(), "agents", "reviewer.md")
	if result.File != expected {
		t.Errorf("File = %q, want %q", result.File, expected)
	}

	if _, err := os.Stat(result.File); err != nil {
		t.Errorf("agent file missing: %v", err)
	}
}

func TestAddAgent_DuplicateRejected(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	if _, err := AddAgent("reviewer"); err != nil {
		t.Fatalf("first AddAgent() error: %v", err)
	}

	_, err := AddAgent("reviewer")
	if err == nil {
		t.Error("second AddAgent() = nil, want error for duplicate")
	}
}

// ── AddRule ───────────────────────────────────────────────────────────────────

func TestAddRule_CreatesFile(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	result, err := AddRule("style")
	if err != nil {
		t.Fatalf("AddRule() error: %v", err)
	}

	expected := filepath.Join(Dir(), "rules", "style.md")
	if result.File != expected {
		t.Errorf("File = %q, want %q", result.File, expected)
	}

	if _, err := os.Stat(result.File); err != nil {
		t.Errorf("rule file missing: %v", err)
	}
}

func TestAddRule_DuplicateRejected(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	if _, err := AddRule("style"); err != nil {
		t.Fatalf("first AddRule() error: %v", err)
	}

	_, err := AddRule("style")
	if err == nil {
		t.Error("second AddRule() = nil, want error for duplicate")
	}
}
