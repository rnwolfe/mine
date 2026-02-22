//go:build darwin

package vault

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type darwinKeychain struct{}

// NewPlatformStore returns the macOS Keychain implementation using the
// built-in `security` CLI tool (no extra dependencies required).
func NewPlatformStore() PassphraseStore {
	return &darwinKeychain{}
}

func (d *darwinKeychain) Get(service string) (string, error) {
	cmd := exec.Command("security", "find-generic-password", "-s", service, "-w")
	out, err := cmd.CombinedOutput()
	if err != nil {
		outStr := string(out)
		if strings.Contains(outStr, "could not be found") || strings.Contains(outStr, "not found") {
			return "", os.ErrNotExist
		}
		return "", fmt.Errorf("reading from keychain: %w: %s", err, strings.TrimSpace(outStr))
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return "", os.ErrNotExist
	}
	return s, nil
}

func (d *darwinKeychain) Set(service, passphrase string) error {
	// Delete any existing entry first — `security add-generic-password` errors
	// if a duplicate entry already exists for the same service name.
	_ = exec.Command("security", "delete-generic-password", "-s", service).Run()

	// Pass passphrase via stdin using `-w` with no argument to avoid exposing
	// it in the process list (visible via `ps aux`).
	cmd := exec.Command("security", "add-generic-password",
		"-s", service,
		"-a", "mine",
		"-w",
	)
	cmd.Stdin = strings.NewReader(passphrase)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("storing in keychain: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (d *darwinKeychain) Delete(service string) error {
	out, err := exec.Command("security", "delete-generic-password", "-s", service).CombinedOutput()
	if err != nil {
		outStr := string(out)
		if strings.Contains(outStr, "could not be found") || strings.Contains(outStr, "not found") {
			return os.ErrNotExist
		}
		return fmt.Errorf("deleting from keychain: %w: %s", err, strings.TrimSpace(outStr))
	}
	return nil
}
