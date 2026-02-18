package contrib

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// --- ValidateRepo ---

func TestValidateRepo(t *testing.T) {
	tests := []struct {
		name    string
		repo    string
		wantErr bool
	}{
		{"valid slug", "owner/repo", false},
		{"valid with dots", "my.org/my-repo.go", false},
		{"valid with numbers", "user123/repo456", false},
		{"mine repo", "rnwolfe/mine", false},
		{"empty", "", true},
		{"whitespace only", "   ", true},
		{"missing slash", "ownerrepo", true},
		{"starts with slash", "/repo", true},
		{"ends with slash", "owner/", true}, // requires non-empty name after slash
		{"double slash", "owner//repo", true},
		{"spaces inside", "owner name/repo", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepo(tt.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRepo(%q) error = %v, wantErr %v", tt.repo, err, tt.wantErr)
			}
		})
	}
}

// --- BranchName ---

func TestBranchName(t *testing.T) {
	tests := []struct {
		number int
		title  string
		want   string
	}{
		{16, "Add community contribution command", "issue-16-add-community-contribution-command"},
		{1, "Fix bug", "issue-1-fix-bug"},
		{42, "Feature: special chars & more!", "issue-42-feature-special-chars-more"},
		{99, "  leading and trailing spaces  ", "issue-99-leading-and-trailing-spaces"},
	}
	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := BranchName(tt.number, tt.title)
			if got != tt.want {
				t.Errorf("BranchName(%d, %q) = %q, want %q", tt.number, tt.title, got, tt.want)
			}
		})
	}
}

func TestBranchName_TruncatesLongTitle(t *testing.T) {
	long := strings.Repeat("a", 60)
	got := BranchName(1, long)
	if len(got) > len("issue-1-")+40 {
		t.Errorf("BranchName did not truncate long title, got: %s", got)
	}
	if !strings.HasPrefix(got, "issue-1-") {
		t.Errorf("BranchName missing issue prefix, got: %s", got)
	}
}

// --- Issue tui.Item interface ---

func TestIssue_TuiItem(t *testing.T) {
	issue := Issue{
		Number:     42,
		IssueTitle: "Implement feature X",
		Labels: []struct {
			Name string `json:"name"`
		}{
			{Name: "feature"},
			{Name: "good-first-issue"},
		},
	}

	if !strings.Contains(issue.FilterValue(), "#42") {
		t.Errorf("FilterValue missing issue number: %s", issue.FilterValue())
	}
	if !strings.Contains(issue.FilterValue(), "Implement feature X") {
		t.Errorf("FilterValue missing title: %s", issue.FilterValue())
	}
	if !strings.Contains(issue.Title(), "#42") {
		t.Errorf("Title missing issue number: %s", issue.Title())
	}
	if !strings.Contains(issue.Title(), "Implement feature X") {
		t.Errorf("Title missing issue title: %s", issue.Title())
	}

	desc := issue.Description()
	if !strings.Contains(desc, "feature") {
		t.Errorf("Description missing label: %s", desc)
	}
	if !strings.Contains(desc, "good-first-issue") {
		t.Errorf("Description missing label: %s", desc)
	}
}

func TestIssue_NoLabels(t *testing.T) {
	issue := Issue{Number: 1, IssueTitle: "Test"}
	if issue.Description() != "" {
		t.Errorf("expected empty description for unlabeled issue, got %q", issue.Description())
	}
}

// --- Issue JSON unmarshaling ---

func TestIssue_JSONUnmarshal(t *testing.T) {
	raw := `{
		"number": 16,
		"title": "Community contribution command",
		"body": "Issue body here",
		"labels": [{"name": "agent-ready"}, {"name": "feature"}]
	}`

	var issue Issue
	if err := json.Unmarshal([]byte(raw), &issue); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if issue.Number != 16 {
		t.Errorf("Number = %d, want 16", issue.Number)
	}
	if issue.IssueTitle != "Community contribution command" {
		t.Errorf("IssueTitle = %q, want %q", issue.IssueTitle, "Community contribution command")
	}
	if len(issue.Labels) != 2 {
		t.Errorf("Labels len = %d, want 2", len(issue.Labels))
	}
	if issue.Labels[0].Name != "agent-ready" {
		t.Errorf("Labels[0] = %q, want %q", issue.Labels[0].Name, "agent-ready")
	}
}

// --- CheckGH ---

func TestCheckGH_NotInstalled(t *testing.T) {
	origLookPath := lookPath
	defer func() { lookPath = origLookPath }()

	lookPath = func(file string) (string, error) {
		return "", exec.ErrNotFound
	}

	err := CheckGH()
	if err == nil {
		t.Fatal("expected error when gh not found")
	}
	if !strings.Contains(err.Error(), "gh CLI not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCheckGH_NotAuthenticated(t *testing.T) {
	origLookPath := lookPath
	origExecCommand := execCommand
	defer func() {
		lookPath = origLookPath
		execCommand = origExecCommand
	}()

	lookPath = func(file string) (string, error) {
		return "/usr/bin/gh", nil
	}
	execCommand = func(name string, args ...string) *exec.Cmd {
		// Use test binary with a nonexistent flag to guarantee non-zero exit.
		return exec.Command(os.Args[0], "-test.run=^$", "-test.nonexistent-flag")
	}

	err := CheckGH()
	if err == nil {
		t.Fatal("expected error when gh not authenticated")
	}
	if !strings.Contains(err.Error(), "not authenticated") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// --- FetchIssue ---

func TestFetchIssue_Success(t *testing.T) {
	orig := execCommand
	defer func() { execCommand = orig }()

	issueJSON := `{"number":16,"title":"Community contribution","body":"","labels":[]}`
	execCommand = func(name string, args ...string) *exec.Cmd {
		// Echo the JSON to stdout with exit 0.
		return exec.Command("sh", "-c", "echo '"+issueJSON+"'")
	}

	issue, err := FetchIssue("rnwolfe/mine", 16)
	if err != nil {
		t.Fatalf("FetchIssue unexpected error: %v", err)
	}
	if issue.Number != 16 {
		t.Errorf("Number = %d, want 16", issue.Number)
	}
	if issue.IssueTitle != "Community contribution" {
		t.Errorf("IssueTitle = %q", issue.IssueTitle)
	}
}

func TestFetchIssue_NotFound(t *testing.T) {
	orig := execCommand
	defer func() { execCommand = orig }()

	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo 'not found' >&2; exit 1")
	}

	_, err := FetchIssue("rnwolfe/mine", 9999)
	if err == nil {
		t.Fatal("expected error for missing issue")
	}
}

// --- fetchIssues (indirectly via FetchCandidateIssues) ---

func TestFetchCandidateIssues_AgentReadyPreferred(t *testing.T) {
	orig := execCommand
	defer func() { execCommand = orig }()

	agentReadyJSON := `[{"number":1,"title":"Agent ready issue","body":"","labels":[{"name":"agent-ready"}]}]`
	callCount := 0

	execCommand = func(name string, args ...string) *exec.Cmd {
		callCount++
		// First call (with --label agent-ready) returns results.
		return exec.Command("sh", "-c", "echo '"+agentReadyJSON+"'")
	}

	issues, agentReady, err := FetchCandidateIssues("rnwolfe/mine")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !agentReady {
		t.Error("expected agentReady = true")
	}
	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}
}

func TestFetchCandidateIssues_FallbackToAll(t *testing.T) {
	orig := execCommand
	defer func() { execCommand = orig }()

	callCount := 0
	allJSON := `[{"number":5,"title":"Open issue","body":"","labels":[]},{"number":6,"title":"Another","body":"","labels":[]}]`

	execCommand = func(name string, args ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			// First call (agent-ready filter) returns empty.
			return exec.Command("sh", "-c", "echo '[]'")
		}
		// Second call returns all issues.
		return exec.Command("sh", "-c", "echo '"+allJSON+"'")
	}

	issues, agentReady, err := FetchCandidateIssues("rnwolfe/mine")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agentReady {
		t.Error("expected agentReady = false when no agent-ready label")
	}
	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(issues))
	}
}

// --- validateCloneDir ---

func TestValidateCloneDir_Safe(t *testing.T) {
	cases := []struct {
		dir     string
		wantErr bool
	}{
		{"/home/user/projects/mine", false},
		{"./mine", false},
		{"/tmp/mine", false},
		// Path-traversal attempts.
		{"../outside", true},
		{"..", true},
	}

	for _, tc := range cases {
		t.Run(tc.dir, func(t *testing.T) {
			err := validateCloneDir(tc.dir)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateCloneDir(%q) error = %v, wantErr %v", tc.dir, err, tc.wantErr)
			}
		})
	}
}

// --- CheckForkState ---

func TestCheckForkState_NoCloneNoFork(t *testing.T) {
	orig := execCommand
	defer func() { execCommand = orig }()

	execCommand = func(name string, args ...string) *exec.Cmd {
		// All gh commands fail (no fork).
		return exec.Command("sh", "-c", "exit 1")
	}

	state, err := CheckForkState("rnwolfe/mine")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.ForkExists {
		t.Error("expected ForkExists = false")
	}
	if state.CloneExists {
		t.Error("expected CloneExists = false")
	}

	// CloneDir should end with the repo name.
	if !strings.HasSuffix(state.CloneDir, "mine") {
		t.Errorf("CloneDir %q should end with 'mine'", state.CloneDir)
	}
}

func TestCheckForkState_ExistingClone(t *testing.T) {
	orig := execCommand
	defer func() { execCommand = orig }()

	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exit 1")
	}

	// Create a temporary directory to simulate an existing clone.
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "mine")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Change working directory to tmpDir so CheckForkState finds it.
	origWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWd) //nolint:errcheck

	state, err := CheckForkState("rnwolfe/mine")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !state.CloneExists {
		t.Error("expected CloneExists = true")
	}
}

// --- MineRepo constant ---

func TestMineRepo(t *testing.T) {
	if MineRepo != "rnwolfe/mine" {
		t.Errorf("MineRepo = %q, want %q", MineRepo, "rnwolfe/mine")
	}
}
