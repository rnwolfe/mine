// Package contrib orchestrates AI-assisted contribution workflows for GitHub repos.
// It handles fork/clone orchestration, issue selection, and workspace setup.
package contrib

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// MineRepo is the canonical mine repository slug.
const MineRepo = "rnwolfe/mine"

// repoPattern validates "owner/name" GitHub repo slugs.
var repoPattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`)

// Issue represents a GitHub issue candidate for contribution.
type Issue struct {
	Number    int    `json:"number"`
	IssueTitle string `json:"title"`
	Body      string `json:"body"`
	Labels    []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

// FilterValue implements tui.Item for fuzzy matching.
func (i Issue) FilterValue() string {
	return fmt.Sprintf("#%d %s", i.Number, i.IssueTitle)
}

// Title implements tui.Item.
func (i Issue) Title() string {
	return fmt.Sprintf("#%d  %s", i.Number, i.IssueTitle)
}

// Description implements tui.Item — shows labels.
func (i Issue) Description() string {
	labels := make([]string, 0, len(i.Labels))
	for _, l := range i.Labels {
		labels = append(labels, l.Name)
	}
	if len(labels) == 0 {
		return ""
	}
	return strings.Join(labels, ", ")
}

// execCommand is injectable for tests.
var execCommand = exec.Command

// lookPath is injectable for tests.
var lookPath = exec.LookPath

// CheckGH verifies that the gh CLI is installed and authenticated.
func CheckGH() error {
	if _, err := lookPath("gh"); err != nil {
		return errors.New("gh CLI not found — install it from https://cli.github.com")
	}
	cmd := execCommand("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		return errors.New("gh is not authenticated — run `gh auth login` first")
	}
	return nil
}

// ValidateRepo checks that a repo slug is in "owner/name" format.
func ValidateRepo(repo string) error {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return errors.New("repo is required — use --repo owner/name")
	}
	if !repoPattern.MatchString(repo) {
		return fmt.Errorf("invalid repo %q — expected format: owner/name", repo)
	}
	return nil
}

// FetchCandidateIssues fetches open issues from the repo, preferring
// those with the "agent-ready" label. Returns all open issues if no
// agent-ready issues exist. The bool indicates whether agent-ready issues were found.
func FetchCandidateIssues(repo string) ([]Issue, bool, error) {
	// Try agent-ready issues first.
	agentReady, err := fetchIssues(repo, "agent-ready")
	if err != nil {
		return nil, false, err
	}
	if len(agentReady) > 0 {
		return agentReady, true, nil
	}

	// Fall back to all open issues.
	all, err := fetchIssues(repo, "")
	if err != nil {
		return nil, false, err
	}
	return all, false, nil
}

// FetchIssue retrieves a single issue by number.
func FetchIssue(repo string, number int) (*Issue, error) {
	args := []string{
		"issue", "view", fmt.Sprintf("%d", number),
		"--repo", repo,
		"--json", "number,title,body,labels",
	}
	cmd := execCommand("gh", args...)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("issue #%d not found in %s: %s",
				number, repo, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("fetching issue #%d: %w", number, err)
	}

	var issue Issue
	if err := json.Unmarshal(out, &issue); err != nil {
		return nil, fmt.Errorf("parsing issue response: %w", err)
	}
	return &issue, nil
}

// fetchIssues fetches open issues from a repo, optionally filtered by label.
func fetchIssues(repo, label string) ([]Issue, error) {
	args := []string{
		"issue", "list",
		"--repo", repo,
		"--state", "open",
		"--json", "number,title,body,labels",
		"--limit", "50",
	}
	if label != "" {
		args = append(args, "--label", label)
	}

	cmd := execCommand("gh", args...)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("gh issue list failed: %s",
				strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("gh issue list failed: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("parsing issues response: %w", err)
	}
	return issues, nil
}

// ForkState describes the current fork/clone situation for a repo.
type ForkState struct {
	ForkExists  bool   // GitHub fork already exists
	CloneExists bool   // Local clone directory already exists
	CloneDir    string // Expected local clone directory
	ForkSlug    string // "user/repo-name" of the fork
}

// CheckForkState inspects the fork and clone state for a given repo.
func CheckForkState(repo string) (ForkState, error) {
	state := ForkState{}

	// Derive expected local clone directory from repo name.
	parts := strings.Split(repo, "/")
	repoName := parts[len(parts)-1]
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	state.CloneDir = filepath.Join(cwd, repoName)

	// Check if local clone dir exists.
	if _, err := os.Stat(state.CloneDir); err == nil {
		state.CloneExists = true
	}

	// Check if a fork exists by asking gh.
	forkSlug, err := findFork(repo)
	if err == nil && forkSlug != "" {
		state.ForkExists = true
		state.ForkSlug = forkSlug
	}

	return state, nil
}

// findFork asks gh for the authenticated user's fork of repo, if any.
func findFork(repo string) (string, error) {
	// Get current user login.
	userCmd := execCommand("gh", "api", "user", "--jq", ".login")
	userOut, err := userCmd.Output()
	if err != nil {
		// Non-fatal: if we can't get the user, we can't identify forks.
		return "", nil //nolint:nilerr
	}
	login := strings.TrimSpace(string(userOut))

	// List forks of the repo and find one owned by the current user.
	cmd := execCommand("gh", "api",
		fmt.Sprintf("repos/%s/forks?per_page=100", repo),
		"--jq", ".[].full_name",
	)
	out, err := cmd.Output()
	if err != nil {
		// Fork list failure is non-fatal — assume no fork.
		return "", nil //nolint:nilerr
	}

	// Find a fork owned by the current user.
	forks := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, fork := range forks {
		fork = strings.TrimSpace(fork)
		if strings.HasPrefix(fork, login+"/") {
			return fork, nil
		}
	}
	return "", nil
}

// EnsureFork creates a GitHub fork of the repo if one doesn't exist.
// Returns the fork slug (owner/repo).
func EnsureFork(repo string) (string, error) {
	state, err := CheckForkState(repo)
	if err != nil {
		return "", err
	}

	if state.ForkExists {
		return state.ForkSlug, nil
	}

	// Create the fork.
	cmd := execCommand("gh", "repo", "fork", repo, "--clone=false")
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("forking %s: %s", repo, strings.TrimSpace(string(out)))
	}

	// Re-check to get the fork slug.
	state, err = CheckForkState(repo)
	if err != nil {
		return "", err
	}
	return state.ForkSlug, nil
}

// CloneRepo clones the fork (or upstream if no fork) into the local clone dir.
// Returns the clone directory path.
func CloneRepo(repo, forkSlug string) (string, error) {
	parts := strings.Split(repo, "/")
	repoName := parts[len(parts)-1]
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	cloneDir := filepath.Join(cwd, repoName)

	// Sanitize the clone directory path.
	if err := validateCloneDir(cloneDir); err != nil {
		return "", err
	}

	// Choose what to clone.
	cloneTarget := repo
	if forkSlug != "" {
		cloneTarget = forkSlug
	}

	cmd := execCommand("gh", "repo", "clone", cloneTarget, cloneDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("cloning %s: %s", cloneTarget, strings.TrimSpace(string(out)))
	}

	// If we cloned the fork, add upstream remote.
	if forkSlug != "" {
		upstreamURL := fmt.Sprintf("https://github.com/%s.git", repo)
		addCmd := execCommand("git", "-C", cloneDir, "remote", "add", "upstream", upstreamURL)
		// Non-fatal: upstream remote add failure won't block contribution.
		_ = addCmd.Run()
	}

	return cloneDir, nil
}

// validateCloneDir checks that the clone directory path is safe.
func validateCloneDir(dir string) error {
	clean := filepath.Clean(dir)
	// Reject paths that escape the current working directory via "..".
	if strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return fmt.Errorf("unsafe clone directory path: %s", dir)
	}
	return nil
}

// BranchName returns a suggested branch name for working on an issue.
func BranchName(issueNumber int, title string) string {
	// Normalize title: lowercase, replace non-alphanumeric chars with dashes.
	slug := strings.ToLower(title)
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 40 {
		slug = slug[:40]
		slug = strings.TrimRight(slug, "-")
	}
	return fmt.Sprintf("issue-%d-%s", issueNumber, slug)
}

// CreateBranch creates and checks out a new branch in the clone directory.
func CreateBranch(cloneDir, branch string) error {
	cmd := execCommand("git", "-C", cloneDir, "checkout", "-b", branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("creating branch %q: %s", branch, strings.TrimSpace(string(out)))
	}
	return nil
}

// WorkspaceInfo holds everything needed to describe a contribution workspace.
type WorkspaceInfo struct {
	Repo     string
	Issue    *Issue
	ForkSlug string
	CloneDir string
	Branch   string
}
