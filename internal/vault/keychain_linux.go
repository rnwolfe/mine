//go:build linux

package vault

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type linuxKeychain struct{}

// NewPlatformStore returns a secret-tool backed PassphraseStore, or a no-op
// implementation if secret-tool is not installed (graceful degradation).
func NewPlatformStore() PassphraseStore {
	if _, err := exec.LookPath("secret-tool"); err != nil {
		return &noopKeychain{}
	}
	return &linuxKeychain{}
}

func (l *linuxKeychain) Get(service string) (string, error) {
	out, err := exec.Command("secret-tool", "lookup", "service", service).CombinedOutput()
	if err != nil {
		outStr := strings.TrimSpace(string(out))
		// secret-tool exits non-zero with no output when the entry doesn't exist.
		if outStr == "" {
			return "", os.ErrNotExist
		}
		return "", fmt.Errorf("retrieving from keychain: %w: %s", err, outStr)
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return "", os.ErrNotExist
	}
	return s, nil
}

func (l *linuxKeychain) Set(service, passphrase string) error {
	cmd := exec.Command("secret-tool", "store",
		"--label=mine vault passphrase",
		"service", service,
	)
	cmd.Stdin = strings.NewReader(passphrase)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("storing in keychain: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (l *linuxKeychain) Delete(service string) error {
	cmd := exec.Command("secret-tool", "clear", "service", service)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("deleting from keychain: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
