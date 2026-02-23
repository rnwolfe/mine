package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rnwolfe/mine/internal/gitutil"
)

// LinkHealthState represents the health of a single link entry.
type LinkHealthState string

const (
	// LinkHealthLinked means the symlink exists and points to the canonical store,
	// or copy mode and content matches.
	LinkHealthLinked LinkHealthState = "linked"

	// LinkHealthBroken means a symlink exists but its target is missing (dangling).
	LinkHealthBroken LinkHealthState = "broken"

	// LinkHealthReplaced means the path exists as a regular file/dir where a symlink
	// was expected (someone replaced the symlink manually).
	LinkHealthReplaced LinkHealthState = "replaced"

	// LinkHealthUnlinked means the target path does not exist.
	LinkHealthUnlinked LinkHealthState = "unlinked"

	// LinkHealthDiverged means copy mode and the content differs from the canonical store.
	LinkHealthDiverged LinkHealthState = "diverged"
)

// LinkHealth pairs a manifest entry with its computed health state.
type LinkHealth struct {
	Entry   LinkEntry
	State   LinkHealthState
	Message string // optional extra context (e.g. symlink destination)
}

// StoreInfo contains metadata about the canonical agents store git repo.
type StoreInfo struct {
	Dir              string
	CommitCount      int
	RemoteURL        string
	UnpushedCommits  int
	UncommittedFiles int
}

// StatusResult holds the complete status report for the agents store.
type StatusResult struct {
	Store  StoreInfo
	Agents []Agent
	Links  []LinkHealth
}

// CheckStatus assembles a full status report by re-detecting agents and evaluating
// link health for every entry in the manifest.
func CheckStatus() (*StatusResult, error) {
	m, err := ReadManifest()
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	store, err := queryStoreInfo()
	if err != nil {
		return nil, fmt.Errorf("querying store: %w", err)
	}

	detected := DetectAgents()

	storeDir := Dir()
	links := make([]LinkHealth, 0, len(m.Links))
	for _, entry := range m.Links {
		h := CheckLinkHealth(entry, storeDir)
		links = append(links, h)
	}

	return &StatusResult{
		Store:  store,
		Agents: detected,
		Links:  links,
	}, nil
}

// CheckLinkHealth evaluates the health of a single link entry.
// storeDir is the canonical agents store directory (used to resolve source paths).
func CheckLinkHealth(entry LinkEntry, storeDir string) LinkHealth {
	h := LinkHealth{Entry: entry}

	sourcePath := filepath.Join(storeDir, entry.Source)

	info, err := os.Lstat(entry.Target)
	if err != nil {
		if os.IsNotExist(err) {
			h.State = LinkHealthUnlinked
		} else {
			h.State = LinkHealthUnlinked
			h.Message = err.Error()
		}
		return h
	}

	if entry.Mode == "copy" {
		return checkCopyHealth(h, sourcePath, entry.Target, info)
	}

	return checkSymlinkHealth(h, sourcePath, entry.Target, info)
}

// checkCopyHealth evaluates a copy-mode link entry.
func checkCopyHealth(h LinkHealth, sourcePath, target string, info os.FileInfo) LinkHealth {
	if info.Mode()&os.ModeSymlink != 0 {
		// Someone replaced the copy with a symlink — treat as replaced.
		h.State = LinkHealthReplaced
		if dest, err := os.Readlink(target); err == nil {
			h.Message = fmt.Sprintf("symlink → %s", dest)
		}
		return h
	}

	if contentMatches(sourcePath, target) {
		h.State = LinkHealthLinked
	} else {
		h.State = LinkHealthDiverged
	}
	return h
}

// checkSymlinkHealth evaluates a symlink-mode link entry.
func checkSymlinkHealth(h LinkHealth, sourcePath, target string, info os.FileInfo) LinkHealth {
	if info.Mode()&os.ModeSymlink == 0 {
		// Regular file or directory where we expected a symlink.
		h.State = LinkHealthReplaced
		return h
	}

	dest, err := os.Readlink(target)
	if err != nil {
		h.State = LinkHealthBroken
		h.Message = err.Error()
		return h
	}

	if dest == sourcePath {
		// Symlink points to our canonical store — verify the source still exists.
		if _, statErr := os.Stat(sourcePath); statErr != nil {
			h.State = LinkHealthBroken
			h.Message = fmt.Sprintf("canonical source missing: %s", sourcePath)
		} else {
			h.State = LinkHealthLinked
		}
		return h
	}

	// Symlink points somewhere else.
	if _, statErr := os.Stat(target); statErr != nil {
		// Dangling symlink pointing away from our store.
		h.State = LinkHealthBroken
		h.Message = fmt.Sprintf("points to %s (missing)", dest)
	} else {
		// Valid symlink but pointing to a different location.
		h.State = LinkHealthReplaced
		h.Message = fmt.Sprintf("points to %s", dest)
	}
	return h
}

// queryStoreInfo gathers metadata from the canonical agents store git repo.
// Non-fatal git failures (e.g. no commits yet) are silently ignored.
func queryStoreInfo() (StoreInfo, error) {
	dir := Dir()
	info := StoreInfo{Dir: dir}

	// Commit count — may fail on a brand new repo with no commits.
	if out, err := gitutil.RunCmd(dir, "rev-list", "--count", "HEAD"); err == nil {
		if n, parseErr := strconv.Atoi(strings.TrimSpace(out)); parseErr == nil {
			info.CommitCount = n
		}
	}

	// Remote URL (origin).
	if out, err := gitutil.RunCmd(dir, "remote", "get-url", "origin"); err == nil {
		info.RemoteURL = strings.TrimSpace(out)
	}

	// Unpushed commits — only meaningful when a remote is configured.
	if info.RemoteURL != "" {
		if out, err := gitutil.RunCmd(dir, "rev-list", "--count", "origin/HEAD..HEAD"); err == nil {
			if n, parseErr := strconv.Atoi(strings.TrimSpace(out)); parseErr == nil {
				info.UnpushedCommits = n
			}
		}
	}

	// Uncommitted changes.
	if out, err := gitutil.RunCmd(dir, "status", "--porcelain"); err == nil {
		count := 0
		for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
			if l != "" {
				count++
			}
		}
		info.UncommittedFiles = count
	}

	return info, nil
}

// contentMatches returns true if the content of path a and path b are identical.
// Works for both regular files and directories (recursive comparison).
func contentMatches(a, b string) bool {
	aInfo, err := os.Stat(a)
	if err != nil {
		return false
	}
	bInfo, err := os.Stat(b)
	if err != nil {
		return false
	}

	if aInfo.IsDir() != bInfo.IsDir() {
		return false
	}

	if aInfo.IsDir() {
		return dirContentMatches(a, b)
	}
	return fileContentMatches(a, b)
}

// fileContentMatches compares two regular files byte-for-byte.
func fileContentMatches(a, b string) bool {
	aData, err := os.ReadFile(a)
	if err != nil {
		return false
	}
	bData, err := os.ReadFile(b)
	if err != nil {
		return false
	}
	if len(aData) != len(bData) {
		return false
	}
	for i := range aData {
		if aData[i] != bData[i] {
			return false
		}
	}
	return true
}

// dirContentMatches compares two directories by recursively checking each entry.
// Returns false if the entry lists differ in count, names, or any file content differs.
func dirContentMatches(a, b string) bool {
	aEntries, err := os.ReadDir(a)
	if err != nil {
		return false
	}
	bEntries, err := os.ReadDir(b)
	if err != nil {
		return false
	}
	if len(aEntries) != len(bEntries) {
		return false
	}
	// Build a set of names present in b for explicit name comparison.
	bNames := make(map[string]bool, len(bEntries))
	for _, entry := range bEntries {
		bNames[entry.Name()] = true
	}
	for _, entry := range aEntries {
		if !bNames[entry.Name()] {
			return false
		}
		aPath := filepath.Join(a, entry.Name())
		bPath := filepath.Join(b, entry.Name())
		if !contentMatches(aPath, bPath) {
			return false
		}
	}
	return true
}
