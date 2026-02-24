package agents

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ContentItem represents a single piece of content in the agents store.
type ContentItem struct {
	Name        string // display name (e.g., "my-skill", "deploy.md")
	Description string // short description extracted from the content
	Path        string // absolute path in the store
}

// ListResult holds all categorized content found in the agents store.
type ListResult struct {
	Skills       []ContentItem
	Commands     []ContentItem
	Agents       []ContentItem
	Rules        []ContentItem
	Instructions []ContentItem
	Settings     []ContentItem
}

// validTypes is the set of accepted Type values for ListOptions.
var validTypes = map[string]bool{
	"skills":       true,
	"commands":     true,
	"agents":       true,
	"rules":        true,
	"instructions": true,
	"settings":     true,
}

// ListOptions controls List behavior.
type ListOptions struct {
	// Type filters to a single content type. Valid values:
	// "skills", "commands", "agents", "rules", "instructions", "settings", "".
	// An empty string returns all types.
	Type string
}

// List reads the agents store and returns a categorized inventory of content.
// If opts.Type is set, only that category is populated; all others are nil.
// Returns an error if opts.Type is set to an unknown value.
func List(opts ListOptions) (*ListResult, error) {
	dir := Dir()
	result := &ListResult{}

	t := strings.ToLower(strings.TrimSpace(opts.Type))
	if t != "" && !validTypes[t] {
		return nil, fmt.Errorf("unknown type %q — valid types: skills, commands, agents, rules, instructions, settings", opts.Type)
	}

	if t == "" || t == "skills" {
		items, err := listSkills(dir)
		if err != nil {
			return nil, err
		}
		result.Skills = items
	}

	if t == "" || t == "commands" {
		items, err := listMarkdownFiles(filepath.Join(dir, "commands"))
		if err != nil {
			return nil, fmt.Errorf("listing commands: %w", err)
		}
		result.Commands = items
	}

	if t == "" || t == "agents" {
		items, err := listMarkdownFiles(filepath.Join(dir, "agents"))
		if err != nil {
			return nil, fmt.Errorf("listing agents: %w", err)
		}
		result.Agents = items
	}

	if t == "" || t == "rules" {
		items, err := listMarkdownFiles(filepath.Join(dir, "rules"))
		if err != nil {
			return nil, fmt.Errorf("listing rules: %w", err)
		}
		result.Rules = items
	}

	if t == "" || t == "instructions" {
		items, err := listMarkdownFiles(filepath.Join(dir, "instructions"))
		if err != nil {
			return nil, fmt.Errorf("listing instructions: %w", err)
		}
		result.Instructions = items
	}

	if t == "" || t == "settings" {
		items, err := listSettings(filepath.Join(dir, "settings"))
		if err != nil {
			return nil, fmt.Errorf("listing settings: %w", err)
		}
		result.Settings = items
	}

	return result, nil
}

// listSkills reads skills/<name>/SKILL.md for each subdirectory and extracts
// the description from the YAML frontmatter.
func listSkills(storeDir string) ([]ContentItem, error) {
	skillsDir := filepath.Join(storeDir, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var items []ContentItem
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		skillMDPath := filepath.Join(skillsDir, name, "SKILL.md")
		desc := parseFrontmatterDescription(skillMDPath)
		items = append(items, ContentItem{
			Name:        name,
			Description: desc,
			Path:        filepath.Join(skillsDir, name),
		})
	}
	return items, nil
}

// listMarkdownFiles reads *.md files in the given directory and extracts a
// one-line description from the first non-empty, non-heading line.
func listMarkdownFiles(dir string) ([]ContentItem, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var items []ContentItem
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		path := filepath.Join(dir, name)
		desc := parseMarkdownDescription(path)
		// Strip .md extension for display name.
		displayName := strings.TrimSuffix(name, ".md")
		items = append(items, ContentItem{
			Name:        displayName,
			Description: desc,
			Path:        path,
		})
	}
	return items, nil
}

// listSettings reads *.json files in the settings directory.
func listSettings(dir string) ([]ContentItem, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var items []ContentItem
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		path := filepath.Join(dir, name)
		// Strip .json extension for display name (consistent with listMarkdownFiles).
		displayName := strings.TrimSuffix(name, ".json")
		desc := displayName + " agent settings"
		items = append(items, ContentItem{
			Name:        displayName,
			Description: desc,
			Path:        path,
		})
	}
	return items, nil
}

// parseFrontmatterDescription reads a SKILL.md file and extracts the description
// field from its YAML frontmatter.
//
// Supports both single-line and folded/literal block scalars (> and |):
//
//	description: single line value
//	description: >
//	  multi-line folded
//	  continues here
func parseFrontmatterDescription(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	// Skip until the first --- delimiter.
	inFrontmatter := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			inFrontmatter = true
			break
		}
	}
	if !inFrontmatter {
		return ""
	}

	// Parse frontmatter lines.
	var descLines []string
	collectingDesc := false

	for scanner.Scan() {
		line := scanner.Text()

		// End of frontmatter.
		if line == "---" {
			break
		}

		if collectingDesc {
			// If we're collecting a multi-line description, check if this line
			// is indented (continuation) or a new key (done).
			if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
				descLines = append(descLines, strings.TrimSpace(line))
				continue
			}
			// New key or blank line: stop collecting.
			break
		}

		// Look for the description key.
		if !strings.HasPrefix(line, "description:") {
			continue
		}

		value := strings.TrimPrefix(line, "description:")
		value = strings.TrimSpace(value)

		if value == ">" || value == "|" {
			// Multi-line scalar — collect subsequent indented lines.
			collectingDesc = true
			continue
		}

		if value == "" {
			// Bare "description:" with no value is a null scalar, not a block scalar.
			return ""
		}

		// Single-line value.
		return value
	}

	// Combine multi-line description lines.
	if len(descLines) > 0 {
		return strings.Join(descLines, " ")
	}

	return ""
}

// parseMarkdownDescription reads a markdown file and returns the first
// non-empty, non-heading line as a brief description.
func parseMarkdownDescription(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inFrontmatter := false
	frontmatterDone := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Track YAML frontmatter delimiters.
		// Once the closing --- is seen, stop toggling to avoid treating
		// thematic breaks in the body as frontmatter.
		if line == "---" && !frontmatterDone {
			inFrontmatter = !inFrontmatter
			if !inFrontmatter {
				frontmatterDone = true
			}
			continue
		}
		if inFrontmatter {
			continue
		}
		if line == "" {
			continue
		}
		// Skip heading lines (starting with #).
		if strings.HasPrefix(line, "#") {
			continue
		}
		return line
	}
	return ""
}

