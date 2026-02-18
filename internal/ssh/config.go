package ssh

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Host represents a single SSH host entry from ~/.ssh/config.
type Host struct {
	Alias    string
	Hostname string
	User     string
	Port     string
	KeyFile  string
	// raw lines for this block (preserves comments inside the block)
	raw []string
}

// FilterValue implements tui.Item.
func (h Host) FilterValue() string { return h.Alias }

// Title implements tui.Item.
func (h Host) Title() string { return h.Alias }

// Description implements tui.Item.
func (h Host) Description() string {
	parts := []string{}
	if h.Hostname != "" {
		parts = append(parts, h.Hostname)
	}
	if h.User != "" {
		parts = append(parts, h.User+"@")
	}
	if h.Port != "" && h.Port != "22" {
		parts = append(parts, ":"+h.Port)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "  ")
}

// ConfigPath returns the path to ~/.ssh/config.
func ConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", ".ssh", "config")
	}
	return filepath.Join(home, ".ssh", "config")
}

// ReadHosts parses ~/.ssh/config and returns all non-wildcard Host entries.
func ReadHosts() ([]Host, error) {
	return ReadHostsFrom(ConfigPath())
}

// ReadHostsFrom parses the given SSH config file and returns all non-wildcard Host entries.
func ReadHostsFrom(path string) ([]Host, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading ssh config: %w", err)
	}
	defer f.Close()

	return parseConfig(f)
}

// parseConfig parses an SSH config from a reader.
func parseConfig(r io.Reader) ([]Host, error) {
	scanner := bufio.NewScanner(r)
	var hosts []Host
	var current *Host

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip global comments and blank lines between blocks
		if current == nil && (trimmed == "" || strings.HasPrefix(trimmed, "#")) {
			continue
		}

		key, value := parseKeyValue(trimmed)
		upperKey := strings.ToUpper(key)

		switch upperKey {
		case "HOST":
			// Flush previous block
			if current != nil {
				hosts = append(hosts, *current)
			}
			// SSH config allows multiple patterns on one Host line.
			// Use the first non-wildcard alias as canonical; skip entire entry if
			// any alias is a wildcard pattern.
			aliases := strings.Fields(value)
			if len(aliases) == 0 {
				current = nil
				continue
			}
			skip := false
			for _, alias := range aliases {
				if strings.ContainsAny(alias, "?*") {
					skip = true
					break
				}
			}
			if skip {
				current = nil
				continue
			}
			current = &Host{Alias: aliases[0]}

		case "INCLUDE":
			// Include directives are not expanded; hosts in included files are not listed.
			continue

		default:
			if current == nil {
				continue
			}
			current.raw = append(current.raw, line)
			switch upperKey {
			case "HOSTNAME":
				current.Hostname = value
			case "USER":
				current.User = value
			case "PORT":
				current.Port = value
			case "IDENTITYFILE":
				current.KeyFile = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning ssh config: %w", err)
	}

	// Flush last block
	if current != nil {
		hosts = append(hosts, *current)
	}

	return hosts, nil
}

// parseKeyValue splits "Key Value" or "Key=Value" into key, value.
// Returns empty strings for comment lines.
func parseKeyValue(line string) (string, string) {
	if line == "" || strings.HasPrefix(line, "#") {
		return "", ""
	}
	// SSH config allows "Key Value" or "Key=Value"
	line = strings.TrimSpace(line)
	idx := strings.IndexAny(line, " \t=")
	if idx < 0 {
		return line, ""
	}
	key := line[:idx]
	rest := strings.TrimLeft(line[idx:], " \t=")
	// Strip inline comments: '#' starts a comment when preceded by whitespace.
	runes := []rune(rest)
	commentIdx := -1
	prevIsWS := true // treat start-of-string as whitespace for leading '#' comments
	for i, r := range runes {
		if r == '#' && prevIsWS {
			commentIdx = i
			break
		}
		prevIsWS = r == ' ' || r == '\t'
	}
	if commentIdx >= 0 {
		rest = strings.TrimSpace(string(runes[:commentIdx]))
	}
	return key, rest
}

// FindHost returns the first host matching alias (case-insensitive).
func FindHost(alias string, hosts []Host) (Host, error) {
	lower := strings.ToLower(alias)
	for _, h := range hosts {
		if strings.ToLower(h.Alias) == lower {
			return h, nil
		}
	}
	return Host{}, fmt.Errorf("ssh host %q not found", alias)
}

// AppendHost adds a new Host block to ~/.ssh/config.
func AppendHost(h Host) error {
	return AppendHostTo(ConfigPath(), h)
}

// AppendHostTo adds a new Host block to the given config file.
// Uses a write-to-temp + atomic rename to prevent partial writes on failure.
func AppendHostTo(path string, h Host) error {
	if err := ensureSSHDir(filepath.Dir(path)); err != nil {
		return err
	}

	// Read existing content (tolerate missing file).
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading ssh config: %w", err)
	}

	block := formatHostBlock(h)
	newContent := append(existing, []byte(block)...)

	// Write atomically: temp file in the same directory, then rename.
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".ssh_config_tmp_*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(newContent); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("writing ssh config: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("setting ssh config permissions: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("updating ssh config: %w", err)
	}
	return nil
}

// formatHostBlock formats a Host into a config block string.
func formatHostBlock(h Host) string {
	var sb strings.Builder
	sb.WriteString("\nHost " + h.Alias + "\n")
	if h.Hostname != "" {
		sb.WriteString("    HostName " + h.Hostname + "\n")
	}
	if h.User != "" {
		sb.WriteString("    User " + h.User + "\n")
	}
	if h.Port != "" {
		sb.WriteString("    Port " + h.Port + "\n")
	}
	if h.KeyFile != "" {
		sb.WriteString("    IdentityFile " + h.KeyFile + "\n")
	}
	return sb.String()
}

// RemoveHost removes a host block by alias from ~/.ssh/config.
func RemoveHost(alias string) error {
	return RemoveHostFrom(ConfigPath(), alias)
}

// RemoveHostFrom removes a host block by alias from the given config file.
func RemoveHostFrom(path string, alias string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("ssh host %q not found (no config file)", alias)
		}
		return fmt.Errorf("reading ssh config: %w", err)
	}

	result, removed := removeHostBlock(string(data), alias)
	if !removed {
		return fmt.Errorf("ssh host %q not found in config", alias)
	}

	if err := os.WriteFile(path, []byte(result), 0o600); err != nil {
		return fmt.Errorf("writing ssh config: %w", err)
	}
	return nil
}

// removeHostBlock removes a Host block from a config string.
// Returns the modified string and whether a block was removed.
func removeHostBlock(content string, alias string) (string, bool) {
	lines := strings.Split(content, "\n")
	var out []string
	inBlock := false
	removed := false
	lower := strings.ToLower(alias)

	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		key, value := parseKeyValue(trimmed)

		if strings.ToUpper(key) == "HOST" {
			// Multi-alias Host lines (e.g. "Host web web-prod") — check each alias.
			match := false
			for _, a := range strings.Fields(value) {
				if strings.ToLower(a) == lower {
					match = true
					break
				}
			}
			if match {
				// Skip this block until next Host or EOF
				inBlock = true
				removed = true
				i++
				continue
			}
			inBlock = false
		}

		if inBlock {
			i++
			continue
		}

		out = append(out, line)
		i++
	}

	// Trim trailing blank lines
	result := strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n"
	return result, removed
}

// ensureSSHDir creates the ~/.ssh directory with correct permissions.
func ensureSSHDir(dir string) error {
	return os.MkdirAll(dir, 0o700)
}

