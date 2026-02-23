package agents

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rnwolfe/mine/internal/gitutil"
)

// AdoptOptions controls the behavior of the Adopt operation.
type AdoptOptions struct {
	Agent  string // filter to a single agent name; empty means all detected agents
	DryRun bool   // show what would be imported without making changes
	Copy   bool   // import files into store but don't replace originals with symlinks
}

// AdoptItem describes a single file or directory that can be adopted.
type AdoptItem struct {
	Agent      string // which agent this came from
	SourcePath string // absolute path of the file in the agent's config dir
	StoreRel   string // relative path within the canonical store
	StoreAbs   string // absolute path within the canonical store
	Kind       string // "instruction", "skills", "commands", "settings", "mcp", "agents", "rules"
	Conflict   bool   // the store already has different content for this item
	Status     string // "imported", "skipped", "conflict", "already-managed"
	Err        error  // non-nil if the operation failed
}

// Adopt scans detected agents for existing configs, imports them into the
// canonical store, and optionally replaces originals with symlinks.
//
// Workflow:
//  1. Detect which agents have content to adopt
//  2. Scan each agent's config directory for adoptable items
//  3. Copy items into the canonical store (first agent's instruction file wins;
//     subsequent agents with differing content are flagged as conflicts)
//  4. Replace original files with symlinks to the store (unless --copy)
//  5. Auto-commit the imported content to the store's git history
func Adopt(opts AdoptOptions) ([]AdoptItem, error) {
	if !IsInitialized() {
		return nil, fmt.Errorf("agents store not initialized — run %s first", "mine agents init")
	}

	m, err := ReadManifest()
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	// Auto-detect agents if none have been registered yet.
	if len(m.Agents) == 0 {
		detected := DetectAgents()
		m.Agents = detected
		if err := WriteManifest(m); err != nil {
			return nil, fmt.Errorf("saving detection results: %w", err)
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("determining home directory: %w", err)
	}

	storeDir := Dir()
	specs := buildLinkRegistry(home)
	var allItems []AdoptItem

	for _, spec := range specs {
		if opts.Agent != "" && spec.Name != opts.Agent {
			continue
		}
		if !isAgentDetected(m, spec.Name) {
			continue
		}

		items := scanAdoptableItems(storeDir, spec)
		allItems = append(allItems, items...)
	}

	if opts.DryRun {
		return allItems, nil
	}

	// Import items into the canonical store.
	var adoptedAgents []string
	for i := range allItems {
		item := &allItems[i]
		if item.Conflict || item.Status == "already-managed" {
			if item.Conflict {
				item.Status = "conflict"
			}
			continue
		}

		*item = performAdopt(*item)
		if item.Status == "imported" {
			adoptedAgents = appendUniq(adoptedAgents, item.Agent)
		}
	}

	// Replace originals with symlinks (unless --copy).
	if !opts.Copy && len(adoptedAgents) > 0 {
		for _, agentName := range adoptedAgents {
			linkOpts := LinkOptions{Agent: agentName, Copy: false, Force: true}
			_, _ = Link(linkOpts) // best-effort; errors are non-fatal for adopt
		}
	}

	// Auto-commit the imported content.
	if len(adoptedAgents) > 0 {
		commitMsg := "adopt: imported configs from " + strings.Join(adoptedAgents, ", ")
		_, _ = gitutil.RunCmd(storeDir, "add", ".")
		_, _ = gitutil.RunCmd(storeDir, "commit", "-m", commitMsg)
	}

	return allItems, nil
}

// scanAdoptableItems returns all adoptable items found in the agent's config directory.
// Items that are already managed (symlink pointing to the store) are omitted.
func scanAdoptableItems(storeDir string, spec linkSpec) []AdoptItem {
	var items []AdoptItem

	// 1. Instruction file (e.g. CLAUDE.md, GEMINI.md → instructions/AGENTS.md).
	instrPath := filepath.Join(spec.ConfigDir, spec.InstructionFilename)
	if fileExists(instrPath) && !isAlreadyManagedByStore(instrPath, storeDir) {
		storeRel := "instructions/AGENTS.md"
		storeAbs := filepath.Join(storeDir, storeRel)
		item := AdoptItem{
			Agent:      spec.Name,
			SourcePath: instrPath,
			StoreRel:   storeRel,
			StoreAbs:   storeAbs,
			Kind:       "instruction",
		}
		if fileExists(storeAbs) {
			if fileConflict(instrPath, storeAbs) {
				item.Conflict = true
			} else {
				item.Status = "already-managed"
			}
		}
		items = append(items, item)
	}

	// 2. Skills directory.
	if spec.SkillsDir != "" && dirNonEmpty(spec.SkillsDir) && !isAlreadyManagedByStore(spec.SkillsDir, storeDir) {
		items = append(items, AdoptItem{
			Agent:      spec.Name,
			SourcePath: spec.SkillsDir,
			StoreRel:   "skills",
			StoreAbs:   filepath.Join(storeDir, "skills"),
			Kind:       "skills",
		})
	}

	// 3. Commands directory (Claude only).
	if spec.CommandsDir != "" && dirNonEmpty(spec.CommandsDir) && !isAlreadyManagedByStore(spec.CommandsDir, storeDir) {
		items = append(items, AdoptItem{
			Agent:      spec.Name,
			SourcePath: spec.CommandsDir,
			StoreRel:   "commands",
			StoreAbs:   filepath.Join(storeDir, "commands"),
			Kind:       "commands",
		})
	}

	// 4. Settings file → settings/{agent}.json (per-agent, no conflict possible).
	if spec.SettingsFilename != "" {
		settingsPath := filepath.Join(spec.ConfigDir, spec.SettingsFilename)
		if fileExists(settingsPath) && !isAlreadyManagedByStore(settingsPath, storeDir) {
			storeRel := "settings/" + spec.Name + ".json"
			storeAbs := filepath.Join(storeDir, storeRel)
			item := AdoptItem{
				Agent:      spec.Name,
				SourcePath: settingsPath,
				StoreRel:   storeRel,
				StoreAbs:   storeAbs,
				Kind:       "settings",
			}
			if fileExists(storeAbs) {
				item.Status = "already-managed"
			}
			items = append(items, item)
		}
	}

	// 5. MCP config → mcp/.mcp.json.
	if spec.MCPConfigPath != "" && fileExists(spec.MCPConfigPath) && !isAlreadyManagedByStore(spec.MCPConfigPath, storeDir) {
		storeRel := "mcp/.mcp.json"
		storeAbs := filepath.Join(storeDir, storeRel)
		item := AdoptItem{
			Agent:      spec.Name,
			SourcePath: spec.MCPConfigPath,
			StoreRel:   storeRel,
			StoreAbs:   storeAbs,
			Kind:       "mcp",
		}
		if fileExists(storeAbs) {
			if fileConflict(spec.MCPConfigPath, storeAbs) {
				item.Conflict = true
			} else {
				item.Status = "already-managed"
			}
		}
		items = append(items, item)
	}

	// 6. Agents sub-directory (e.g. ~/.claude/agents/).
	agentsSubDir := filepath.Join(spec.ConfigDir, "agents")
	if dirNonEmpty(agentsSubDir) && !isAlreadyManagedByStore(agentsSubDir, storeDir) {
		items = append(items, AdoptItem{
			Agent:      spec.Name,
			SourcePath: agentsSubDir,
			StoreRel:   "agents",
			StoreAbs:   filepath.Join(storeDir, "agents"),
			Kind:       "agents",
		})
	}

	// 7. Rules sub-directory (e.g. ~/.claude/rules/).
	rulesSubDir := filepath.Join(spec.ConfigDir, "rules")
	if dirNonEmpty(rulesSubDir) && !isAlreadyManagedByStore(rulesSubDir, storeDir) {
		items = append(items, AdoptItem{
			Agent:      spec.Name,
			SourcePath: rulesSubDir,
			StoreRel:   "rules",
			StoreAbs:   filepath.Join(storeDir, "rules"),
			Kind:       "rules",
		})
	}

	return items
}

// performAdopt copies a single item into the canonical store.
// For file kinds (instruction, settings, mcp) it copies the file.
// For directory kinds (skills, commands, agents, rules) it merges content,
// skipping files that already exist in the store.
func performAdopt(item AdoptItem) AdoptItem {
	switch item.Kind {
	case "instruction", "settings", "mcp":
		if err := os.MkdirAll(filepath.Dir(item.StoreAbs), 0o755); err != nil {
			item.Status = "skipped"
			item.Err = fmt.Errorf("creating store directory: %w", err)
			return item
		}
		if err := copyFile(item.SourcePath, item.StoreAbs); err != nil {
			item.Status = "skipped"
			item.Err = fmt.Errorf("copying file: %w", err)
			return item
		}
	default:
		// Directory: merge without overwriting existing store files.
		if err := mergeDir(item.SourcePath, item.StoreAbs); err != nil {
			item.Status = "skipped"
			item.Err = fmt.Errorf("merging directory: %w", err)
			return item
		}
	}
	item.Status = "imported"
	return item
}

// mergeDir copies files from src into dst, skipping files that already exist in dst.
// This is the non-destructive merge strategy for directory adoption.
func mergeDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}

		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		// Skip files that already exist in the store.
		if fileExists(target) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFileMode(path, target, info.Mode())
	})
}

// fileConflict reports whether storeAbs already exists with different content than sourcePath.
func fileConflict(sourcePath, storeAbs string) bool {
	if !fileExists(storeAbs) {
		return false
	}
	srcData, err1 := os.ReadFile(sourcePath)
	dstData, err2 := os.ReadFile(storeAbs)
	if err1 != nil || err2 != nil {
		return false
	}
	return !bytes.Equal(srcData, dstData)
}

// isAlreadyManagedByStore reports whether path is a symlink that points inside storeDir.
func isAlreadyManagedByStore(path, storeDir string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	dest, err := os.Readlink(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(storeDir, dest)
	return err == nil && !strings.HasPrefix(rel, "..")
}

// appendUniq appends s to slice only if not already present.
func appendUniq(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}
