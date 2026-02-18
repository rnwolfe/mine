package vault

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testPassphrase = "test-passphrase-12345"

func newTestVault(t *testing.T, passphrase string) *Vault {
	t.Helper()
	path := filepath.Join(t.TempDir(), "vault.age")
	return newWithPath(path, passphrase)
}

func TestSetAndGet(t *testing.T) {
	v := newTestVault(t, testPassphrase)

	if err := v.Set("my.key", "super-secret-value"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := v.Get("my.key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "super-secret-value" {
		t.Errorf("Get = %q, want %q", got, "super-secret-value")
	}
}

func TestSetOverwrite(t *testing.T) {
	v := newTestVault(t, testPassphrase)

	if err := v.Set("key", "first"); err != nil {
		t.Fatalf("Set first: %v", err)
	}
	if err := v.Set("key", "second"); err != nil {
		t.Fatalf("Set second: %v", err)
	}

	got, err := v.Get("key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "second" {
		t.Errorf("Get = %q, want %q", got, "second")
	}
}

func TestMultipleSecrets(t *testing.T) {
	v := newTestVault(t, testPassphrase)

	secrets := map[string]string{
		"ai.claude.api_key":   "sk-ant-123",
		"ai.openai.api_key":   "sk-openai-456",
		"db.password":         "hunter2",
	}

	for k, val := range secrets {
		if err := v.Set(k, val); err != nil {
			t.Fatalf("Set(%q): %v", k, err)
		}
	}

	for k, want := range secrets {
		got, err := v.Get(k)
		if err != nil {
			t.Fatalf("Get(%q): %v", k, err)
		}
		if got != want {
			t.Errorf("Get(%q) = %q, want %q", k, got, want)
		}
	}
}

func TestGetMissingKey(t *testing.T) {
	v := newTestVault(t, testPassphrase)

	if err := v.Set("existing", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	_, err := v.Get("missing")
	if err == nil {
		t.Fatal("Get missing key: expected error, got nil")
	}
}

func TestDelete(t *testing.T) {
	v := newTestVault(t, testPassphrase)

	if err := v.Set("to-delete", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := v.Delete("to-delete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := v.Get("to-delete")
	if err == nil {
		t.Fatal("Get after Delete: expected error, got nil")
	}
}

func TestDeleteMissingKey(t *testing.T) {
	v := newTestVault(t, testPassphrase)

	if err := v.Set("present", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	err := v.Delete("absent")
	if err == nil {
		t.Fatal("Delete absent key: expected error, got nil")
	}
}

func TestDeleteOnEmptyVault(t *testing.T) {
	v := newTestVault(t, testPassphrase)
	// Delete when vault doesn't exist yet should not error.
	if err := v.Delete("anything"); err != nil {
		t.Fatalf("Delete on empty vault: %v", err)
	}
}

func TestList(t *testing.T) {
	v := newTestVault(t, testPassphrase)

	keys := []string{"b.key", "a.key", "c.key"}
	for _, k := range keys {
		if err := v.Set(k, "val"); err != nil {
			t.Fatalf("Set(%q): %v", k, err)
		}
	}

	got, err := v.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("List len = %d, want 3", len(got))
	}
	// Should be sorted
	if got[0] != "a.key" || got[1] != "b.key" || got[2] != "c.key" {
		t.Errorf("List = %v, want sorted [a.key b.key c.key]", got)
	}
}

func TestListEmptyVault(t *testing.T) {
	v := newTestVault(t, testPassphrase)
	keys, err := v.List()
	if err != nil {
		t.Fatalf("List on empty vault: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("List on empty vault = %v, want empty", keys)
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	data := &vaultData{
		Secrets: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	encrypted, err := encryptData(data, testPassphrase)
	if err != nil {
		t.Fatalf("encryptData: %v", err)
	}

	// Verify plaintext is not visible in the encrypted output.
	if bytes.Contains(encrypted, []byte("value1")) {
		t.Error("plaintext value found in encrypted output")
	}
	if bytes.Contains(encrypted, []byte("value2")) {
		t.Error("plaintext value found in encrypted output")
	}

	decrypted, err := decryptData(encrypted, testPassphrase)
	if err != nil {
		t.Fatalf("decryptData: %v", err)
	}

	if decrypted.Secrets["key1"] != "value1" {
		t.Errorf("key1 = %q, want %q", decrypted.Secrets["key1"], "value1")
	}
	if decrypted.Secrets["key2"] != "value2" {
		t.Errorf("key2 = %q, want %q", decrypted.Secrets["key2"], "value2")
	}
}

func TestWrongPassphrase(t *testing.T) {
	v := newTestVault(t, testPassphrase)

	if err := v.Set("key", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Try to read with a different passphrase.
	vWrong := newWithPath(v.path, "wrong-passphrase")
	_, err := vWrong.Get("key")
	if err == nil {
		t.Fatal("Get with wrong passphrase: expected error, got nil")
	}
	if !errors.Is(err, ErrWrongPassphrase) {
		t.Errorf("Get with wrong passphrase: error = %v, want ErrWrongPassphrase", err)
	}
}

func TestCorruptedVaultFile(t *testing.T) {
	v := newTestVault(t, testPassphrase)

	// Write garbage data to the vault file.
	if err := os.WriteFile(v.path, []byte("this is not a valid age file"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := v.Get("any")
	if err == nil {
		t.Fatal("Get on corrupted vault: expected error, got nil")
	}
	if !errors.Is(err, ErrCorruptedVault) {
		t.Errorf("Get on corrupted vault: error = %v, want ErrCorruptedVault", err)
	}
}

func TestEmptyKeyRejected(t *testing.T) {
	v := newTestVault(t, testPassphrase)
	err := v.Set("", "value")
	if err == nil {
		t.Fatal("Set with empty key: expected error, got nil")
	}
}

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.age")
	data := []byte("test content")

	if err := atomicWrite(path, data); err != nil {
		t.Fatalf("atomicWrite: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("content mismatch: got %q, want %q", got, data)
	}

	// Check file permissions.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("file mode = %o, want 0600", info.Mode().Perm())
	}

	// Verify no temp files remain.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".vault-") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func TestExportImport(t *testing.T) {
	src := newTestVault(t, testPassphrase)
	if err := src.Set("exported.key", "exported-value"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Export to a buffer.
	var buf bytes.Buffer
	if err := src.Export(&buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	// Import into a new vault at a different path.
	dst := newTestVault(t, testPassphrase)
	if err := dst.Import(&buf); err != nil {
		t.Fatalf("Import: %v", err)
	}

	// Verify the imported secret is accessible.
	got, err := dst.Get("exported.key")
	if err != nil {
		t.Fatalf("Get after Import: %v", err)
	}
	if got != "exported-value" {
		t.Errorf("Get after Import = %q, want %q", got, "exported-value")
	}
}

func TestImportWrongPassphrase(t *testing.T) {
	src := newTestVault(t, testPassphrase)
	if err := src.Set("key", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	var buf bytes.Buffer
	if err := src.Export(&buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	// Import with wrong passphrase should fail.
	dst := newTestVault(t, "different-passphrase")
	err := dst.Import(&buf)
	if err == nil {
		t.Fatal("Import with wrong passphrase: expected error, got nil")
	}
	if !errors.Is(err, ErrWrongPassphrase) {
		t.Errorf("Import with wrong passphrase: error = %v, want ErrWrongPassphrase", err)
	}
}

func TestExportEmptyVault(t *testing.T) {
	v := newTestVault(t, testPassphrase)
	var buf bytes.Buffer
	err := v.Export(&buf)
	if err == nil {
		t.Fatal("Export empty vault: expected error, got nil")
	}
}

func TestPlaintextNotOnDisk(t *testing.T) {
	v := newTestVault(t, testPassphrase)
	secret := "super-secret-api-key-xyz"
	if err := v.Set("key", secret); err != nil {
		t.Fatalf("Set: %v", err)
	}

	raw, err := os.ReadFile(v.path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if bytes.Contains(raw, []byte(secret)) {
		t.Error("plaintext secret found in vault file on disk")
	}
}

func TestFilePermissions(t *testing.T) {
	v := newTestVault(t, testPassphrase)
	if err := v.Set("key", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	info, err := os.Stat(v.path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("vault file mode = %o, want 0600", info.Mode().Perm())
	}
}
