package craft

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	recipes := r.List()
	if len(recipes) == 0 {
		t.Fatal("expected built-in recipes, got none")
	}

	// Verify we have the expected categories
	categories := make(map[string]bool)
	for _, recipe := range recipes {
		categories[recipe.Category] = true
	}
	for _, want := range []string{"dev", "ci"} {
		if !categories[want] {
			t.Errorf("expected category %q in registry", want)
		}
	}
}

func TestRegistryGet(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		category string
		name     string
		wantOk   bool
		wantName string
	}{
		{"dev", "go", true, "go"},
		{"dev", "golang", true, "go"},
		{"dev", "rust", true, "rust"},
		{"dev", "rs", true, "rust"},
		{"dev", "docker", true, "docker"},
		{"dev", "container", true, "docker"},
		{"dev", "node", true, "node"},
		{"dev", "nodejs", true, "node"},
		{"dev", "js", true, "node"},
		{"dev", "python", true, "python"},
		{"dev", "py", true, "python"},
		{"ci", "github", true, "github"},
		{"ci", "gh", true, "github"},
		{"ci", "github-actions", true, "github"},
		{"dev", "nonexistent", false, ""},
		{"ci", "nonexistent", false, ""},
	}

	for _, tt := range tests {
		recipe, ok := r.Get(tt.category, tt.name)
		if ok != tt.wantOk {
			t.Errorf("Get(%q, %q) ok = %v, want %v", tt.category, tt.name, ok, tt.wantOk)
			continue
		}
		if ok && recipe.Name != tt.wantName {
			t.Errorf("Get(%q, %q) name = %q, want %q", tt.category, tt.name, recipe.Name, tt.wantName)
		}
	}
}

func TestRegistryList(t *testing.T) {
	r := NewRegistry()
	recipes := r.List()

	// Check sorted order (by category, then name)
	for i := 1; i < len(recipes); i++ {
		prev := recipes[i-1].Category + "/" + recipes[i-1].Name
		curr := recipes[i].Category + "/" + recipes[i].Name
		if prev >= curr {
			t.Errorf("recipes not sorted: %s comes before %s", prev, curr)
		}
	}

	// Check no duplicates
	seen := make(map[string]bool)
	for _, r := range recipes {
		key := r.Category + "/" + r.Name
		if seen[key] {
			t.Errorf("duplicate recipe in list: %s", key)
		}
		seen[key] = true
	}
}

func TestRenderTemplate(t *testing.T) {
	content := "Hello {{.Dir}}!"
	result, err := renderTemplate("test", content, TemplateData{Dir: "myproject"})
	if err != nil {
		t.Fatalf("renderTemplate failed: %v", err)
	}
	if result != "Hello myproject!" {
		t.Errorf("got %q, want %q", result, "Hello myproject!")
	}
}

func TestRenderTemplateError(t *testing.T) {
	content := "Hello {{.Invalid"
	_, err := renderTemplate("test", content, TemplateData{Dir: "myproject"})
	if err == nil {
		t.Fatal("expected error for invalid template")
	}
}

func TestExecute(t *testing.T) {
	dir := t.TempDir()

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	recipe := &Recipe{
		Category:    "dev",
		Name:        "test",
		Description: "Test recipe",
		Files: []FileTemplate{
			{
				Path:    "hello.txt",
				Content: "Hello {{.Dir}}!",
			},
			{
				Path:    "sub/nested.txt",
				Content: "Nested in {{.Dir}}",
			},
		},
	}

	data := TemplateData{Dir: "testproj"}
	created, err := Execute(recipe, data)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(created) != 2 {
		t.Fatalf("expected 2 created files, got %d", len(created))
	}

	// Verify file contents
	content, err := os.ReadFile(filepath.Join(dir, "hello.txt"))
	if err != nil {
		t.Fatalf("reading hello.txt: %v", err)
	}
	if string(content) != "Hello testproj!" {
		t.Errorf("hello.txt content = %q, want %q", string(content), "Hello testproj!")
	}

	content, err = os.ReadFile(filepath.Join(dir, "sub/nested.txt"))
	if err != nil {
		t.Fatalf("reading sub/nested.txt: %v", err)
	}
	if string(content) != "Nested in testproj" {
		t.Errorf("sub/nested.txt content = %q, want %q", string(content), "Nested in testproj")
	}
}

func TestExecuteSkipExisting(t *testing.T) {
	dir := t.TempDir()

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Create existing file
	os.WriteFile("existing.txt", []byte("original"), 0o644)

	recipe := &Recipe{
		Category: "dev",
		Name:     "test",
		Files: []FileTemplate{
			{
				Path:         "existing.txt",
				Content:      "replaced",
				SkipIfExists: true,
			},
			{
				Path:    "new.txt",
				Content: "new content",
			},
		},
	}

	created, err := Execute(recipe, TemplateData{Dir: "test"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(created) != 1 {
		t.Fatalf("expected 1 created file, got %d: %v", len(created), created)
	}
	if created[0] != "new.txt" {
		t.Errorf("expected new.txt, got %s", created[0])
	}

	// Original file should be unchanged
	content, _ := os.ReadFile("existing.txt")
	if string(content) != "original" {
		t.Errorf("existing.txt was overwritten: %q", string(content))
	}
}

func TestExecuteSkipFile(t *testing.T) {
	dir := t.TempDir()

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Create the skip file
	os.WriteFile("go.mod", []byte("module test"), 0o644)

	recipe := &Recipe{
		Category: "dev",
		Name:     "go",
		SkipFile: "go.mod",
		Files: []FileTemplate{
			{Path: "main.go", Content: "package main"},
		},
	}

	_, err := Execute(recipe, TemplateData{Dir: "test"})
	if err == nil {
		t.Fatal("expected error when skip file exists")
	}
	if !strings.Contains(err.Error(), "already initialized") {
		t.Errorf("expected 'already initialized' error, got: %v", err)
	}
}

func TestTemplateArgs(t *testing.T) {
	data := TemplateData{Dir: "myproject"}

	args := []string{"mod", "init", "{{.Dir}}"}
	result := TemplateArgs(args, data)

	if len(result) != 3 {
		t.Fatalf("expected 3 args, got %d", len(result))
	}
	if result[0] != "mod" {
		t.Errorf("arg[0] = %q, want %q", result[0], "mod")
	}
	if result[2] != "myproject" {
		t.Errorf("arg[2] = %q, want %q", result[2], "myproject")
	}
}

func TestTemplateArgsNoTemplate(t *testing.T) {
	data := TemplateData{Dir: "myproject"}

	args := []string{"init", "-y"}
	result := TemplateArgs(args, data)

	if result[0] != "init" || result[1] != "-y" {
		t.Errorf("args should pass through unchanged: %v", result)
	}
}

func TestBuiltinRecipesHaveTemplates(t *testing.T) {
	recipes := builtinRecipes()
	for _, recipe := range recipes {
		for _, ft := range recipe.Files {
			if ft.Content == "" {
				t.Errorf("recipe %s/%s: file %s has empty template", recipe.Category, recipe.Name, ft.Path)
			}
		}
	}
}

func TestBuiltinRecipesHaveDescriptions(t *testing.T) {
	recipes := builtinRecipes()
	for _, recipe := range recipes {
		if recipe.Description == "" {
			t.Errorf("recipe %s/%s has no description", recipe.Category, recipe.Name)
		}
	}
}

func TestLoadUserRecipesEmptyDir(t *testing.T) {
	r := NewRegistry()
	countBefore := len(r.List())

	// Loading from nonexistent dir should not error
	err := r.LoadUserRecipes()
	if err != nil {
		t.Fatalf("LoadUserRecipes failed: %v", err)
	}

	countAfter := len(r.List())
	if countBefore != countAfter {
		t.Errorf("recipe count changed: %d -> %d", countBefore, countAfter)
	}
}

func TestUserRecipeLoading(t *testing.T) {
	dir := t.TempDir()

	// Create a user recipe directory structure
	recipeDir := filepath.Join(dir, "dev-custom")
	os.MkdirAll(recipeDir, 0o755)
	os.WriteFile(filepath.Join(recipeDir, "hello.txt"), []byte("Hello {{.Dir}}"), 0o644)

	recipe, err := loadUserRecipe(dir, "dev-custom")
	if err != nil {
		t.Fatalf("loadUserRecipe failed: %v", err)
	}
	if recipe == nil {
		t.Fatal("expected recipe, got nil")
	}
	if recipe.Category != "dev" {
		t.Errorf("category = %q, want %q", recipe.Category, "dev")
	}
	if recipe.Name != "custom" {
		t.Errorf("name = %q, want %q", recipe.Name, "custom")
	}
	if len(recipe.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(recipe.Files))
	}
}

func TestUserRecipeSkipsInvalidDirName(t *testing.T) {
	dir := t.TempDir()

	// Directory without category-name format should be skipped
	recipeDir := filepath.Join(dir, "invalid")
	os.MkdirAll(recipeDir, 0o755)

	recipe, err := loadUserRecipe(dir, "invalid")
	if err != nil {
		t.Fatalf("loadUserRecipe failed: %v", err)
	}
	if recipe != nil {
		t.Error("expected nil recipe for invalid dir name")
	}
}

func TestCurrentDir(t *testing.T) {
	data := CurrentDir()
	if data.Dir == "" {
		t.Error("CurrentDir returned empty Dir")
	}
}
