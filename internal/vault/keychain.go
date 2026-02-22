package vault

import (
	"errors"
	"os"
)

// ErrNotSupported is returned by PassphraseStore implementations on
// platforms where no keychain integration is available.
var ErrNotSupported = errors.New("keychain not supported on this platform")

// ServiceName is the keychain service identifier used for vault passphrases.
const ServiceName = "mine-vault"

// PassphraseStore is the interface for OS-native keychain integration.
// Platform implementations shell out to OS CLI tools (security on macOS,
// secret-tool on Linux). A no-op fallback is used on unsupported platforms.
type PassphraseStore interface {
	// Get retrieves the stored passphrase for the given service.
	// Returns os.ErrNotExist when no passphrase is stored.
	// Returns ErrNotSupported on unsupported platforms.
	Get(service string) (string, error)

	// Set stores the passphrase for the given service in the OS keychain.
	// Returns ErrNotSupported on unsupported platforms.
	Set(service string, passphrase string) error

	// Delete removes the stored passphrase for the given service.
	// Returns os.ErrNotExist if no passphrase is stored.
	// Returns ErrNotSupported on unsupported platforms.
	Delete(service string) error
}

// noopKeychain is a PassphraseStore that always returns ErrNotSupported.
// Used on unsupported platforms or when required tools are unavailable.
type noopKeychain struct{}

func (n *noopKeychain) Get(_ string) (string, error)    { return "", ErrNotSupported }
func (n *noopKeychain) Set(_ string, _ string) error    { return ErrNotSupported }
func (n *noopKeychain) Delete(_ string) error           { return ErrNotSupported }

// IsKeychainMiss returns true if err means "no passphrase stored" —
// either the platform is unsupported or no entry exists yet.
func IsKeychainMiss(err error) bool {
	return errors.Is(err, ErrNotSupported) || errors.Is(err, os.ErrNotExist)
}
