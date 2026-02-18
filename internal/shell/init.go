package shell

import "fmt"

// InitScript generates the complete shell initialization script.
// This is designed to be used with: eval "$(mine shell init bash)"
// It includes: aliases, utility functions, and prompt integration.
func InitScript(shellName string) (string, error) {
	if !ValidShell(shellName) {
		return "", ShellError(shellName)
	}

	var out string
	out += fmt.Sprintf("# mine shell init (%s) — https://mine.rwolfe.io\n", shellName)
	out += "# Add to your shell config: eval \"$(mine shell init " + shellName + ")\"\n\n"

	// Section 1: Aliases
	out += aliasesScript(shellName)

	// Section 2: Utility functions
	// Safe to ignore error: shellName already validated above.
	funcs, _ := FunctionsScript(shellName)
	out += funcs

	// Section 3: Prompt integration
	// Safe to ignore error: shellName already validated above.
	prompt, _ := PromptScript(shellName)
	out += prompt

	return out, nil
}

func aliasesScript(shellName string) string {
	type alias struct {
		short, full string
	}
	aliases := []alias{
		{"m", "mine"},
		{"mt", "mine todo"},
		{"mta", "mine todo add"},
		{"mtd", "mine todo done"},
		{"md", "mine dig"},
		{"mc", "mine craft"},
		{"ms", "mine stash"},
		{"mx", "mine tmux"},
		{"mg", "mine git"},
	}

	out := "# mine aliases\n"
	for _, a := range aliases {
		switch shellName {
		case Fish:
			out += fmt.Sprintf("alias %s '%s'\n", a.short, a.full)
		default:
			out += fmt.Sprintf("alias %s='%s'\n", a.short, a.full)
		}
	}
	out += "\n"
	return out
}
