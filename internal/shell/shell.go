package shell

import "fmt"

// Supported shell types.
const (
	Bash = "bash"
	Zsh  = "zsh"
	Fish = "fish"
)

// ValidShell returns true if the shell name is supported.
func ValidShell(name string) bool {
	switch name {
	case Bash, Zsh, Fish:
		return true
	}
	return false
}

// ShellError is returned for unsupported shell types.
func ShellError(name string) error {
	return fmt.Errorf("unknown shell %q — supported: bash, zsh, fish", name)
}
