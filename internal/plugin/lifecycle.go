package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

// PluginsRegistry tracks installed plugins.
type PluginsRegistry struct {
	Plugins []PluginEntry `toml:"plugins"`
}

// PluginEntry is a single entry in the plugins registry.
type PluginEntry struct {
	Name        string `toml:"name"`
	Version     string `toml:"version"`
	Source      string `toml:"source"` // e.g. "github.com/user/mine-plugin-foo"
	Dir         string `toml:"dir"`
	InstalledAt string `toml:"installed_at"`
	Enabled     bool   `toml:"enabled"`
}

// LoadRegistry reads the plugins registry from disk.
func LoadRegistry() (*PluginsRegistry, error) {
	path := PluginsConfigFile()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &PluginsRegistry{}, nil
		}
		return nil, fmt.Errorf("reading plugin registry: %w", err)
	}

	var reg PluginsRegistry
	if err := toml.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parsing plugin registry: %w", err)
	}
	return &reg, nil
}

// SaveRegistry writes the plugins registry to disk.
func SaveRegistry(reg *PluginsRegistry) error {
	path := PluginsConfigFile()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return toml.NewEncoder(f).Encode(reg)
}

// Install installs a plugin from a local directory containing mine-plugin.toml.
// For GitHub installs, the caller should clone/download first, then call this.
func Install(sourceDir, source string) (*InstalledPlugin, error) {
	manifestPath := filepath.Join(sourceDir, "mine-plugin.toml")
	manifest, err := ParseManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("invalid plugin: %w", err)
	}

	// Create plugin directory
	pluginDir := filepath.Join(PluginsDir(), manifest.Plugin.Name)
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating plugin dir: %w", err)
	}

	// Copy manifest
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "mine-plugin.toml"), data, 0o644); err != nil {
		return nil, err
	}

	// Copy entrypoint binary if it exists in source
	entrypoint := manifest.Entrypoint()
	srcBin := filepath.Join(sourceDir, entrypoint)
	if info, err := os.Stat(srcBin); err == nil && !info.IsDir() {
		binData, err := os.ReadFile(srcBin)
		if err != nil {
			return nil, fmt.Errorf("reading plugin binary: %w", err)
		}
		destBin := filepath.Join(pluginDir, entrypoint)
		if err := os.WriteFile(destBin, binData, 0o755); err != nil {
			return nil, fmt.Errorf("writing plugin binary: %w", err)
		}
	}

	// Register in plugins.toml
	reg, err := LoadRegistry()
	if err != nil {
		return nil, err
	}

	// Remove existing entry if upgrading
	filtered := make([]PluginEntry, 0, len(reg.Plugins))
	for _, p := range reg.Plugins {
		if p.Name != manifest.Plugin.Name {
			filtered = append(filtered, p)
		}
	}

	filtered = append(filtered, PluginEntry{
		Name:        manifest.Plugin.Name,
		Version:     manifest.Plugin.Version,
		Source:      source,
		Dir:         pluginDir,
		InstalledAt: time.Now().UTC().Format(time.RFC3339),
		Enabled:     true,
	})
	reg.Plugins = filtered

	if err := SaveRegistry(reg); err != nil {
		return nil, fmt.Errorf("saving registry: %w", err)
	}

	return &InstalledPlugin{
		Manifest:    *manifest,
		Dir:         pluginDir,
		InstalledAt: time.Now(),
		Enabled:     true,
	}, nil
}

// Remove uninstalls a plugin by name.
func Remove(name string) error {
	reg, err := LoadRegistry()
	if err != nil {
		return err
	}

	var found bool
	var pluginDir string
	filtered := make([]PluginEntry, 0, len(reg.Plugins))
	for _, p := range reg.Plugins {
		if p.Name == name {
			found = true
			pluginDir = p.Dir
		} else {
			filtered = append(filtered, p)
		}
	}

	if !found {
		return fmt.Errorf("plugin %q not found", name)
	}

	// Remove plugin directory
	if pluginDir != "" {
		os.RemoveAll(pluginDir)
	}

	reg.Plugins = filtered
	return SaveRegistry(reg)
}

// List returns all installed plugins with their manifests.
func List() ([]InstalledPlugin, error) {
	reg, err := LoadRegistry()
	if err != nil {
		return nil, err
	}

	var plugins []InstalledPlugin
	for _, entry := range reg.Plugins {
		manifestPath := filepath.Join(entry.Dir, "mine-plugin.toml")
		manifest, err := ParseManifest(manifestPath)
		if err != nil {
			// Plugin is broken but still listed
			plugins = append(plugins, InstalledPlugin{
				Manifest: Manifest{Plugin: PluginMeta{
					Name:    entry.Name,
					Version: entry.Version,
				}},
				Dir:     entry.Dir,
				Enabled: entry.Enabled,
			})
			continue
		}

		installedAt, _ := time.Parse(time.RFC3339, entry.InstalledAt)
		plugins = append(plugins, InstalledPlugin{
			Manifest:    *manifest,
			Dir:         entry.Dir,
			InstalledAt: installedAt,
			Enabled:     entry.Enabled,
		})
	}

	return plugins, nil
}

// Get returns a single installed plugin by name.
func Get(name string) (*InstalledPlugin, error) {
	plugins, err := List()
	if err != nil {
		return nil, err
	}

	for _, p := range plugins {
		if p.Manifest.Plugin.Name == name {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("plugin %q not found", name)
}
