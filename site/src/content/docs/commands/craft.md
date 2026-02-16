---
title: mine craft
description: Scaffold projects and bootstrap dev tool configurations
---

Data-driven project scaffolding. Recipes are embedded templates — extensible via user-local recipes.

## Bootstrap a project

```bash
mine craft dev go       # Go project with module, main.go, Makefile
mine craft dev node     # Node.js with package.json
mine craft dev python   # Python with pyproject.toml and .venv
mine craft dev rust     # Rust project with Cargo.toml, src/main.rs, Makefile
mine craft dev docker   # Dockerfile, docker-compose.yml, .dockerignore
```

## Set up git

```bash
mine craft git          # git init + .gitignore
```

## Generate CI/CD templates

```bash
mine craft ci github    # GitHub Actions CI workflow (.github/workflows/ci.yml)
```

## List available recipes

```bash
mine craft list         # show all recipes with details and aliases
```

## User-local recipes

Drop template files in `~/.config/mine/recipes/` using the naming convention `<category>-<name>/` (e.g. `dev-mytemplate/`). Files inside are Go `text/template` files with `{{.Dir}}` available as the project directory name.

```
~/.config/mine/recipes/
└── dev-myframework/
    ├── main.go
    └── config.yaml
```

Then run:

```bash
mine craft dev myframework
```
