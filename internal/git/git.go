// Package git provides a convenience layer over common git operations.
// It shells out to the git binary — no libgit2 or go-git dependency.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Available reports whether the git binary is in PATH.
func Available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// Branch represents a local git branch.
type Branch struct {
	Name    string
	Current bool
	Merged  bool
}

// FilterValue implements tui.Item.
func (b Branch) FilterValue() string { return b.Name }

// Title implements tui.Item.
func (b Branch) Title() string {
	if b.Current {
		return b.Name + " *"
	}
	return b.Name
}

// Description implements tui.Item.
func (b Branch) Description() string {
	if b.Current {
		return "current branch"
	}
	return ""
}

// runGit executes a git command and returns trimmed stdout.
var runGit = func(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(out.String()), nil
}

// CurrentBranch returns the name of the current branch.
func CurrentBranch() (string, error) {
	return runGit("rev-parse", "--abbrev-ref", "HEAD")
}

// ListBranches returns all local branches, sorted by most recently used.
func ListBranches() ([]Branch, error) {
	// Use reflog ordering so most-recently-checked-out branches appear first.
	out, err := runGit("branch", "--sort=-committerdate", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}

	current, _ := CurrentBranch()

	lines := strings.Split(out, "\n")
	branches := make([]Branch, 0, len(lines))
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		branches = append(branches, Branch{
			Name:    name,
			Current: name == current,
		})
	}
	return branches, nil
}

// SwitchBranch switches to the given branch name.
var SwitchBranch = func(name string) error {
	_, err := runGit("switch", name)
	return err
}

// MergedBranches returns local branches that have been merged into the current branch
// (excluding main/master/develop and the current branch itself).
func MergedBranches() ([]Branch, error) {
	out, err := runGit("branch", "--merged")
	if err != nil {
		return nil, err
	}

	current, _ := CurrentBranch()

	protected := map[string]bool{
		"main":    true,
		"master":  true,
		"develop": true,
		current:   true,
	}

	lines := strings.Split(out, "\n")
	var branches []Branch
	for _, line := range lines {
		name := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if name == "" || protected[name] {
			continue
		}
		branches = append(branches, Branch{Name: name, Merged: true})
	}
	return branches, nil
}

// DeleteBranch deletes a local branch.
var DeleteBranch = func(name string) error {
	_, err := runGit("branch", "-d", name)
	return err
}

// PruneRemote prunes stale remote-tracking branches.
var PruneRemote = func() error {
	_, err := runGit("remote", "prune", "origin")
	return err
}

// LastCommitMessage returns the message of the most recent commit.
func LastCommitMessage() (string, error) {
	return runGit("log", "-1", "--pretty=%s")
}

// UndoLastCommit performs a soft reset of the last commit.
var UndoLastCommit = func() error {
	_, err := runGit("reset", "--soft", "HEAD~1")
	return err
}

// WipCommit stages all changes and creates a "wip" commit.
var WipCommit = func() error {
	if _, err := runGit("add", "-A"); err != nil {
		return err
	}
	_, err := runGit("commit", "-m", "wip")
	return err
}

// IsWipCommit reports whether the last commit message is exactly "wip".
func IsWipCommit() (bool, error) {
	msg, err := LastCommitMessage()
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(msg), "wip"), nil
}

// CommitLog returns the last n commit log lines in a compact pretty format.
func CommitLog(n int) (string, error) {
	return runGit(
		"log",
		fmt.Sprintf("-%d", n),
		"--pretty=format:%C(yellow)%h%Creset %C(bold blue)%an%Creset %s %C(dim green)(%cr)%Creset",
		"--graph",
		"--decorate",
		"--color",
	)
}

// DefaultBase detects the most likely base branch (main, master, or develop).
func DefaultBase() string {
	for _, candidate := range []string{"main", "master", "develop"} {
		if _, err := runGit("rev-parse", "--verify", candidate); err == nil {
			return candidate
		}
	}
	return "main"
}

// CommitsBetween returns the commit subjects between two refs (from..to).
func CommitsBetween(from, to string) ([]string, error) {
	out, err := runGit("log", from+".."+to, "--pretty=format:%s", "--no-merges")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	lines := strings.Split(out, "\n")
	var commits []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			commits = append(commits, l)
		}
	}
	return commits, nil
}

// Changelog generates a Markdown changelog from conventional commits between two refs.
func Changelog(from, to string) (string, error) {
	commits, err := CommitsBetween(from, to)
	if err != nil {
		return "", err
	}

	sections := map[string][]string{
		"feat":     {},
		"fix":      {},
		"docs":     {},
		"refactor": {},
		"chore":    {},
		"other":    {},
	}
	order := []string{"feat", "fix", "docs", "refactor", "chore", "other"}
	headings := map[string]string{
		"feat":     "Features",
		"fix":      "Bug Fixes",
		"docs":     "Documentation",
		"refactor": "Refactoring",
		"chore":    "Chores",
		"other":    "Other",
	}

	for _, c := range commits {
		typ := parseConventionalType(c)
		if _, ok := sections[typ]; ok {
			sections[typ] = append(sections[typ], c)
		} else {
			sections["other"] = append(sections["other"], c)
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Changelog (%s..%s)\n\n", from, to))

	for _, key := range order {
		items := sections[key]
		if len(items) == 0 {
			continue
		}
		sb.WriteString("### " + headings[key] + "\n\n")
		for _, item := range items {
			sb.WriteString("- " + item + "\n")
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// parseConventionalType extracts the conventional commit type from a message.
func parseConventionalType(msg string) string {
	// Matches: feat(scope): ..., fix: ..., docs!: ...
	idx := strings.IndexAny(msg, ":(!")
	if idx <= 0 {
		return "other"
	}
	typ := strings.TrimSpace(msg[:idx])
	// Strip scope suffix: feat(api) → feat
	if p := strings.Index(typ, "("); p >= 0 {
		typ = typ[:p]
	}
	typ = strings.ToLower(strings.TrimSpace(typ))
	switch typ {
	case "feat", "fix", "docs", "refactor", "chore", "test", "ci", "style", "perf":
		return typ
	}
	return "other"
}

// GitAliases returns the opinionated git aliases to install.
func GitAliases() []GitAlias {
	return []GitAlias{
		{Name: "co", Value: "checkout", Desc: "Shortcut for checkout"},
		{Name: "br", Value: "branch", Desc: "Shortcut for branch"},
		{Name: "st", Value: "status -sb", Desc: "Compact status"},
		{Name: "lg", Value: "log --oneline --graph --decorate --all", Desc: "Pretty graph log"},
		{Name: "last", Value: "log -1 HEAD --stat", Desc: "Show last commit with stats"},
		{Name: "unstage", Value: "reset HEAD --", Desc: "Unstage a file"},
		{Name: "undo", Value: "reset --soft HEAD~1", Desc: "Soft undo last commit"},
		{Name: "wip", Value: `!git add -A && git commit -m "wip"`, Desc: "Quick WIP commit"},
		{Name: "aliases", Value: "config --get-regexp alias", Desc: "List all aliases"},
	}
}

// GitAlias represents a git alias entry.
type GitAlias struct {
	Name  string
	Value string
	Desc  string
}

// InstallAlias sets a git alias in the global config.
var InstallAlias = func(alias GitAlias) error {
	_, err := runGit("config", "--global", "alias."+alias.Name, alias.Value)
	return err
}

// HasGhCLI reports whether the gh CLI is available.
func HasGhCLI() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

// PRInfo holds information needed to create a pull request.
type PRInfo struct {
	Title  string
	Body   string
	Base   string
	Branch string
}

// BuildPRInfo generates PR title and body from the current branch and commits.
func BuildPRInfo() (*PRInfo, error) {
	branch, err := CurrentBranch()
	if err != nil {
		return nil, err
	}

	base := DefaultBase()

	// Generate title from branch name.
	title := branchToTitle(branch)

	// Generate body from commits.
	commits, err := CommitsBetween(base, branch)
	if err != nil {
		commits = nil
	}

	var body strings.Builder
	body.WriteString("## Summary\n\n")
	if len(commits) > 0 {
		for _, c := range commits {
			body.WriteString("- " + c + "\n")
		}
	} else {
		body.WriteString("_No commits yet._\n")
	}
	body.WriteString("\n## Test Plan\n\n- [ ] Manual testing\n")

	return &PRInfo{
		Title:  title,
		Body:   body.String(),
		Base:   base,
		Branch: branch,
	}, nil
}

// branchToTitle converts a branch name to a human-readable PR title.
// e.g. "feat/add-user-auth" → "feat: add user auth"
func branchToTitle(branch string) string {
	// Remove common prefixes and clean up.
	prefixes := []string{"feat/", "fix/", "chore/", "docs/", "refactor/", "test/"}
	title := branch
	for _, pfx := range prefixes {
		if strings.HasPrefix(title, pfx) {
			typ := strings.TrimSuffix(pfx, "/")
			rest := strings.TrimPrefix(title, pfx)
			rest = strings.ReplaceAll(rest, "-", " ")
			rest = strings.ReplaceAll(rest, "_", " ")
			return typ + ": " + rest
		}
	}
	title = strings.ReplaceAll(title, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")
	return title
}
