// Package gitutil provides a shared helper for running git commands
// inside a specific working directory. It is used by internal packages
// that maintain git-backed stores (stash, agents).
package gitutil

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// RunCmd runs a git command in dir and returns stdout.
// On failure it returns the trimmed stderr output (or the raw error
// string if stderr is empty) so callers can wrap it with context.
func RunCmd(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return stdout.String(), nil
}
