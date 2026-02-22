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
	out, err := exec.Command("security", "find-generic-password", "-s", service, "-w").Output()
	if err != nil {
		return "", os.ErrNotExist
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

	cmd := exec.Command("security", "add-generic-password",
		"-s", service,
		"-a", "mine",
		"-w", passphrase,
	)
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
