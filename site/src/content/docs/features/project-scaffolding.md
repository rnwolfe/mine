---
title: Project Scaffolding
description: Bootstrap projects with built-in and custom recipes for Go, Node, Python, Rust, Docker, and CI
---

Stop copy-pasting boilerplate. `mine craft` bootstraps projects with opinionated templates for Go, Node.js, Python, Rust, Docker, and GitHub Actions CI — and you can add your own.

## Key Capabilities

- **Built-in recipes** — Go, Node.js, Python, Rust, Docker, and GitHub Actions CI
- **Custom recipes** — drop template directories into `~/.config/mine/recipes/` to add your own
- **Go templates** — recipes use `text/template` with `{{.Dir}}` for the project directory name
- **Git init** — standalone `mine craft git` sets up git with a `.gitignore`
- **Discoverable** — `mine craft list` shows all available recipes with aliases

## Quick Example

```bash
# Bootstrap a Go project
mkdir myapi && cd myapi
mine craft dev go

# Add GitHub Actions CI
mine craft ci github

# List all available recipes
mine craft list
```

## How It Works

Recipes are data-driven templates embedded in the binary. Each recipe category (`dev`, `ci`, `git`) groups related templates. Run `mine craft dev go` and you get a Go module, `main.go`, and `Makefile` in the current directory. Run `mine craft ci github` to add a GitHub Actions workflow.

For custom templates, create a directory in `~/.config/mine/recipes/` following the `<category>-<name>/` convention (e.g., `dev-myframework/`). Files inside are processed as Go templates. Then `mine craft dev myframework` just works.

## Learn More

See the [command reference](/commands/craft/) for all recipes, custom recipe details, and usage examples.
