package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseManifest(t *testing.T) {
	dir := t.TempDir()
	manifest := `
[plugin]
name = "test-plugin"
version = "0.1.0"
description = "A test plugin"
author = "tester"
protocol_version = "1.0.0"

[[hooks]]
command = "todo.add"
stage = "postexec"
mode = "notify"
timeout = "10s"

[[hooks]]
command = "todo.*"
stage = "preexec"
mode = "transform"

[[commands]]
name = "sync"
description = "Sync stuff"
args = "[--force]"

[permissions]
network = true
filesystem = ["~/notes"]
store = false
config_read = true
env_vars = ["MY_TOKEN"]
`

	path := filepath.Join(dir, "mine-plugin.toml")
	if err := os.WriteFile(path, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := ParseManifest(path)
	if err != nil {
		t.Fatalf("ParseManifest() error: %v", err)
	}

	if m.Plugin.Name != "test-plugin" {
		t.Errorf("Name = %q, want %q", m.Plugin.Name, "test-plugin")
	}
	if m.Plugin.Version != "0.1.0" {
		t.Errorf("Version = %q, want %q", m.Plugin.Version, "0.1.0")
	}
	if len(m.Hooks) != 2 {
		t.Errorf("Hooks len = %d, want 2", len(m.Hooks))
	}
	if len(m.Commands) != 1 {
		t.Errorf("Commands len = %d, want 1", len(m.Commands))
	}
	if !m.Permissions.Network {
		t.Error("Permissions.Network should be true")
	}
	if !m.Permissions.ConfigRead {
		t.Error("Permissions.ConfigRead should be true")
	}
	if len(m.Permissions.EnvVars) != 1 || m.Permissions.EnvVars[0] != "MY_TOKEN" {
		t.Errorf("Permissions.EnvVars = %v, want [MY_TOKEN]", m.Permissions.EnvVars)
	}
}

func TestParseManifest_Invalid(t *testing.T) {
	tests := []struct {
		name     string
		manifest string
	}{
		{"missing name", `[plugin]
version = "0.1.0"
description = "test"
author = "tester"
protocol_version = "1.0.0"`},
		{"missing version", `[plugin]
name = "test"
description = "test"
author = "tester"
protocol_version = "1.0.0"`},
		{"invalid hook stage", `[plugin]
name = "test"
version = "0.1.0"
description = "test"
author = "tester"
protocol_version = "1.0.0"
[[hooks]]
command = "todo.add"
stage = "invalid"
mode = "notify"`},
		{"invalid hook mode", `[plugin]
name = "test"
version = "0.1.0"
description = "test"
author = "tester"
protocol_version = "1.0.0"
[[hooks]]
command = "todo.add"
stage = "preexec"
mode = "invalid"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "mine-plugin.toml")
			os.WriteFile(path, []byte(tt.manifest), 0o644)

			_, err := ParseManifest(path)
			if err == nil {
				t.Error("expected error for invalid manifest")
			}
		})
	}
}

func TestEntrypoint(t *testing.T) {
	m := &Manifest{Plugin: PluginMeta{Name: "obsidian-sync"}}
	if got := m.Entrypoint(); got != "mine-plugin-obsidian-sync" {
		t.Errorf("Entrypoint() = %q, want %q", got, "mine-plugin-obsidian-sync")
	}

	m.Plugin.Entrypoint = "custom-binary"
	if got := m.Entrypoint(); got != "custom-binary" {
		t.Errorf("Entrypoint() = %q, want %q", got, "custom-binary")
	}
}

func TestPermissionSummary(t *testing.T) {
	perms := Permissions{
		Network:    true,
		Filesystem: []string{"~/notes", "~/docs"},
		Store:      true,
		EnvVars:    []string{"TOKEN"},
	}

	lines := PermissionSummary(perms)
	if len(lines) != 4 {
		t.Errorf("PermissionSummary() returned %d lines, want 4", len(lines))
	}
}

func TestPermissionSummary_NoPerms(t *testing.T) {
	perms := Permissions{}
	lines := PermissionSummary(perms)
	if len(lines) != 1 || lines[0] != "No special permissions required" {
		t.Errorf("PermissionSummary() = %v, want [No special permissions required]", lines)
	}
}

func TestHasEscalation(t *testing.T) {
	current := Permissions{
		Network:    false,
		Filesystem: []string{"~/notes"},
		EnvVars:    []string{"TOKEN"},
	}

	proposed := Permissions{
		Network:    true,
		Filesystem: []string{"~/notes", "~/secrets"},
		Store:      true,
		EnvVars:    []string{"TOKEN", "SECRET_KEY"},
	}

	escalations := HasEscalation(current, proposed)
	if len(escalations) != 4 {
		t.Errorf("HasEscalation() returned %d escalations, want 4: %v", len(escalations), escalations)
	}
}

func TestHasEscalation_None(t *testing.T) {
	perms := Permissions{Network: true, Store: true}
	escalations := HasEscalation(perms, perms)
	if len(escalations) != 0 {
		t.Errorf("HasEscalation() returned %d escalations, want 0", len(escalations))
	}
}

func TestRegistryLifecycle(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("XDG_DATA_HOME", dir)

	// Load empty registry
	reg, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error: %v", err)
	}
	if len(reg.Plugins) != 0 {
		t.Errorf("empty registry has %d plugins", len(reg.Plugins))
	}

	// Add a plugin
	reg.Plugins = append(reg.Plugins, PluginEntry{
		Name:    "test-plugin",
		Version: "0.1.0",
		Source:  "github.com/user/mine-plugin-test",
		Dir:     filepath.Join(dir, "mine", "plugins", "test-plugin"),
		Enabled: true,
	})

	if err := SaveRegistry(reg); err != nil {
		t.Fatalf("SaveRegistry() error: %v", err)
	}

	// Reload and verify
	reg2, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() reload error: %v", err)
	}
	if len(reg2.Plugins) != 1 {
		t.Fatalf("registry has %d plugins, want 1", len(reg2.Plugins))
	}
	if reg2.Plugins[0].Name != "test-plugin" {
		t.Errorf("plugin name = %q, want %q", reg2.Plugins[0].Name, "test-plugin")
	}
}

func TestInstallAndRemove(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("XDG_DATA_HOME", dir)

	// Create a mock plugin source directory
	srcDir := filepath.Join(dir, "source")
	os.MkdirAll(srcDir, 0o755)

	manifest := `[plugin]
name = "mock-plugin"
version = "1.0.0"
description = "A mock plugin for testing"
author = "tester"
protocol_version = "1.0.0"

[permissions]
network = false
`
	os.WriteFile(filepath.Join(srcDir, "mine-plugin.toml"), []byte(manifest), 0o644)

	// Install
	p, err := Install(srcDir, "github.com/user/mine-plugin-mock")
	if err != nil {
		t.Fatalf("Install() error: %v", err)
	}
	if p.Manifest.Plugin.Name != "mock-plugin" {
		t.Errorf("Name = %q, want %q", p.Manifest.Plugin.Name, "mock-plugin")
	}

	// List
	plugins, err := List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("List() returned %d plugins, want 1", len(plugins))
	}

	// Get
	got, err := Get("mock-plugin")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.Manifest.Plugin.Name != "mock-plugin" {
		t.Errorf("Get() name = %q, want %q", got.Manifest.Plugin.Name, "mock-plugin")
	}

	// Remove
	if err := Remove("mock-plugin"); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}

	plugins, err = List()
	if err != nil {
		t.Fatalf("List() after remove error: %v", err)
	}
	if len(plugins) != 0 {
		t.Errorf("List() after remove returned %d plugins, want 0", len(plugins))
	}
}

func TestRemoveNonexistent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("XDG_DATA_HOME", dir)

	err := Remove("nonexistent")
	if err == nil {
		t.Error("expected error removing nonexistent plugin")
	}
}

func TestAuditLog(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)

	if err := AuditLog("test-plugin", "install", "version=1.0.0"); err != nil {
		t.Fatalf("AuditLog() error: %v", err)
	}

	if err := AuditLog("test-plugin", "hook.execute", "command=todo.add"); err != nil {
		t.Fatalf("AuditLog() second call error: %v", err)
	}

	data, err := os.ReadFile(AuditLogPath())
	if err != nil {
		t.Fatalf("reading audit log: %v", err)
	}

	content := string(data)
	if len(content) == 0 {
		t.Error("audit log is empty")
	}
}
