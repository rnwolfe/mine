package agents

import (
	"fmt"
	"os"
	"path/filepath"
)

// projectSpec defines the project-level configuration structure for a single coding agent.
type projectSpec struct {
	Name             string // agent identifier: "claude", "codex", "gemini", "opencode"
	ConfigDir        string // relative config dir within project: ".claude", ".agents", etc.
	InstructionFile  string // instruction filename at project root: "CLAUDE.md", "AGENTS.md", etc.
	SkillsSubDir     string // skills subdirectory within config dir: "skills"
	CommandsSubDir   string // commands subdirectory within config dir (claude only): "commands"
	SettingsFilename string // settings JSON filename within config dir: "settings.json"
}

// buildProjectSpecRegistry returns the project-level specs for all supported agents.
func buildProjectSpecRegistry() []projectSpec {
	return []projectSpec{
		{
			Name:             "claude",
			ConfigDir:        ".claude",
			InstructionFile:  "CLAUDE.md",
			SkillsSubDir:     "skills",
			CommandsSubDir:   "commands",
			SettingsFilename: "settings.json",
		},
		{
			Name:             "codex",
			ConfigDir:        ".agents",
			InstructionFile:  "AGENTS.md",
			SkillsSubDir:     "skills",
			SettingsFilename: "settings.json",
		},
		{
			Name:             "gemini",
			ConfigDir:        ".gemini",
			InstructionFile:  "GEMINI.md",
			SkillsSubDir:     "skills",
			SettingsFilename: "settings.json",
		},
		{
			Name:             "opencode",
			ConfigDir:        ".opencode",
			InstructionFile:  "AGENTS.md",
			SkillsSubDir:     "skills",
			SettingsFilename: "settings.json",
		},
	}
}

// Starter instruction file templates for project-level init.
var projectInstructionTemplates = map[string]string{
	"AGENTS.md": `# Agent Instructions

This file contains project-specific instructions for your coding agents.

## Project Context

<!-- Describe this project's purpose, conventions, and coding standards -->

## Instructions

<!-- Add project-specific agent instructions below this line -->
`,
	"CLAUDE.md": `# Claude Instructions

This file contains project-specific instructions for Claude Code.

## Project Context

<!-- Describe this project's purpose, conventions, and coding standards for Claude -->

## Instructions

<!-- Add Claude-specific project instructions below this line -->
`,
	"GEMINI.md": `# Gemini Instructions

This file contains project-specific instructions for Gemini CLI.

## Project Context

<!-- Describe this project's purpose, conventions, and coding standards for Gemini -->

## Instructions

<!-- Add Gemini-specific project instructions below this line -->
`,
}

// ProjectInitOptions controls the behavior of ProjectInit.
type ProjectInitOptions struct {
	Force bool // overwrite existing files without prompting
}

// ProjectInitAction describes the result of a single file or directory operation during init.
type ProjectInitAction struct {
	Kind   string // "dir" or "file"
	Path   string // absolute path
	Status string // "created", "exists", "skipped"
	Err    error  // non-nil if the action failed
}

// ProjectLinkOptions controls the behavior of ProjectLink.
type ProjectLinkOptions struct {
	Agent string // filter to a single agent name; empty means all detected agents
	Copy  bool   // copy files instead of creating symlinks
	Force bool   // overwrite existing files
}

// ProjectInit scaffolds project-level agent configuration directories.
//
// projectPath is the directory to scaffold; if empty, defaults to the current
// working directory. Only scaffolds directories for detected agents (using the
// manifest if the store is initialized, live scan otherwise). Re-running is
// safe — existing files and directories are skipped unless Force is set.
func ProjectInit(projectPath string, opts ProjectInitOptions) ([]ProjectInitAction, error) {
	var err error
	if projectPath == "" {
		projectPath, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting current directory: %w", err)
		}
	}

	if err := validateProjectPath(projectPath); err != nil {
		return nil, err
	}

	detected := projectDetectedAgents()
	if len(detected) == 0 {
		return nil, nil
	}

	detectedSet := make(map[string]bool, len(detected))
	for _, a := range detected {
		detectedSet[a.Name] = true
	}

	storeDir := Dir()
	specs := buildProjectSpecRegistry()
	var actions []ProjectInitAction
	createdInstructions := make(map[string]bool)

	for _, spec := range specs {
		if !detectedSet[spec.Name] {
			continue
		}

		configDir := filepath.Join(projectPath, spec.ConfigDir)

		// 1. Create agent config directory.
		a := initProjectDir(configDir)
		actions = append(actions, a)

		// 2. Create skills subdirectory.
		if spec.SkillsSubDir != "" {
			a = initProjectDir(filepath.Join(configDir, spec.SkillsSubDir))
			actions = append(actions, a)
		}

		// 3. Create commands subdirectory (claude only).
		if spec.CommandsSubDir != "" {
			a = initProjectDir(filepath.Join(configDir, spec.CommandsSubDir))
			actions = append(actions, a)
		}

		// 4. Seed settings from canonical store if a template exists.
		if spec.SettingsFilename != "" && IsInitialized() {
			settingsSrc := filepath.Join(storeDir, "settings", spec.Name+".json")
			if fileExists(settingsSrc) {
				dst := filepath.Join(configDir, spec.SettingsFilename)
				a = seedProjectFile(settingsSrc, dst, opts.Force)
				actions = append(actions, a)
			}
		}

		// 5. Create instruction file at project root (deduped for shared filenames).
		if spec.InstructionFile != "" && !createdInstructions[spec.InstructionFile] {
			instrPath := filepath.Join(projectPath, spec.InstructionFile)
			content := projectInstructionTemplates[spec.InstructionFile]
			a = initProjectInstructionFile(instrPath, content, opts.Force)
			actions = append(actions, a)
			createdInstructions[spec.InstructionFile] = true
		}
	}

	return actions, nil
}

// ProjectLink creates symlinks (or copies) from the canonical agents store to
// project-level skill directories.
//
// projectPath is the project directory to link into; if empty, defaults to the
// current working directory. Requires the agents store to be initialized.
// Only processes detected agents. Results are tracked in the global manifest.
func ProjectLink(projectPath string, opts ProjectLinkOptions) ([]LinkAction, error) {
	if !IsInitialized() {
		return nil, fmt.Errorf("agents store not initialized — run %s first", "mine agents init")
	}

	var err error
	if projectPath == "" {
		projectPath, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting current directory: %w", err)
		}
	}

	if err := validateProjectPath(projectPath); err != nil {
		return nil, err
	}

	m, err := ReadManifest()
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	storeDir := Dir()
	specs := buildProjectSpecRegistry()
	linkOpts := LinkOptions{Copy: opts.Copy, Force: opts.Force}
	var allActions []LinkAction

	for _, spec := range specs {
		if opts.Agent != "" && spec.Name != opts.Agent {
			continue
		}
		if !isAgentDetected(m, spec.Name) {
			continue
		}

		configDir := filepath.Join(projectPath, spec.ConfigDir)

		// Link skills directory if canonical skills/ is non-empty.
		if spec.SkillsSubDir != "" {
			skillsSrc := filepath.Join(storeDir, "skills")
			if dirNonEmpty(skillsSrc) {
				target := filepath.Join(configDir, spec.SkillsSubDir)
				a := createDirLink(skillsSrc, "skills", target, spec.Name, linkOpts, m)
				allActions = append(allActions, a)
			}
		}

		// Link commands directory for agents that support it (claude only).
		if spec.CommandsSubDir != "" {
			cmdSrc := filepath.Join(storeDir, "commands")
			if dirNonEmpty(cmdSrc) {
				target := filepath.Join(configDir, spec.CommandsSubDir)
				a := createDirLink(cmdSrc, "commands", target, spec.Name, linkOpts, m)
				allActions = append(allActions, a)
			}
		}

		// Link settings file if canonical settings exist.
		if spec.SettingsFilename != "" {
			settingsSrc := filepath.Join(storeDir, "settings", spec.Name+".json")
			if fileExists(settingsSrc) {
				target := filepath.Join(configDir, spec.SettingsFilename)
				a := createFileLink(settingsSrc, "settings/"+spec.Name+".json", target, spec.Name, linkOpts, m)
				allActions = append(allActions, a)
			}
		}
	}

	if err := WriteManifest(m); err != nil {
		return allActions, fmt.Errorf("saving manifest: %w", err)
	}

	return allActions, nil
}

// validateProjectPath checks that the given path exists and is a directory.
func validateProjectPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("project path %q does not exist", path)
		}
		return fmt.Errorf("checking project path %q: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("project path %q is not a directory", path)
	}
	return nil
}

// projectDetectedAgents returns detected agents from the manifest when the store
// is initialized, or falls back to a live detection scan when it is not.
func projectDetectedAgents() []Agent {
	if IsInitialized() {
		m, err := ReadManifest()
		if err != nil {
			// When initialized, the manifest is authoritative. If it can't be read,
			// treat this as zero detected agents rather than falling back to live detection.
			return nil
		}
		var detected []Agent
		for _, a := range m.Agents {
			if a.Detected {
				detected = append(detected, a)
			}
		}
		// The manifest is authoritative when the store is initialized,
		// even if it reports zero detected agents.
		return detected
	}
	// Store not initialized: fall back to live detection.
	var detected []Agent
	for _, a := range DetectAgents() {
		if a.Detected {
			detected = append(detected, a)
		}
	}
	return detected
}

// initProjectDir creates a directory if it doesn't already exist.
func initProjectDir(path string) ProjectInitAction {
	action := ProjectInitAction{Kind: "dir", Path: path}

	info, err := os.Lstat(path)
	if err == nil {
		if info.IsDir() {
			action.Status = "exists"
			return action
		}
		// If it's a symlink, follow it and accept if it resolves to a directory
		// (e.g. a project already linked via mine agents project link).
		if info.Mode()&os.ModeSymlink != 0 {
			resolved, statErr := os.Stat(path)
			if statErr != nil {
				action.Status = "skipped"
				action.Err = fmt.Errorf("checking symlink target: %w", statErr)
				return action
			}
			if resolved.IsDir() {
				action.Status = "exists"
				return action
			}
		}
		action.Status = "skipped"
		action.Err = fmt.Errorf("path exists but is not a directory")
		return action
	}
	if !os.IsNotExist(err) {
		action.Status = "skipped"
		action.Err = fmt.Errorf("checking path: %w", err)
		return action
	}

	if err := os.MkdirAll(path, 0o755); err != nil {
		action.Status = "skipped"
		action.Err = fmt.Errorf("creating directory: %w", err)
		return action
	}

	action.Status = "created"
	return action
}

// seedProjectFile copies src to dst if dst doesn't exist (or force is set).
func seedProjectFile(src, dst string, force bool) ProjectInitAction {
	action := ProjectInitAction{Kind: "file", Path: dst}

	_, err := os.Lstat(dst)
	if err == nil {
		if !force {
			action.Status = "exists"
			return action
		}
		if removeErr := os.Remove(dst); removeErr != nil {
			action.Status = "skipped"
			action.Err = fmt.Errorf("removing existing file: %w", removeErr)
			return action
		}
	} else if !os.IsNotExist(err) {
		action.Status = "skipped"
		action.Err = fmt.Errorf("checking destination: %w", err)
		return action
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		action.Status = "skipped"
		action.Err = fmt.Errorf("creating parent directory: %w", err)
		return action
	}

	if err := copyFile(src, dst); err != nil {
		action.Status = "skipped"
		action.Err = fmt.Errorf("seeding settings file: %w", err)
		return action
	}

	action.Status = "created"
	return action
}

// initProjectInstructionFile creates an instruction file with the given content
// if it doesn't already exist (or force is set).
func initProjectInstructionFile(path, content string, force bool) ProjectInitAction {
	action := ProjectInitAction{Kind: "file", Path: path}

	if _, err := os.Lstat(path); err == nil {
		if !force {
			action.Status = "exists"
			return action
		}
		if removeErr := os.Remove(path); removeErr != nil {
			action.Status = "skipped"
			action.Err = fmt.Errorf("removing existing file: %w", removeErr)
			return action
		}
	} else if !os.IsNotExist(err) {
		action.Status = "skipped"
		action.Err = fmt.Errorf("checking path: %w", err)
		return action
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		action.Status = "skipped"
		action.Err = fmt.Errorf("writing instruction file: %w", err)
		return action
	}

	action.Status = "created"
	return action
}
