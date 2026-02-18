package git

import (
	"fmt"
	"strings"
	"testing"
)

// --- parseConventionalType ---

func TestParseConventionalType(t *testing.T) {
	cases := []struct {
		msg      string
		wantType string
	}{
		{"feat: add user auth", "feat"},
		{"fix(api): handle nil pointer", "fix"},
		{"docs: update README", "docs"},
		{"chore: bump deps", "chore"},
		{"refactor: extract helper", "refactor"},
		{"test: add unit tests", "test"},
		{"ci: fix workflow", "ci"},
		{"style: format code", "style"},
		{"perf: optimize loop", "perf"},
		{"feat(scope): with scope", "feat"},
		{"feat!: breaking change", "feat"},
		{"random commit message", "other"},
		{"", "other"},
		{"  ", "other"},
		{"just a message", "other"},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			got := parseConventionalType(c.msg)
			if got != c.wantType {
				t.Errorf("parseConventionalType(%q) = %q, want %q", c.msg, got, c.wantType)
			}
		})
	}
}

// --- branchToTitle ---

func TestBranchToTitle(t *testing.T) {
	cases := []struct {
		branch string
		want   string
	}{
		{"feat/add-user-auth", "feat: add user auth"},
		{"fix/null-pointer-crash", "fix: null pointer crash"},
		{"chore/bump-deps", "chore: bump deps"},
		{"docs/update-readme", "docs: update readme"},
		{"refactor/extract-helper", "refactor: extract helper"},
		{"test/add-unit-tests", "test: add unit tests"},
		{"main", "main"},
		{"my-feature-branch", "my feature branch"},
		{"my_feature_branch", "my feature branch"},
	}

	for _, c := range cases {
		t.Run(c.branch, func(t *testing.T) {
			got := branchToTitle(c.branch)
			if got != c.want {
				t.Errorf("branchToTitle(%q) = %q, want %q", c.branch, got, c.want)
			}
		})
	}
}

// --- Changelog ---

func TestChangelog(t *testing.T) {
	// Override runGit to return fake commits.
	origRunGit := runGit
	defer func() { runGit = origRunGit }()

	runGit = func(args ...string) (string, error) {
		// Simulate log output between two refs.
		if len(args) >= 1 && args[0] == "log" {
			return strings.Join([]string{
				"feat: add login page",
				"fix(api): handle 404 error",
				"docs: update contributing guide",
				"chore: bump go version",
				"random thing without type",
			}, "\n"), nil
		}
		return "", nil
	}

	cl, err := Changelog("v1.0.0", "HEAD")
	if err != nil {
		t.Fatalf("Changelog() error: %v", err)
	}

	// Should contain the section headings.
	if !strings.Contains(cl, "### Features") {
		t.Error("expected Features section")
	}
	if !strings.Contains(cl, "### Bug Fixes") {
		t.Error("expected Bug Fixes section")
	}
	if !strings.Contains(cl, "### Documentation") {
		t.Error("expected Documentation section")
	}
	if !strings.Contains(cl, "### Chores") {
		t.Error("expected Chores section")
	}
	if !strings.Contains(cl, "### Other") {
		t.Error("expected Other section")
	}

	// Should contain commit messages.
	if !strings.Contains(cl, "feat: add login page") {
		t.Error("expected feat commit in changelog")
	}
	if !strings.Contains(cl, "fix(api): handle 404 error") {
		t.Error("expected fix commit in changelog")
	}
}

func TestChangelogEmpty(t *testing.T) {
	origRunGit := runGit
	defer func() { runGit = origRunGit }()

	runGit = func(args ...string) (string, error) {
		return "", nil // no commits
	}

	cl, err := Changelog("v1.0.0", "HEAD")
	if err != nil {
		t.Fatalf("Changelog() error: %v", err)
	}

	// Should still have the header.
	if !strings.Contains(cl, "## Changelog") {
		t.Error("expected Changelog header even with no commits")
	}
}

// --- IsWipCommit ---

func TestIsWipCommit(t *testing.T) {
	origRunGit := runGit
	defer func() { runGit = origRunGit }()

	t.Run("is wip", func(t *testing.T) {
		runGit = func(args ...string) (string, error) {
			return "wip", nil
		}
		ok, err := IsWipCommit()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Error("expected true for wip commit")
		}
	})

	t.Run("is wip uppercase", func(t *testing.T) {
		runGit = func(args ...string) (string, error) {
			return "WIP", nil
		}
		ok, err := IsWipCommit()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Error("expected true for WIP commit")
		}
	})

	t.Run("not wip", func(t *testing.T) {
		runGit = func(args ...string) (string, error) {
			return "feat: add something", nil
		}
		ok, err := IsWipCommit()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Error("expected false for non-wip commit")
		}
	})

	t.Run("git error", func(t *testing.T) {
		runGit = func(args ...string) (string, error) {
			return "", fmt.Errorf("not a git repository")
		}
		_, err := IsWipCommit()
		if err == nil {
			t.Error("expected error when git fails")
		}
	})
}

// --- MergedBranches sweep logic ---

func TestMergedBranchesFiltersProtected(t *testing.T) {
	origRunGit := runGit
	defer func() { runGit = origRunGit }()

	callCount := 0
	runGit = func(args ...string) (string, error) {
		callCount++
		if len(args) >= 2 && args[0] == "branch" && args[1] == "--merged" {
			// Simulate merged branches output (with * for current).
			return "* main\n  feature-a\n  feature-b\n  develop", nil
		}
		if len(args) >= 2 && args[0] == "rev-parse" && args[1] == "--abbrev-ref" {
			return "main", nil
		}
		return "", nil
	}

	branches, err := MergedBranches()
	if err != nil {
		t.Fatalf("MergedBranches() error: %v", err)
	}

	// main, develop should be filtered; feature-a and feature-b should remain.
	names := make(map[string]bool)
	for _, b := range branches {
		names[b.Name] = true
	}

	if names["main"] {
		t.Error("main should be filtered out")
	}
	if names["develop"] {
		t.Error("develop should be filtered out")
	}
	if !names["feature-a"] {
		t.Error("feature-a should be included")
	}
	if !names["feature-b"] {
		t.Error("feature-b should be included")
	}
}

// --- GitAliases ---

func TestGitAliases(t *testing.T) {
	aliases := GitAliases()
	if len(aliases) == 0 {
		t.Fatal("expected at least one alias")
	}

	names := make(map[string]bool)
	for _, a := range aliases {
		if a.Name == "" {
			t.Error("alias with empty name")
		}
		if a.Value == "" {
			t.Error("alias with empty value")
		}
		if a.Desc == "" {
			t.Error("alias with empty desc")
		}
		if names[a.Name] {
			t.Errorf("duplicate alias name: %s", a.Name)
		}
		names[a.Name] = true
	}

	// Check some expected aliases.
	expected := []string{"co", "br", "st", "lg"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected alias %q to be present", name)
		}
	}
}

// --- ListBranches ---

func TestListBranches(t *testing.T) {
	origRunGit := runGit
	defer func() { runGit = origRunGit }()

	runGit = func(args ...string) (string, error) {
		if len(args) >= 1 && args[0] == "branch" {
			return "main\nfeature-a\nfeature-b", nil
		}
		if len(args) >= 2 && args[0] == "rev-parse" && args[1] == "--abbrev-ref" {
			return "main", nil
		}
		return "", nil
	}

	branches, err := ListBranches()
	if err != nil {
		t.Fatalf("ListBranches() error: %v", err)
	}
	if len(branches) != 3 {
		t.Fatalf("expected 3 branches, got %d", len(branches))
	}

	// Current branch should be marked.
	var currentCount int
	for _, b := range branches {
		if b.Current {
			currentCount++
			if b.Name != "main" {
				t.Errorf("unexpected current branch: %s", b.Name)
			}
		}
	}
	if currentCount != 1 {
		t.Errorf("expected exactly 1 current branch, got %d", currentCount)
	}
}

// --- Branch tui.Item interface ---

func TestBranchItem(t *testing.T) {
	b := Branch{Name: "main", Current: true}
	if b.FilterValue() != "main" {
		t.Errorf("FilterValue = %q, want %q", b.FilterValue(), "main")
	}
	if b.Title() != "main *" {
		t.Errorf("Title() = %q, want %q", b.Title(), "main *")
	}
	if b.Description() != "current branch" {
		t.Errorf("Description() = %q, want %q", b.Description(), "current branch")
	}

	b2 := Branch{Name: "feature-x", Current: false}
	if b2.Title() != "feature-x" {
		t.Errorf("Title() = %q, want %q", b2.Title(), "feature-x")
	}
	if b2.Description() != "" {
		t.Errorf("Description() = %q, want empty", b2.Description())
	}
}

// --- BuildPRInfo ---

func TestBuildPRInfo(t *testing.T) {
	origRunGit := runGit
	defer func() { runGit = origRunGit }()

	runGit = func(args ...string) (string, error) {
		// CurrentBranch
		if len(args) >= 2 && args[0] == "rev-parse" && args[1] == "--abbrev-ref" {
			return "feat/add-oauth", nil
		}
		// DefaultBase — verify main exists
		if len(args) >= 2 && args[0] == "rev-parse" && args[1] == "--verify" {
			if args[2] == "main" {
				return "abc123", nil
			}
			return "", fmt.Errorf("not found")
		}
		// CommitsBetween
		if len(args) >= 1 && args[0] == "log" {
			return "feat: add google oauth\nfeat: add github oauth", nil
		}
		return "", nil
	}

	info, err := BuildPRInfo()
	if err != nil {
		t.Fatalf("BuildPRInfo() error: %v", err)
	}

	if info.Branch != "feat/add-oauth" {
		t.Errorf("Branch = %q, want %q", info.Branch, "feat/add-oauth")
	}
	if info.Base != "main" {
		t.Errorf("Base = %q, want %q", info.Base, "main")
	}
	if !strings.Contains(info.Title, "feat") {
		t.Errorf("Title %q should contain 'feat'", info.Title)
	}
	if !strings.Contains(info.Body, "## Summary") {
		t.Error("Body should contain ## Summary")
	}
}
