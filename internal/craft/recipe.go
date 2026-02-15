// Package craft provides a data-driven project scaffolding system.
//
// Recipes are template sets embedded in the binary or loaded from user-local
// directories (~/.config/mine/recipes/). Each recipe defines files to create,
// commands to run, and metadata for display.
package craft

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/rnwolfe/mine/internal/config"
)

//go:embed templates/*
var embedded embed.FS

// Recipe describes a scaffolding recipe.
type Recipe struct {
	// Category groups recipes (e.g. "dev", "ci").
	Category string
	// Name is the recipe identifier (e.g. "go", "rust", "github").
	Name string
	// Description is a short human-readable summary.
	Description string
	// Aliases are alternative names that resolve to this recipe.
	Aliases []string
	// SkipFile is a file whose presence means the project is already set up.
	SkipFile string
	// Files maps relative output paths to template content.
	Files []FileTemplate
	// PostCommands are shell commands to run after files are written.
	PostCommands []PostCommand
}

// FileTemplate is a single file to generate.
type FileTemplate struct {
	// Path is the output path relative to the working directory.
	Path string
	// Content is a Go text/template string.
	Content string
	// SkipIfExists skips this file if it already exists.
	SkipIfExists bool
}

// PostCommand is a command to run after scaffolding.
type PostCommand struct {
	// Name is the binary to run.
	Name string
	// Args are the command arguments.
	Args []string
	// Description is shown to the user before running.
	Description string
	// Optional means failure is not fatal.
	Optional bool
}

// TemplateData is passed to every template during rendering.
type TemplateData struct {
	// Dir is the name of the current working directory.
	Dir string
}

// Registry holds all known recipes.
type Registry struct {
	recipes map[string]*Recipe
}

// NewRegistry creates a registry pre-loaded with built-in recipes.
func NewRegistry() *Registry {
	r := &Registry{recipes: make(map[string]*Recipe)}
	for _, recipe := range builtinRecipes() {
		r.Register(recipe)
	}
	return r
}

// Register adds a recipe to the registry.
func (r *Registry) Register(recipe *Recipe) {
	key := recipe.Category + "/" + recipe.Name
	r.recipes[key] = recipe
	for _, alias := range recipe.Aliases {
		r.recipes[recipe.Category+"/"+alias] = recipe
	}
}

// Get returns a recipe by category and name.
func (r *Registry) Get(category, name string) (*Recipe, bool) {
	recipe, ok := r.recipes[category+"/"+strings.ToLower(name)]
	return recipe, ok
}

// List returns all unique recipes sorted by category then name.
func (r *Registry) List() []*Recipe {
	seen := make(map[string]bool)
	var out []*Recipe
	for _, recipe := range r.recipes {
		key := recipe.Category + "/" + recipe.Name
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, recipe)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Category != out[j].Category {
			return out[i].Category < out[j].Category
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// LoadUserRecipes scans ~/.config/mine/recipes/ for user-defined recipes.
// Each subdirectory is treated as a recipe; its metadata is derived from the
// directory name and its files are used as templates.
func (r *Registry) LoadUserRecipes() error {
	recipesDir := filepath.Join(config.GetPaths().ConfigDir, "recipes")
	if _, err := os.Stat(recipesDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(recipesDir)
	if err != nil {
		return fmt.Errorf("reading recipes directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		recipe, err := loadUserRecipe(recipesDir, entry.Name())
		if err != nil {
			return fmt.Errorf("loading recipe %s: %w", entry.Name(), err)
		}
		if recipe != nil {
			r.Register(recipe)
		}
	}
	return nil
}

// loadUserRecipe loads a recipe from a user directory.
// The directory name determines category/name (e.g. "dev-rust" → dev/rust).
func loadUserRecipe(baseDir, dirName string) (*Recipe, error) {
	parts := strings.SplitN(dirName, "-", 2)
	if len(parts) != 2 {
		return nil, nil // skip directories without category-name format
	}

	category := strings.TrimSpace(parts[0])
	name := strings.TrimSpace(parts[1])
	if category == "" || name == "" {
		return nil, nil // skip directories with empty category or name
	}
	recipeDir := filepath.Join(baseDir, dirName)

	var files []FileTemplate
	err := filepath.WalkDir(recipeDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(recipeDir, path)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		files = append(files, FileTemplate{
			Path:         relPath,
			Content:      string(content),
			SkipIfExists: true,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &Recipe{
		Category:    category,
		Name:        name,
		Description: fmt.Sprintf("User recipe: %s/%s", category, name),
		Files:       files,
	}, nil
}

// Execute runs a recipe, rendering templates and writing files.
func Execute(recipe *Recipe, data TemplateData) ([]string, error) {
	// Check if project is already set up
	if recipe.SkipFile != "" {
		if _, err := os.Stat(recipe.SkipFile); err == nil {
			return nil, fmt.Errorf("already initialized (%s exists)", recipe.SkipFile)
		}
	}

	var created []string
	for _, ft := range recipe.Files {
		if ft.SkipIfExists {
			if _, err := os.Stat(ft.Path); err == nil {
				continue
			}
		}

		content, err := renderTemplate(ft.Path, ft.Content, data)
		if err != nil {
			return created, fmt.Errorf("rendering %s: %w", ft.Path, err)
		}

		// Ensure parent directory exists
		dir := filepath.Dir(ft.Path)
		if dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return created, fmt.Errorf("creating directory %s: %w", dir, err)
			}
		}

		if err := os.WriteFile(ft.Path, []byte(content), 0o644); err != nil {
			return created, fmt.Errorf("writing %s: %w", ft.Path, err)
		}
		created = append(created, ft.Path)
	}

	return created, nil
}

func renderTemplate(name, content string, data TemplateData) (string, error) {
	tmpl, err := template.New(name).Parse(content)
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// CurrentDir returns a TemplateData populated with the current directory name.
func CurrentDir() TemplateData {
	cwd, _ := os.Getwd()
	return TemplateData{Dir: filepath.Base(cwd)}
}
