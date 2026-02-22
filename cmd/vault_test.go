package cmd

import (
	"errors"
	"os"
	"testing"

	"github.com/rnwolfe/mine/internal/vault"
)

// mockPassphraseStore is a PassphraseStore implementation for tests.
type mockPassphraseStore struct {
	stored   string
	setErr   error
	deleteErr error
}

func (m *mockPassphraseStore) Get(_ string) (string, error) {
	if m.stored == "" {
		return "", os.ErrNotExist
	}
	return m.stored, nil
}

func (m *mockPassphraseStore) Set(_ string, passphrase string) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.stored = passphrase
	return nil
}

func (m *mockPassphraseStore) Delete(_ string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.stored = ""
	return nil
}

// setKeychainStore replaces vaultKeychainStore and returns a restore func.
func setKeychainStore(s vault.PassphraseStore) func() {
	orig := vaultKeychainStore
	vaultKeychainStore = s
	return func() { vaultKeychainStore = orig }
}

// --- readPassphrase resolution order tests ---

func TestReadPassphrase_EnvVarWins(t *testing.T) {
	t.Setenv("MINE_VAULT_PASSPHRASE", "from-env")
	restore := setKeychainStore(&mockPassphraseStore{stored: "from-keychain"})
	defer restore()

	got, err := readPassphrase(false)
	if err != nil {
		t.Fatalf("readPassphrase: %v", err)
	}
	if got != "from-env" {
		t.Errorf("expected from-env, got %q", got)
	}
}

func TestReadPassphrase_KeychainFallback(t *testing.T) {
	t.Setenv("MINE_VAULT_PASSPHRASE", "")
	restore := setKeychainStore(&mockPassphraseStore{stored: "from-keychain"})
	defer restore()

	got, err := readPassphrase(false)
	if err != nil {
		t.Fatalf("readPassphrase: %v", err)
	}
	if got != "from-keychain" {
		t.Errorf("expected from-keychain, got %q", got)
	}
}

func TestReadPassphrase_NoopFallsThrough(t *testing.T) {
	t.Setenv("MINE_VAULT_PASSPHRASE", "")
	// Empty mock = returns ErrNotExist, so keychain is skipped.
	restore := setKeychainStore(&mockPassphraseStore{stored: ""})
	defer restore()

	// Non-interactive stdin (os.Pipe) should trigger the headless error.
	origStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	_ = w.Close()
	os.Stdin = r
	defer func() {
		os.Stdin = origStdin
		_ = r.Close()
	}()

	_, err = readPassphrase(false)
	if err == nil {
		t.Fatal("expected error when no passphrase source available")
	}
}

// --- readEnvPassphrase resolution order tests ---

func TestReadEnvPassphrase_EnvVarWins(t *testing.T) {
	t.Setenv("MINE_ENV_PASSPHRASE", "from-env-passphrase")
	t.Setenv("MINE_VAULT_PASSPHRASE", "from-vault-env")
	restore := setKeychainStore(&mockPassphraseStore{stored: "from-keychain"})
	defer restore()

	got, err := readEnvPassphrase()
	if err != nil {
		t.Fatalf("readEnvPassphrase: %v", err)
	}
	if got != "from-env-passphrase" {
		t.Errorf("expected from-env-passphrase, got %q", got)
	}
}

func TestReadEnvPassphrase_VaultEnvFallback(t *testing.T) {
	t.Setenv("MINE_ENV_PASSPHRASE", "")
	t.Setenv("MINE_VAULT_PASSPHRASE", "from-vault-env")
	restore := setKeychainStore(&mockPassphraseStore{stored: "from-keychain"})
	defer restore()

	got, err := readEnvPassphrase()
	if err != nil {
		t.Fatalf("readEnvPassphrase: %v", err)
	}
	if got != "from-vault-env" {
		t.Errorf("expected from-vault-env, got %q", got)
	}
}

func TestReadEnvPassphrase_KeychainFallback(t *testing.T) {
	t.Setenv("MINE_ENV_PASSPHRASE", "")
	t.Setenv("MINE_VAULT_PASSPHRASE", "")
	restore := setKeychainStore(&mockPassphraseStore{stored: "from-keychain"})
	defer restore()

	got, err := readEnvPassphrase()
	if err != nil {
		t.Fatalf("readEnvPassphrase: %v", err)
	}
	if got != "from-keychain" {
		t.Errorf("expected from-keychain, got %q", got)
	}
}

// --- vaultLockCmd tests ---

func TestRunVaultLock_NotSupported(t *testing.T) {
	restore := setKeychainStore(&mockPassphraseStore{deleteErr: vault.ErrNotSupported})
	defer restore()

	err := runVaultLock(nil, nil)
	if err == nil {
		t.Fatal("expected error when keychain not supported")
	}
	if !errors.Is(err, vault.ErrNotSupported) {
		// The error should wrap or contain ErrNotSupported context — check the message.
		if err.Error() == "" {
			t.Errorf("expected non-empty error message")
		}
	}
}

func TestRunVaultLock_NothingStored(t *testing.T) {
	restore := setKeychainStore(&mockPassphraseStore{deleteErr: os.ErrNotExist})
	defer restore()

	// Should succeed silently (informational message, no error).
	err := runVaultLock(nil, nil)
	if err != nil {
		t.Fatalf("runVaultLock with nothing stored: %v", err)
	}
}

func TestRunVaultLock_Success(t *testing.T) {
	mock := &mockPassphraseStore{stored: "stored-passphrase"}
	restore := setKeychainStore(mock)
	defer restore()

	err := runVaultLock(nil, nil)
	if err != nil {
		t.Fatalf("runVaultLock: %v", err)
	}
	if mock.stored != "" {
		t.Errorf("expected stored to be empty after lock, got %q", mock.stored)
	}
}
