package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/rnwolfe/mine/internal/meta"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var (
	metaFrDesc    string
	metaFrUseCase string
	metaDryRun    bool
	metaBugSteps  string
	metaBugExpect string
	metaBugActual string
)

var metaCmd = &cobra.Command{
	Use:   "meta",
	Short: "Interact with mine-as-a-product",
	Long:  `Commands for feature requests, bug reports, and other mine meta-operations.`,
	RunE:  runMetaHelp,
}

var metaFrCmd = &cobra.Command{
	Use:   "fr [title]",
	Short: "Submit a feature request",
	Long:  `Submit a feature request as a GitHub issue on the mine repo.`,
	RunE:  runMetaFr,
}

var metaBugCmd = &cobra.Command{
	Use:   "bug [title]",
	Short: "Report a bug",
	Long:  `Report a bug as a GitHub issue on the mine repo with auto-detected system info.`,
	RunE:  runMetaBug,
}

func init() {
	rootCmd.AddCommand(metaCmd)
	metaCmd.AddCommand(metaFrCmd)
	metaCmd.AddCommand(metaBugCmd)

	metaFrCmd.Flags().StringVarP(&metaFrDesc, "description", "d", "", "What the feature should do")
	metaFrCmd.Flags().StringVarP(&metaFrUseCase, "use-case", "u", "", "Why you need this feature")
	metaFrCmd.Flags().BoolVar(&metaDryRun, "dry-run", false, "Preview the issue without submitting")

	metaBugCmd.Flags().StringVarP(&metaBugSteps, "steps", "s", "", "Steps to reproduce")
	metaBugCmd.Flags().StringVarP(&metaBugExpect, "expected", "e", "", "Expected behavior")
	metaBugCmd.Flags().StringVarP(&metaBugActual, "actual", "a", "", "Actual behavior")
	metaBugCmd.Flags().BoolVar(&metaDryRun, "dry-run", false, "Preview the issue without submitting")
}

func runMetaHelp(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println(ui.Title.Render("  Meta — interact with mine itself"))
	fmt.Println()
	fmt.Println("  Available commands:")
	fmt.Println()
	fmt.Printf("    %s    %s\n", ui.Accent.Render("mine meta fr <title>"), ui.Muted.Render("Submit a feature request"))
	fmt.Printf("    %s   %s\n", ui.Accent.Render("mine meta bug <title>"), ui.Muted.Render("Report a bug"))
	fmt.Println()
	ui.Tip("Use --dry-run to preview before submitting.")
	fmt.Println()
	return nil
}

func runMetaFr(_ *cobra.Command, args []string) error {
	if !metaDryRun {
		if err := meta.CheckGH(); err != nil {
			return err
		}
	}

	reader := bufio.NewReader(os.Stdin)

	title := strings.Join(args, " ")
	if strings.TrimSpace(title) == "" {
		title = promptField(reader, "Title (short summary of the feature)")
	}
	if err := meta.ValidateTitle(title); err != nil {
		return fmt.Errorf("invalid title: %w", err)
	}
	title = meta.RedactPII(title)

	description := metaFrDesc
	if description == "" {
		description = promptField(reader, "Description (what should this feature do?)")
	}
	if err := meta.ValidateRequired(description, "description"); err != nil {
		return err
	}

	useCase := metaFrUseCase
	if useCase == "" {
		useCase = promptField(reader, "Use case (why do you need this?)")
	}
	if err := meta.ValidateRequired(useCase, "use case"); err != nil {
		return err
	}

	body := meta.FormatFeatureRequest(description, useCase)
	body = meta.RedactPII(body)

	if metaDryRun {
		printDryRun(title, "feature-request", body)
		return nil
	}

	printDryRun(title, "feature-request", body)
	if !confirmSubmit(reader) {
		fmt.Println()
		ui.Warn("Cancelled.")
		return nil
	}

	url, err := meta.CreateIssue(title, body, "feature-request")
	if err != nil {
		return err
	}

	fmt.Println()
	ui.Ok("Feature request submitted!")
	fmt.Printf("  %s\n", ui.Accent.Render(url))
	fmt.Println()
	return nil
}

func runMetaBug(_ *cobra.Command, args []string) error {
	if !metaDryRun {
		if err := meta.CheckGH(); err != nil {
			return err
		}
	}

	reader := bufio.NewReader(os.Stdin)

	title := strings.Join(args, " ")
	if strings.TrimSpace(title) == "" {
		title = promptField(reader, "Title (short summary of the bug)")
	}
	if err := meta.ValidateTitle(title); err != nil {
		return fmt.Errorf("invalid title: %w", err)
	}
	title = meta.RedactPII(title)

	steps := metaBugSteps
	if steps == "" {
		steps = promptField(reader, "Steps to reproduce")
	}
	if err := meta.ValidateRequired(steps, "steps to reproduce"); err != nil {
		return err
	}

	expected := metaBugExpect
	if expected == "" {
		expected = promptField(reader, "Expected behavior")
	}
	if err := meta.ValidateRequired(expected, "expected behavior"); err != nil {
		return err
	}

	actual := metaBugActual
	if actual == "" {
		actual = promptField(reader, "Actual behavior")
	}
	if err := meta.ValidateRequired(actual, "actual behavior"); err != nil {
		return err
	}

	info := meta.CollectSystemInfo()
	body := meta.FormatBugReport(steps, expected, actual, info)
	body = meta.RedactPII(body)

	if metaDryRun {
		printDryRun(title, "bug", body)
		return nil
	}

	printDryRun(title, "bug", body)
	if !confirmSubmit(reader) {
		fmt.Println()
		ui.Warn("Cancelled.")
		return nil
	}

	url, err := meta.CreateIssue(title, body, "bug")
	if err != nil {
		return err
	}

	fmt.Println()
	ui.Ok("Bug report submitted!")
	fmt.Printf("  %s\n", ui.Accent.Render(url))
	fmt.Println()
	return nil
}

// promptField asks the user for input when a flag was not provided.
func promptField(reader *bufio.Reader, question string) string {
	fmt.Printf("\n  %s\n  %s ", ui.Muted.Render(question+":"), ui.Accent.Render(">"))
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// confirmSubmit asks the user to confirm before creating the issue.
func confirmSubmit(reader *bufio.Reader) bool {
	fmt.Printf("\n  %s ", ui.Accent.Render("Submit this issue? [y/N]"))
	line, _ := reader.ReadString('\n')
	ans := strings.TrimSpace(strings.ToLower(line))
	return ans == "y" || ans == "yes"
}

func printDryRun(title, label, body string) {
	fmt.Println()
	ui.Header("Preview")
	fmt.Println()
	ui.Kv("Title", title)
	ui.Kv("Label", label)
	ui.Kv("Repo", "rnwolfe/mine")
	fmt.Println()
	fmt.Println(ui.Muted.Render("  --- Body ---"))
	fmt.Println()
	for _, line := range strings.Split(body, "\n") {
		fmt.Printf("  %s\n", line)
	}
}
