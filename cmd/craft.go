package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var craftCmd = &cobra.Command{
	Use:   "craft",
	Short: "Scaffold projects and bootstrap dev tools",
	Long:  `Quickly spin up new projects or install dev tool configurations.`,
	RunE:  runCraftHelp,
}

func init() {
	rootCmd.AddCommand(craftCmd)
	craftCmd.AddCommand(craftGitCmd)
	craftCmd.AddCommand(craftDevCmd)
}

var craftGitCmd = &cobra.Command{
	Use:   "git",
	Short: "Set up a git repository with best practices",
	RunE:  runCraftGit,
}

var craftDevCmd = &cobra.Command{
	Use:   "dev <tool>",
	Short: "Quick-start a dev tool (go, node, python, rust)",
	Args:  cobra.ExactArgs(1),
	RunE:  runCraftDev,
}

func runCraftHelp(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println(ui.Title.Render("  Craft — build and bootstrap"))
	fmt.Println()
	fmt.Println("  Available recipes:")
	fmt.Println()
	fmt.Printf("    %s   %s\n", ui.Accent.Render("mine craft git"), ui.Muted.Render("Set up git with .gitignore, hooks, etc."))
	fmt.Printf("    %s   %s\n", ui.Accent.Render("mine craft dev go"), ui.Muted.Render("Bootstrap a Go project"))
	fmt.Printf("    %s   %s\n", ui.Accent.Render("mine craft dev node"), ui.Muted.Render("Bootstrap a Node.js project"))
	fmt.Printf("    %s   %s\n", ui.Accent.Render("mine craft dev python"), ui.Muted.Render("Bootstrap a Python project"))
	fmt.Println()
	return nil
}

func runCraftGit(_ *cobra.Command, _ []string) error {
	cwd, _ := os.Getwd()

	// Check if already a git repo
	if _, err := os.Stat(".git"); err == nil {
		ui.Ok("Already a git repo: " + cwd)
		fmt.Println()
		return nil
	}

	// git init
	if err := runCmd("git", "init"); err != nil {
		return fmt.Errorf("git init failed: %w", err)
	}

	// Create .gitignore if missing
	if _, err := os.Stat(".gitignore"); os.IsNotExist(err) {
		gitignore := `# OS
.DS_Store
Thumbs.db

# IDE
.idea/
.vscode/
*.swp
*~

# Environment
.env
.env.*

# Build
dist/
build/
bin/
`
		if err := os.WriteFile(".gitignore", []byte(gitignore), 0o644); err != nil {
			return err
		}
		ui.Ok("Created .gitignore")
	}

	ui.Ok("Git repository initialized")
	fmt.Println()
	return nil
}

func runCraftDev(_ *cobra.Command, args []string) error {
	tool := strings.ToLower(args[0])

	switch tool {
	case "go", "golang":
		return craftGo()
	case "node", "nodejs", "js":
		return craftNode()
	case "python", "py":
		return craftPython()
	default:
		return fmt.Errorf("unknown tool %q — try: go, node, python", tool)
	}
}

func craftGo() error {
	cwd, _ := os.Getwd()
	dir := filepath.Base(cwd)

	// Check for existing go.mod
	if _, err := os.Stat("go.mod"); err == nil {
		ui.Ok("Go module already initialized")
		return nil
	}

	fmt.Println()
	ui.Puts("  Setting up Go project...")

	if err := runCmd("go", "mod", "init", dir); err != nil {
		return fmt.Errorf("go mod init: %w", err)
	}

	// Create main.go if missing
	if _, err := os.Stat("main.go"); os.IsNotExist(err) {
		main := `package main

import "fmt"

func main() {
	fmt.Println("Hello from ` + dir + `!")
}
`
		os.WriteFile("main.go", []byte(main), 0o644)
		ui.Ok("Created main.go")
	}

	// Create Makefile if missing
	if _, err := os.Stat("Makefile"); os.IsNotExist(err) {
		makefile := `BINARY := ` + dir + `

.PHONY: build run test clean

build:
	go build -o bin/$(BINARY) .

run:
	go run .

test:
	go test ./... -v

clean:
	rm -rf bin/
`
		os.WriteFile("Makefile", []byte(makefile), 0o644)
		ui.Ok("Created Makefile")
	}

	ui.Ok("Go project ready")
	fmt.Println()
	return nil
}

func craftNode() error {
	if _, err := os.Stat("package.json"); err == nil {
		ui.Ok("Node project already initialized")
		return nil
	}

	fmt.Println()
	ui.Puts("  Setting up Node.js project...")

	if err := runCmd("npm", "init", "-y"); err != nil {
		return fmt.Errorf("npm init: %w", err)
	}

	ui.Ok("Node.js project ready")
	fmt.Println()
	return nil
}

func craftPython() error {
	if _, err := os.Stat("pyproject.toml"); err == nil {
		ui.Ok("Python project already initialized")
		return nil
	}

	cwd, _ := os.Getwd()
	dir := filepath.Base(cwd)

	fmt.Println()
	ui.Puts("  Setting up Python project...")

	pyproject := `[project]
name = "` + dir + `"
version = "0.1.0"
requires-python = ">=3.11"

[build-system]
requires = ["setuptools>=75.0"]
build-backend = "setuptools.backends._legacy:_Backend"
`
	os.WriteFile("pyproject.toml", []byte(pyproject), 0o644)
	ui.Ok("Created pyproject.toml")

	// Create virtual env
	if err := runCmd("python3", "-m", "venv", ".venv"); err != nil {
		ui.Warn("Could not create virtual env: " + err.Error())
	} else {
		ui.Ok("Created .venv")
	}

	ui.Ok("Python project ready")
	fmt.Printf("  Activate: %s\n", ui.Accent.Render("source .venv/bin/activate"))
	fmt.Println()
	return nil
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
