//go:build !darwin && !linux

package vault

// NewPlatformStore returns a no-op PassphraseStore on unsupported platforms.
// Vault commands fall back to env var or interactive prompt.
func NewPlatformStore() PassphraseStore {
	return &noopKeychain{}
}
