package agents

import (
	"fmt"
	"os"
	"path/filepath"
)

// LinkOptions controls the behavior of the Link operation.
type LinkOptions struct {
	Agent string // filter to a single agent name; empty means all detected agents
	Copy  bool   // create file copies instead of symlinks
	Force bool   // overwrite existing non-symlink files
}

// UnlinkOptions controls the behavior of the Unlink operation.
type UnlinkOptions struct {
	Agent string // filter to a single agent name; empty means all linked agents
}

// LinkAction describes what happened for a single link target.
type LinkAction struct {
	Source string // relative store path, e.g. "instructions/AGENTS.md"
	Target string // absolute target path
	Agent  string // which agent this serves
	Mode   string // "symlink" or "copy"
	Status string // "created", "updated", "skipped"
	Err    error  // non-nil if the action was skipped due to an error or safety check
}

// UnlinkAction describes what happened during unlink for a single entry.
type UnlinkAction struct {
	Target string // absolute path that was unlinked
	Agent  string // which agent this served
	Status string // "unlinked", "skipped"
	Err    error  // non-nil if skipped due to error
}

// Link creates symlinks (or copies) from the canonical store to each detected
// agent's expected configuration locations.
//
// Only detected agents are processed. Skips config types that don't exist in the
// store (e.g. empty skills/). Records all results in the manifest.
func Link(opts LinkOptions) ([]LinkAction, error) {
	if !IsInitialized() {
		return nil, fmt.Errorf("agents store not initialized — run %s first", "mine agents init")
	}

	m, err := ReadManifest()
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("determining home directory: %w", err)
	}

	storeDir := Dir()
	specs := buildLinkRegistry(home)
	var allActions []LinkAction

	for _, spec := range specs {
		if opts.Agent != "" && spec.Name != opts.Agent {
			continue
		}

		// Only process detected agents.
		if !isAgentDetected(m, spec.Name) {
			continue
		}

		actions := linkAgent(storeDir, spec, opts, m)
		allActions = append(allActions, actions...)
	}

	if err := WriteManifest(m); err != nil {
		return allActions, fmt.Errorf("saving manifest: %w", err)
	}

	return allActions, nil
}

// isAgentDetected returns true if the named agent is marked detected in the manifest.
func isAgentDetected(m *Manifest, name string) bool {
	for _, a := range m.Agents {
		if a.Name == name && a.Detected {
			return true
		}
	}
	return false
}

// linkAgent processes all link targets for a single agent spec.
func linkAgent(storeDir string, spec linkSpec, opts LinkOptions, m *Manifest) []LinkAction {
	var actions []LinkAction

	// 1. Instructions file — only if it exists in the store.
	instrSource := filepath.Join(storeDir, "instructions", "AGENTS.md")
	if fileExists(instrSource) {
		instrTarget := filepath.Join(spec.ConfigDir, spec.InstructionFilename)
		a := createFileLink(instrSource, "instructions/AGENTS.md", instrTarget, spec.Name, opts, m)
		actions = append(actions, a)
	}

	// 2. Skills directory — only if store's skills/ is non-empty and agent supports it.
	if spec.SkillsDir != "" {
		skillsSource := filepath.Join(storeDir, "skills")
		if dirNonEmpty(skillsSource) {
			a := createDirLink(skillsSource, "skills", spec.SkillsDir, spec.Name, opts, m)
			actions = append(actions, a)
		}
	}

	// 3. Commands directory — only for agents that support it (Claude) and if non-empty.
	if spec.CommandsDir != "" {
		cmdSource := filepath.Join(storeDir, "commands")
		if dirNonEmpty(cmdSource) {
			a := createDirLink(cmdSource, "commands", spec.CommandsDir, spec.Name, opts, m)
			actions = append(actions, a)
		}
	}

	// 4. Settings file — only if settings/{agent}.json exists in the store.
	settingsSource := filepath.Join(storeDir, "settings", spec.Name+".json")
	if fileExists(settingsSource) {
		settingsTarget := filepath.Join(spec.ConfigDir, spec.SettingsFilename)
		a := createFileLink(settingsSource, "settings/"+spec.Name+".json", settingsTarget, spec.Name, opts, m)
		actions = append(actions, a)
	}

	// 5. MCP config — only for agents that support it and if mcp/.mcp.json exists.
	if spec.MCPConfigPath != "" {
		mcpSource := filepath.Join(storeDir, "mcp", ".mcp.json")
		if fileExists(mcpSource) {
			a := createFileLink(mcpSource, "mcp/.mcp.json", spec.MCPConfigPath, spec.Name, opts, m)
			actions = append(actions, a)
		}
	}

	return actions
}

// createFileLink handles single-file link creation with safety checks.
func createFileLink(sourcePath, sourceRel, target, agentName string, opts LinkOptions, m *Manifest) LinkAction {
	mode := "symlink"
	if opts.Copy {
		mode = "copy"
	}

	action := LinkAction{
		Source: sourceRel,
		Target: target,
		Agent:  agentName,
		Mode:   mode,
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		action.Status = "skipped"
		action.Err = fmt.Errorf("creating parent directory: %w", err)
		return action
	}

	existed, alreadyLinked, safeErr := checkFileSafety(sourcePath, target, opts.Force)
	if safeErr != nil {
		action.Status = "skipped"
		action.Err = safeErr
		return action
	}

	if alreadyLinked {
		// Already pointing to our canonical store — update manifest entry silently.
		action.Status = "updated"
		upsertManifestLink(m, sourceRel, target, agentName, mode)
		return action
	}

	if existed {
		if err := os.Remove(target); err != nil {
			action.Status = "skipped"
			action.Err = fmt.Errorf("removing existing target: %w", err)
			return action
		}
	}

	if opts.Copy {
		action.Err = copyFile(sourcePath, target)
	} else {
		action.Err = os.Symlink(sourcePath, target)
	}

	if action.Err != nil {
		action.Status = "skipped"
		action.Err = fmt.Errorf("creating link: %w", action.Err)
		return action
	}

	action.Status = "created"
	upsertManifestLink(m, sourceRel, target, agentName, mode)
	return action
}

// createDirLink handles directory-level link creation with safety checks.
func createDirLink(sourcePath, sourceRel, target, agentName string, opts LinkOptions, m *Manifest) LinkAction {
	mode := "symlink"
	if opts.Copy {
		mode = "copy"
	}

	action := LinkAction{
		Source: sourceRel,
		Target: target,
		Agent:  agentName,
		Mode:   mode,
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		action.Status = "skipped"
		action.Err = fmt.Errorf("creating parent directory: %w", err)
		return action
	}

	existed, alreadyLinked, safeErr := checkDirSafety(sourcePath, target, opts.Force)
	if safeErr != nil {
		action.Status = "skipped"
		action.Err = safeErr
		return action
	}

	if alreadyLinked {
		action.Status = "updated"
		upsertManifestLink(m, sourceRel, target, agentName, mode)
		return action
	}

	if existed {
		if err := os.RemoveAll(target); err != nil {
			action.Status = "skipped"
			action.Err = fmt.Errorf("removing existing target: %w", err)
			return action
		}
	}

	if opts.Copy {
		action.Err = copyDir(sourcePath, target)
	} else {
		action.Err = os.Symlink(sourcePath, target)
	}

	if action.Err != nil {
		action.Status = "skipped"
		action.Err = fmt.Errorf("creating directory link: %w", action.Err)
		return action
	}

	action.Status = "created"
	upsertManifestLink(m, sourceRel, target, agentName, mode)
	return action
}

// checkFileSafety checks whether it is safe to create a file link at target.
//
// Returns:
//   - existed: target path exists
//   - alreadyLinked: target is already a symlink pointing to sourcePath
//   - err: non-nil if the operation should be aborted
func checkFileSafety(sourcePath, target string, force bool) (existed, alreadyLinked bool, err error) {
	info, statErr := os.Lstat(target)
	if statErr != nil {
		// Target doesn't exist — safe to create.
		return false, false, nil
	}

	if info.Mode()&os.ModeSymlink != 0 {
		dest, _ := os.Readlink(target)
		if dest == sourcePath {
			// Already pointing to our canonical store.
			return true, true, nil
		}
		if !force {
			return true, false, fmt.Errorf("target %s is a symlink pointing to %s; use --force to overwrite", target, dest)
		}
		return true, false, nil
	}

	// Regular file.
	if !force {
		return true, false, fmt.Errorf("target %s exists as a regular file; run %s to adopt it first, or use --force to overwrite",
			target, "mine agents adopt")
	}
	return true, false, nil
}

// checkDirSafety checks whether it is safe to create a directory link at target.
//
// Returns:
//   - existed: target path exists
//   - alreadyLinked: target is already a symlink pointing to sourcePath
//   - err: non-nil if the operation should be aborted
func checkDirSafety(sourcePath, target string, force bool) (existed, alreadyLinked bool, err error) {
	info, statErr := os.Lstat(target)
	if statErr != nil {
		// Target doesn't exist — safe to create.
		return false, false, nil
	}

	if info.Mode()&os.ModeSymlink != 0 {
		dest, _ := os.Readlink(target)
		if dest == sourcePath {
			return true, true, nil
		}
		if !force {
			return true, false, fmt.Errorf("target %s is a symlink pointing to %s; use --force to overwrite", target, dest)
		}
		return true, false, nil
	}

	// Regular directory.
	if !force {
		return true, false, fmt.Errorf("target %s exists as a directory; run %s to adopt it first, or use --force to overwrite",
			target, "mine agents adopt")
	}
	return true, false, nil
}

// Unlink reverses symlinks by replacing them with standalone file copies.
// For copy-mode entries, they already stand alone — only manifest tracking is removed.
func Unlink(opts UnlinkOptions) ([]UnlinkAction, error) {
	if !IsInitialized() {
		return nil, fmt.Errorf("agents store not initialized — run %s first", "mine agents init")
	}

	m, err := ReadManifest()
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var actions []UnlinkAction
	var remainingLinks []LinkEntry

	for _, link := range m.Links {
		if opts.Agent != "" && link.Agent != opts.Agent {
			remainingLinks = append(remainingLinks, link)
			continue
		}

		a := unlinkEntry(link)
		actions = append(actions, a)

		if a.Err != nil {
			// Could not unlink — keep in manifest.
			remainingLinks = append(remainingLinks, link)
		}
		// Successfully unlinked — omit from remainingLinks.
	}

	m.Links = remainingLinks
	if m.Links == nil {
		m.Links = []LinkEntry{}
	}

	if err := WriteManifest(m); err != nil {
		return actions, fmt.Errorf("saving manifest: %w", err)
	}

	return actions, nil
}

// unlinkEntry reverses a single link entry.
func unlinkEntry(link LinkEntry) UnlinkAction {
	action := UnlinkAction{
		Target: link.Target,
		Agent:  link.Agent,
	}

	if link.Mode == "copy" {
		// Copies already stand alone — just remove manifest tracking.
		action.Status = "unlinked"
		return action
	}

	// Symlink mode: replace with a standalone copy of the linked content.
	info, err := os.Lstat(link.Target)
	if err != nil {
		if os.IsNotExist(err) {
			// Already gone — remove from manifest silently.
			action.Status = "unlinked"
			return action
		}
		action.Status = "skipped"
		action.Err = fmt.Errorf("checking target: %w", err)
		return action
	}

	if info.Mode()&os.ModeSymlink == 0 {
		// Not a symlink — already a standalone file, just remove from tracking.
		action.Status = "unlinked"
		return action
	}

	dest, err := os.Readlink(link.Target)
	if err != nil {
		action.Status = "skipped"
		action.Err = fmt.Errorf("reading symlink: %w", err)
		return action
	}

	// Check whether the symlink points to a directory or file.
	destInfo, err := os.Stat(dest)
	if err != nil {
		// Dangling symlink — just remove it.
		if removeErr := os.Remove(link.Target); removeErr != nil && !os.IsNotExist(removeErr) {
			action.Status = "skipped"
			action.Err = fmt.Errorf("removing dangling symlink: %w", removeErr)
			return action
		}
		action.Status = "unlinked"
		return action
	}

	if destInfo.IsDir() {
		// Directory symlink: remove symlink and copy the directory content.
		if err := os.Remove(link.Target); err != nil {
			action.Status = "skipped"
			action.Err = fmt.Errorf("removing directory symlink: %w", err)
			return action
		}
		if err := copyDir(dest, link.Target); err != nil {
			action.Status = "skipped"
			action.Err = fmt.Errorf("copying directory content: %w", err)
			return action
		}
	} else {
		// File symlink: read content, remove symlink, write standalone file.
		data, err := os.ReadFile(link.Target)
		if err != nil {
			action.Status = "skipped"
			action.Err = fmt.Errorf("reading symlink content: %w", err)
			return action
		}
		if err := os.Remove(link.Target); err != nil {
			action.Status = "skipped"
			action.Err = fmt.Errorf("removing file symlink: %w", err)
			return action
		}
		if err := os.WriteFile(link.Target, data, destInfo.Mode().Perm()); err != nil {
			action.Status = "skipped"
			action.Err = fmt.Errorf("writing standalone file: %w", err)
			return action
		}
		// Explicitly set permissions to match the source, bypassing the umask.
		if err := os.Chmod(link.Target, destInfo.Mode().Perm()); err != nil {
			action.Status = "skipped"
			action.Err = fmt.Errorf("setting file permissions: %w", err)
			return action
		}
	}

	action.Status = "unlinked"
	return action
}
