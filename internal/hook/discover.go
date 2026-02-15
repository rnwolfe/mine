package hook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rnwolfe/mine/internal/config"
)

// UserHook represents a hook script discovered from the hooks directory.
type UserHook struct {
	Path    string
	Pattern string
	Stage   Stage
	Name    string
}

// HooksDir returns the user hooks directory path.
func HooksDir() string {
	return filepath.Join(config.GetPaths().ConfigDir, "hooks")
}

// Discover scans the user hooks directory and returns all valid hook scripts.
// Scripts follow the naming convention: <command-pattern>.<stage>.<ext>
// Examples: todo.add.preexec.sh, todo.*.notify.py, *.postexec.sh
func Discover() ([]UserHook, error) {
	dir := HooksDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading hooks dir: %w", err)
	}

	var hooks []UserHook
	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		name := e.Name()
		h, err := parseHookFilename(name)
		if err != nil {
			continue // skip files that don't match the naming convention
		}

		path := filepath.Join(dir, name)

		// Check executable permission
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.Mode()&0o111 == 0 {
			continue // not executable
		}

		h.Path = path
		hooks = append(hooks, h)
	}

	return hooks, nil
}

// parseHookFilename parses a hook filename into its components.
// Format: <command-pattern>.<stage>.<ext>
// The command pattern may contain dots, so we parse from right to left.
// Examples:
//
//	todo.add.preexec.sh    → pattern="todo.add", stage="preexec"
//	todo.*.notify.py       → pattern="todo.*",   stage="notify"
//	*.postexec.sh          → pattern="*",         stage="postexec"
func parseHookFilename(name string) (UserHook, error) {
	// Remove extension
	ext := filepath.Ext(name)
	if ext == "" {
		// No extension — still valid if the rest parses
		ext = ""
	}
	base := strings.TrimSuffix(name, ext)

	// Find the stage (last dot-separated component of base)
	lastDot := strings.LastIndex(base, ".")
	if lastDot < 0 {
		return UserHook{}, fmt.Errorf("invalid hook filename: %s", name)
	}

	stageStr := base[lastDot+1:]
	pattern := base[:lastDot]

	stage, err := parseStage(stageStr)
	if err != nil {
		return UserHook{}, fmt.Errorf("invalid stage in %s: %w", name, err)
	}

	if pattern == "" {
		return UserHook{}, fmt.Errorf("empty pattern in %s", name)
	}

	return UserHook{
		Pattern: pattern,
		Stage:   stage,
		Name:    name,
	}, nil
}

// ParseStageStr converts a stage string to a Stage constant. Exported for CLI use.
func ParseStageStr(s string) (Stage, error) {
	return parseStage(s)
}

// parseStage converts a string to a Stage constant.
func parseStage(s string) (Stage, error) {
	switch Stage(s) {
	case StagePrevalidate, StagePreexec, StagePostexec, StageNotify:
		return Stage(s), nil
	default:
		return "", fmt.Errorf("unknown stage: %s", s)
	}
}

// RegisterUserHooks discovers and registers all user-local hooks.
func RegisterUserHooks() error {
	hooks, err := Discover()
	if err != nil {
		return err
	}

	for _, h := range hooks {
		mode := ModeTransform
		if h.Stage == StageNotify {
			mode = ModeNotify
		}

		timeout := DefaultTransformTimeout
		if mode == ModeNotify {
			timeout = DefaultNotifyTimeout
		}

		if err := Register(Hook{
			Pattern: h.Pattern,
			Stage:   h.Stage,
			Mode:    mode,
			Name:    h.Name,
			Source:  "user",
			Handler: ExecHandler(h.Path, mode, timeout),
			Timeout: timeout,
		}); err != nil {
			return fmt.Errorf("registering hook %s: %w", h.Name, err)
		}
	}

	return nil
}

// CreateHookScript generates a starter hook script at the given path.
func CreateHookScript(pattern string, stage Stage) (string, error) {
	// Sanitize pattern to prevent directory traversal
	if strings.ContainsAny(pattern, "/\\") {
		return "", fmt.Errorf("pattern %q must not contain path separators", pattern)
	}
	if pattern == ".." || strings.Contains(pattern, "..") {
		return "", fmt.Errorf("pattern %q must not contain path traversal", pattern)
	}

	dir := HooksDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating hooks dir: %w", err)
	}

	filename := fmt.Sprintf("%s.%s.sh", pattern, stage)
	path := filepath.Join(dir, filename)

	// Verify the resolved path is within the hooks directory
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolving hooks dir: %w", err)
	}
	if !strings.HasPrefix(absPath, absDir+string(filepath.Separator)) {
		return "", fmt.Errorf("hook path escapes hooks directory")
	}

	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("hook already exists: %s", path)
	}

	mode := "transform"
	if stage == StageNotify {
		mode = "notify"
	}

	script := fmt.Sprintf(`#!/bin/bash
# mine hook: %s at %s stage (%s mode)
# Created: %s
#
# This script receives a JSON context on stdin.
# For transform hooks, write modified JSON to stdout.
# For notify hooks, perform side effects (output is ignored).
#
# Input JSON format:
# {
#   "command": "todo.add",
#   "args": ["buy milk"],
#   "flags": {"priority": "high"},
#   "result": null,
#   "timestamp": "2026-01-15T10:30:00Z"
# }

# Read context from stdin
CONTEXT=$(cat)

# Example: log the command
COMMAND=$(echo "$CONTEXT" | grep -o '"command":"[^"]*"' | cut -d'"' -f4)
# echo "Hook fired for: $COMMAND" >&2
`, pattern, stage, mode, time.Now().Format("2006-01-02"))

	if stage != StageNotify {
		script += `
# For transform hooks: echo modified context to stdout
echo "$CONTEXT"
`
	}

	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		return "", fmt.Errorf("writing hook script: %w", err)
	}

	return path, nil
}

// TestHook performs a dry-run of a hook script with sample input.
func TestHook(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("hook not found: %s", path)
	}
	if info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("hook not executable: %s (run: chmod +x %s)", path, path)
	}

	h, err := parseHookFilename(filepath.Base(path))
	if err != nil {
		return "", err
	}

	mode := ModeTransform
	timeout := DefaultTransformTimeout
	if h.Stage == StageNotify {
		mode = ModeNotify
		timeout = DefaultNotifyTimeout
	}

	ctx := NewContext("test.command", []string{"sample", "args"}, map[string]string{
		"flag1": "value1",
	})

	handler := ExecHandler(path, mode, timeout)
	result, err := handler(ctx)
	if err != nil {
		return "", fmt.Errorf("hook execution failed: %w", err)
	}

	if mode == ModeNotify {
		return "Notify hook executed successfully (no output expected)", nil
	}

	data, err := result.JSON()
	if err != nil {
		return "", fmt.Errorf("serializing result: %w", err)
	}
	return string(data), nil
}
