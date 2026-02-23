package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rnwolfe/mine/internal/craft"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var registry *craft.Registry

var craftCmd = &cobra.Command{
	Use:   "craft",
	Short: "Scaffold projects and bootstrap dev tools",
	Long:  `Quickly spin up new projects or install dev tool configurations.`,
	RunE:  hook.Wrap("craft", runCraftHelp),
}

func init() {
	registry = craft.NewRegistry()
	// Attempt to load user recipes (non-fatal if it fails).
	_ = registry.LoadUserRecipes()

	rootCmd.AddCommand(craftCmd)
	craftCmd.AddCommand(craftGitCmd)
	craftCmd.AddCommand(craftDevCmd)
	craftCmd.AddCommand(craftCICmd)
	craftCmd.AddCommand(craftListCmd)
}

var craftGitCmd = &cobra.Command{
	Use:   "git",
	Short: "Set up a git repository with best practices",
	RunE:  hook.Wrap("craft.git", runCraftGit),
}

var craftDevCmd = &cobra.Command{
	Use:   "dev <tool>",
	Short: "Quick-start a dev tool (go, node, python, rust, docker)",
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("craft.dev", runCraftDev),
}

var craftCICmd = &cobra.Command{
	Use:   "ci <provider>",
	Short: "Generate CI/CD workflow templates (github)",
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("craft.ci", runCraftCI),
}

var craftListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available scaffolding recipes",
	RunE:  hook.Wrap("craft.list", runCraftList),
}

func runCraftHelp(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println(ui.Title.Render("  Craft — build and bootstrap"))
	fmt.Println()
	fmt.Println("  Available recipes:")
	fmt.Println()
	fmt.Printf("    %s   %s\n", ui.Accent.Render("mine craft git"), ui.Muted.Render("Set up git with .gitignore, hooks, etc."))

	recipes := registry.List()
	for _, r := range recipes {
		cmd := fmt.Sprintf("mine craft %s %s", r.Category, r.Name)
		fmt.Printf("    %s   %s\n", ui.Accent.Render(fmt.Sprintf("%-26s", cmd)), ui.Muted.Render(r.Description))
	}

	fmt.Println()
	fmt.Printf("    %s   %s\n", ui.Accent.Render("mine craft list"), ui.Muted.Render("Show all recipes with details"))
	fmt.Println()
	return nil
}

func runCraftGit(_ *cobra.Command, _ []string) error {
	cwd, _ := os.Getwd()

	if _, err := os.Stat(".git"); err == nil {
		ui.Ok("Already a git repo: " + cwd)
		fmt.Println()
		return nil
	}

	if err := runCmd("git", "init"); err != nil {
		return fmt.Errorf("git init failed: %w", err)
	}

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
	return runRecipe("dev", args[0])
}

func runCraftCI(_ *cobra.Command, args []string) error {
	return runRecipe("ci", args[0])
}

func runRecipe(category, name string) error {
	recipe, ok := registry.Get(category, strings.ToLower(name))
	if !ok {
		available := recipesInCategory(category)
		return fmt.Errorf("unknown %s recipe %q — try: %s", category, name, strings.Join(available, ", "))
	}

	fmt.Println()
	ui.Puts(fmt.Sprintf("  Setting up %s/%s...", recipe.Category, recipe.Name))

	data := craft.CurrentDir()

	created, err := craft.Execute(recipe, data)
	if err != nil {
		// "already initialized" is not a hard error
		if strings.Contains(err.Error(), "already initialized") {
			ui.Ok(capitalize(recipe.Name) + " project already initialized")
			return nil
		}
		return err
	}

	for _, f := range created {
		ui.Ok("Created " + f)
	}

	// Run post commands
	for _, pc := range recipe.PostCommands {
		renderedArgs := craft.TemplateArgs(pc.Args, data)
		ui.Puts("  " + pc.Description + "...")
		if err := runCmd(pc.Name, renderedArgs...); err != nil {
			if pc.Optional {
				ui.Warn("Could not run " + pc.Name + ": " + err.Error())
			} else {
				return fmt.Errorf("%s: %w", pc.Description, err)
			}
		}
	}

	ui.Ok(capitalize(recipe.Name) + " project ready")
	ui.Tip("register this project with " + ui.Accent.Render("mine proj add ."))
	fmt.Println()
	return nil
}

func runCraftList(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println(ui.Title.Render("  Available Recipes"))
	fmt.Println()

	recipes := registry.List()
	currentCategory := ""
	for _, r := range recipes {
		if r.Category != currentCategory {
			if currentCategory != "" {
				fmt.Println()
			}
			fmt.Println(ui.Subtitle.Render(fmt.Sprintf("  %s", capitalize(r.Category)+" recipes")))
			fmt.Println()
			currentCategory = r.Category
		}

		cmd := fmt.Sprintf("mine craft %s %s", r.Category, r.Name)
		aliases := ""
		if len(r.Aliases) > 0 {
			aliases = ui.Muted.Render(fmt.Sprintf(" (aliases: %s)", strings.Join(r.Aliases, ", ")))
		}
		fmt.Printf("    %s  %s%s\n", ui.Accent.Render(fmt.Sprintf("%-28s", cmd)), ui.Muted.Render(r.Description), aliases)

		if len(r.Files) > 0 {
			var fileNames []string
			for _, f := range r.Files {
				fileNames = append(fileNames, f.Path)
			}
			fmt.Printf("    %s  %s\n", strings.Repeat(" ", 28), ui.Muted.Render("creates: "+strings.Join(fileNames, ", ")))
		}
	}

	fmt.Println()
	ui.Tip("User recipes go in ~/.config/mine/recipes/ (e.g. dev-mytemplate/)")
	fmt.Println()
	return nil
}

func recipesInCategory(category string) []string {
	var names []string
	seen := make(map[string]bool)
	for _, r := range registry.List() {
		if r.Category == category && !seen[r.Name] {
			names = append(names, r.Name)
			seen[r.Name] = true
		}
	}
	return names
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
