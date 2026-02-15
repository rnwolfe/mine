package craft

import (
	"embed"
	"strings"
)

// readTemplate reads an embedded template file.
func readTemplate(fs embed.FS, path string) string {
	data, err := fs.ReadFile(path)
	if err != nil {
		panic("missing embedded template: " + path)
	}
	return string(data)
}

func builtinRecipes() []*Recipe {
	return []*Recipe{
		goRecipe(),
		nodeRecipe(),
		pythonRecipe(),
		rustRecipe(),
		dockerRecipe(),
		githubCIRecipe(),
	}
}

func goRecipe() *Recipe {
	return &Recipe{
		Category:    "dev",
		Name:        "go",
		Description: "Bootstrap a Go project",
		Aliases:     []string{"golang"},
		SkipFile:    "go.mod",
		Files: []FileTemplate{
			{
				Path:         "main.go",
				Content:      readTemplate(embedded, "templates/dev/go/main.go.tmpl"),
				SkipIfExists: true,
			},
			{
				Path:         "Makefile",
				Content:      readTemplate(embedded, "templates/dev/go/Makefile.tmpl"),
				SkipIfExists: true,
			},
		},
		PostCommands: []PostCommand{
			{
				Name:        "go",
				Args:        []string{"mod", "init", "{{.Dir}}"},
				Description: "Initializing Go module",
			},
		},
	}
}

func nodeRecipe() *Recipe {
	return &Recipe{
		Category:    "dev",
		Name:        "node",
		Description: "Bootstrap a Node.js project",
		Aliases:     []string{"nodejs", "js"},
		SkipFile:    "package.json",
		PostCommands: []PostCommand{
			{
				Name:        "npm",
				Args:        []string{"init", "-y"},
				Description: "Initializing Node.js project",
			},
		},
	}
}

func pythonRecipe() *Recipe {
	return &Recipe{
		Category:    "dev",
		Name:        "python",
		Description: "Bootstrap a Python project",
		Aliases:     []string{"py"},
		SkipFile:    "pyproject.toml",
		Files: []FileTemplate{
			{
				Path:         "pyproject.toml",
				Content:      readTemplate(embedded, "templates/dev/python/pyproject.toml.tmpl"),
				SkipIfExists: true,
			},
		},
		PostCommands: []PostCommand{
			{
				Name:        "python3",
				Args:        []string{"-m", "venv", ".venv"},
				Description: "Creating virtual environment",
				Optional:    true,
			},
		},
	}
}

func rustRecipe() *Recipe {
	return &Recipe{
		Category:    "dev",
		Name:        "rust",
		Description: "Bootstrap a Rust project with Cargo",
		Aliases:     []string{"rs"},
		SkipFile:    "Cargo.toml",
		Files: []FileTemplate{
			{
				Path:         "Cargo.toml",
				Content:      readTemplate(embedded, "templates/dev/rust/Cargo.toml.tmpl"),
				SkipIfExists: true,
			},
			{
				Path:         "src/main.rs",
				Content:      readTemplate(embedded, "templates/dev/rust/main.rs.tmpl"),
				SkipIfExists: true,
			},
			{
				Path:         "Makefile",
				Content:      readTemplate(embedded, "templates/dev/rust/Makefile.tmpl"),
				SkipIfExists: true,
			},
			{
				Path:         ".gitignore",
				Content:      readTemplate(embedded, "templates/dev/rust/gitignore.tmpl"),
				SkipIfExists: true,
			},
		},
	}
}

func dockerRecipe() *Recipe {
	return &Recipe{
		Category:    "dev",
		Name:        "docker",
		Description: "Add Dockerfile and docker-compose",
		Aliases:     []string{"container"},
		Files: []FileTemplate{
			{
				Path:         "Dockerfile",
				Content:      readTemplate(embedded, "templates/dev/docker/Dockerfile.tmpl"),
				SkipIfExists: true,
			},
			{
				Path:         "docker-compose.yml",
				Content:      readTemplate(embedded, "templates/dev/docker/docker-compose.yml.tmpl"),
				SkipIfExists: true,
			},
			{
				Path:         ".dockerignore",
				Content:      readTemplate(embedded, "templates/dev/docker/dockerignore.tmpl"),
				SkipIfExists: true,
			},
		},
	}
}

func githubCIRecipe() *Recipe {
	return &Recipe{
		Category:    "ci",
		Name:        "github",
		Description: "GitHub Actions CI/CD workflow",
		Aliases:     []string{"gh", "github-actions"},
		Files: []FileTemplate{
			{
				Path:         ".github/workflows/ci.yml",
				Content:      readTemplate(embedded, "templates/ci/github/ci.yml.tmpl"),
				SkipIfExists: true,
			},
		},
	}
}

// TemplateArgs renders post command args through the template engine.
func TemplateArgs(args []string, data TemplateData) []string {
	out := make([]string, len(args))
	for i, arg := range args {
		if strings.Contains(arg, "{{") {
			rendered, err := renderTemplate("arg", arg, data)
			if err == nil {
				out[i] = rendered
				continue
			}
		}
		out[i] = arg
	}
	return out
}
