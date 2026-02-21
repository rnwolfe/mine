package plugin

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/hook"
)

// examplesDir returns the absolute path to docs/examples/plugins/.
func examplesDir(t *testing.T) string {
	t.Helper()
	// Walk up from internal/plugin/ to the repo root.
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	dir := filepath.Join(repoRoot, "docs", "examples", "plugins")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("examples directory not found: %v", err)
	}
	return dir
}

// TestExampleManifestsValid validates every mine-plugin.toml under
// docs/examples/plugins/ parses successfully and uses the current protocol version.
func TestExampleManifestsValid(t *testing.T) {
	base := examplesDir(t)
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatalf("reading examples dir: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("no example plugins found")
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			manifestPath := filepath.Join(base, name, "mine-plugin.toml")
			m, err := ParseManifest(manifestPath)
			if err != nil {
				t.Fatalf("ParseManifest(%s) error: %v", name, err)
			}

			if m.Plugin.ProtocolVersion != ProtocolVersion {
				t.Errorf("protocol_version = %q, want %q", m.Plugin.ProtocolVersion, ProtocolVersion)
			}

			if m.Plugin.Name == "" {
				t.Error("plugin name is empty")
			}
			if m.Plugin.Version == "" {
				t.Error("plugin version is empty")
			}
			if m.Plugin.Description == "" {
				t.Error("plugin description is empty")
			}
			if m.Plugin.Author == "" {
				t.Error("plugin author is empty")
			}
		})
	}
}

// TestExampleTodoStats exercises the shell-based todo-stats example plugin
// through its full lifecycle: install, run command, invoke hook, lifecycle event, remove.
func TestExampleTodoStats(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("XDG_DATA_HOME", dir)

	srcDir := filepath.Join(examplesDir(t), "todo-stats")

	// Install
	p, err := Install(srcDir, "local://todo-stats")
	if err != nil {
		t.Fatalf("Install() error: %v", err)
	}
	if p.Manifest.Plugin.Name != "todo-stats" {
		t.Errorf("name = %q, want %q", p.Manifest.Plugin.Name, "todo-stats")
	}

	// Verify binary was copied
	binPath := filepath.Join(p.Dir, p.Manifest.Entrypoint())
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("plugin binary not found: %v", err)
	}

	// List and Get
	plugins, err := List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("List() = %d plugins, want 1", len(plugins))
	}

	got, err := Get("todo-stats")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	// Run command
	if err := RunCommand(got, "summary", nil); err != nil {
		t.Fatalf("RunCommand(summary) error: %v", err)
	}

	// Invoke notify hook
	notifyHandler := pluginHookHandler(binPath, "notify", "notify", 30e9, got.Manifest.Permissions)
	ctx := &hook.Context{
		Command:   "todo.done",
		Args:      []string{"buy milk"},
		Flags:     map[string]string{},
		Timestamp: "2026-01-01T00:00:00Z",
	}
	if _, err := notifyHandler(ctx); err != nil {
		t.Fatalf("notify hook error: %v", err)
	}

	// Lifecycle event
	if err := SendLifecycleEvent(got, "health"); err != nil {
		t.Fatalf("SendLifecycleEvent(health) error: %v", err)
	}

	// Remove
	if err := Remove("todo-stats"); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}
	plugins, err = List()
	if err != nil {
		t.Fatalf("List() after remove: %v", err)
	}
	if len(plugins) != 0 {
		t.Errorf("List() after remove = %d, want 0", len(plugins))
	}
}

// TestExampleWebhook exercises the Python-based webhook example plugin.
func TestExampleWebhook(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not found, skipping webhook example test")
	}

	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("XDG_DATA_HOME", dir)

	srcDir := filepath.Join(examplesDir(t), "webhook")

	// Install
	p, err := Install(srcDir, "local://webhook")
	if err != nil {
		t.Fatalf("Install() error: %v", err)
	}
	if p.Manifest.Plugin.Name != "webhook" {
		t.Errorf("name = %q, want %q", p.Manifest.Plugin.Name, "webhook")
	}

	binPath := filepath.Join(p.Dir, p.Manifest.Entrypoint())

	got, err := Get("webhook")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	// Run command
	if err := RunCommand(got, "help", nil); err != nil {
		t.Fatalf("RunCommand(help) error: %v", err)
	}

	// Transform hook (preexec on todo.add)
	transformHandler := pluginHookHandler(binPath, "preexec", "transform", 5e9, got.Manifest.Permissions)
	ctx := &hook.Context{
		Command:   "todo.add",
		Args:      []string{"buy milk"},
		Flags:     map[string]string{},
		Timestamp: "2026-01-01T00:00:00Z",
	}
	result, err := transformHandler(ctx)
	if err != nil {
		t.Fatalf("transform hook error: %v", err)
	}
	if result.Command != "todo.add" {
		t.Errorf("transform result command = %q, want %q", result.Command, "todo.add")
	}

	// Notify hook (todo.done) — no WEBHOOK_URL set, should succeed silently
	notifyHandler := pluginHookHandler(binPath, "notify", "notify", 30e9, got.Manifest.Permissions)
	if _, err := notifyHandler(ctx); err != nil {
		t.Fatalf("notify hook error: %v", err)
	}

	// Lifecycle
	if err := SendLifecycleEvent(got, "health"); err != nil {
		t.Fatalf("SendLifecycleEvent(health) error: %v", err)
	}

	// Remove
	if err := Remove("webhook"); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}
}

// TestExampleTagEnforcer exercises the Go-based tag-enforcer example plugin.
// It builds the binary first, then runs the full lifecycle.
func TestExampleTagEnforcer(t *testing.T) {
	srcDir := filepath.Join(examplesDir(t), "tag-enforcer")

	// Build the Go plugin binary
	binName := "mine-plugin-tag-enforcer"
	binPath := filepath.Join(srcDir, binName)
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = srcDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}
	defer os.Remove(binPath)

	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("XDG_DATA_HOME", dir)

	// Install
	p, err := Install(srcDir, "local://tag-enforcer")
	if err != nil {
		t.Fatalf("Install() error: %v", err)
	}
	if p.Manifest.Plugin.Name != "tag-enforcer" {
		t.Errorf("name = %q, want %q", p.Manifest.Plugin.Name, "tag-enforcer")
	}

	installedBin := filepath.Join(p.Dir, binName)
	if _, err := os.Stat(installedBin); err != nil {
		t.Fatalf("plugin binary not found after install: %v", err)
	}

	got, err := Get("tag-enforcer")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}

	// Run command
	if err := RunCommand(got, "policy", nil); err != nil {
		t.Fatalf("RunCommand(policy) error: %v", err)
	}

	// Transform hook — prevalidate on todo.add (no tags → should add "untagged")
	transformHandler := pluginHookHandler(installedBin, "prevalidate", "transform", 5e9, got.Manifest.Permissions)
	ctx := &hook.Context{
		Command:   "todo.add",
		Args:      []string{"buy milk"},
		Flags:     map[string]string{},
		Timestamp: "2026-01-01T00:00:00Z",
	}
	result, err := transformHandler(ctx)
	if err != nil {
		t.Fatalf("prevalidate hook error: %v", err)
	}
	if result.Flags == nil || result.Flags["tags"] != "untagged" {
		t.Errorf("expected tags='untagged', got flags=%v", result.Flags)
	}

	// Transform hook — prevalidate with existing tags (should pass through)
	ctxWithTags := &hook.Context{
		Command:   "todo.add",
		Args:      []string{"buy milk"},
		Flags:     map[string]string{"tags": "groceries"},
		Timestamp: "2026-01-01T00:00:00Z",
	}
	result2, err := transformHandler(ctxWithTags)
	if err != nil {
		t.Fatalf("prevalidate hook with tags error: %v", err)
	}
	if result2.Flags["tags"] != "groceries" {
		t.Errorf("expected tags='groceries', got %q", result2.Flags["tags"])
	}

	// Postexec hook — should pass through unchanged
	postHandler := pluginHookHandler(installedBin, "postexec", "transform", 5e9, got.Manifest.Permissions)
	result3, err := postHandler(ctx)
	if err != nil {
		t.Fatalf("postexec hook error: %v", err)
	}
	if result3.Command != "todo.add" {
		t.Errorf("postexec result command = %q, want %q", result3.Command, "todo.add")
	}

	// Lifecycle
	if err := SendLifecycleEvent(got, "health"); err != nil {
		t.Fatalf("SendLifecycleEvent(health) error: %v", err)
	}

	// Remove
	if err := Remove("tag-enforcer"); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}
}

// TestExampleHookScripts validates that the example hook scripts in
// docs/examples/hooks/ are properly formatted and executable.
func TestExampleHookScripts(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	hooksDir := filepath.Join(repoRoot, "docs", "examples", "hooks")

	entries, err := os.ReadDir(hooksDir)
	if err != nil {
		t.Fatalf("reading hooks dir: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("no example hook scripts found")
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(hooksDir, name)

			// Verify the file is executable
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat: %v", err)
			}
			if info.Mode()&0o111 == 0 {
				t.Errorf("%s is not executable", name)
			}

			// Verify the filename follows the convention: <pattern>.<stage>.<ext>
			parts := strings.Split(name, ".")
			if len(parts) < 3 {
				t.Errorf("%s: expected at least 3 dot-separated parts (pattern.stage.ext)", name)
				return
			}

			// Parse right-to-left: ext, stage, remainder is pattern
			ext := parts[len(parts)-1]
			stage := parts[len(parts)-2]

			if ext == "" {
				t.Errorf("%s: empty extension", name)
			}

			validStages := map[string]bool{
				"prevalidate": true,
				"preexec":     true,
				"postexec":    true,
				"notify":      true,
			}
			if !validStages[stage] {
				t.Errorf("%s: stage %q is not valid", name, stage)
			}

			// Verify it has a shebang line
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading: %v", err)
			}
			content := string(data)
			if !strings.HasPrefix(content, "#!") {
				t.Errorf("%s: missing shebang line", name)
			}
		})
	}
}

// TestExampleManifestHookInvocation verifies that each example plugin's hooks
// can be serialized into valid Invocation JSON matching the protocol spec.
func TestExampleManifestHookInvocation(t *testing.T) {
	base := examplesDir(t)
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatalf("reading examples dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			manifestPath := filepath.Join(base, name, "mine-plugin.toml")
			m, err := ParseManifest(manifestPath)
			if err != nil {
				t.Fatalf("ParseManifest error: %v", err)
			}

			for i, h := range m.Hooks {
				inv := Invocation{
					ProtocolVersion: ProtocolVersion,
					Type:            InvocationHook,
					Stage:           h.Stage,
					Mode:            h.Mode,
					Context: &hook.Context{
						Command:   h.Command,
						Args:      []string{"test"},
						Flags:     map[string]string{},
						Timestamp: "2026-01-01T00:00:00Z",
					},
				}

				data, err := json.Marshal(inv)
				if err != nil {
					t.Errorf("hooks[%d]: marshal error: %v", i, err)
					continue
				}

				var roundtrip Invocation
				if err := json.Unmarshal(data, &roundtrip); err != nil {
					t.Errorf("hooks[%d]: unmarshal error: %v", i, err)
					continue
				}

				if roundtrip.Stage != h.Stage {
					t.Errorf("hooks[%d]: stage = %q, want %q", i, roundtrip.Stage, h.Stage)
				}
				if roundtrip.Mode != h.Mode {
					t.Errorf("hooks[%d]: mode = %q, want %q", i, roundtrip.Mode, h.Mode)
				}
			}
		})
	}
}
