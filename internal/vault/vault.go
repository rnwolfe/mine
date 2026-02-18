// Package vault provides encrypted secret storage using age encryption.
// Secrets are stored in a single age-encrypted file at
// ~/.local/share/mine/vault.age (XDG DataDir).
//
// The vault uses passphrase-based encryption (age scrypt). All mutations are
// written atomically: data is written to a temp file, fsync'd, then renamed
// into place to prevent corruption on crash.
package vault

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"filippo.io/age"
	"filippo.io/age/armor"

	"github.com/rnwolfe/mine/internal/config"
)

// ErrWrongPassphrase is returned when decryption fails due to a bad passphrase.
var ErrWrongPassphrase = errors.New("wrong passphrase")

// ErrCorruptedVault is returned when the vault file exists but cannot be parsed.
var ErrCorruptedVault = errors.New("vault file is corrupted or unreadable")

// vaultData is the in-memory representation of vault contents (plaintext JSON inside the age file).
type vaultData struct {
	Secrets map[string]string `json:"secrets"`
}

// Vault manages an age-encrypted secret store.
type Vault struct {
	mu         sync.Mutex
	path       string
	passphrase string
}

// New creates a Vault instance backed by the XDG data path.
// The passphrase is required to open/write the vault.
func New(passphrase string) *Vault {
	paths := config.GetPaths()
	return &Vault{
		path:       filepath.Join(paths.DataDir, "vault.age"),
		passphrase: passphrase,
	}
}

// newWithPath creates a Vault at an explicit path (used in tests).
func newWithPath(path, passphrase string) *Vault {
	return &Vault{
		path:       path,
		passphrase: passphrase,
	}
}

// Set stores or updates an encrypted secret.
func (v *Vault) Set(key, value string) error {
	if key == "" {
		return fmt.Errorf("key must not be empty")
	}
	v.mu.Lock()
	defer v.mu.Unlock()

	data, err := v.load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if data == nil {
		data = &vaultData{Secrets: make(map[string]string)}
	}

	data.Secrets[key] = value
	return v.save(data)
}

// Get retrieves a secret by key.
// Returns ErrWrongPassphrase if the passphrase is incorrect.
// Returns ErrCorruptedVault if the file exists but cannot be decrypted/parsed.
// Returns os.ErrNotExist if the vault file does not exist.
func (v *Vault) Get(key string) (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	data, err := v.load()
	if err != nil {
		return "", err
	}

	val, ok := data.Secrets[key]
	if !ok {
		return "", fmt.Errorf("secret %q not found in vault", key)
	}
	return val, nil
}

// Delete removes a secret by key.
func (v *Vault) Delete(key string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	data, err := v.load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // nothing to delete
		}
		return err
	}

	if _, ok := data.Secrets[key]; !ok {
		return fmt.Errorf("secret %q not found in vault", key)
	}

	delete(data.Secrets, key)
	return v.save(data)
}

// List returns all secret keys in sorted order. Values are never included.
func (v *Vault) List() ([]string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	data, err := v.load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}
		return nil, err
	}

	keys := make([]string, 0, len(data.Secrets))
	for k := range data.Secrets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}

// Export writes the raw encrypted vault file bytes to w.
// The exported blob is still age-encrypted and safe to store/transfer.
func (v *Vault) Export(w io.Writer) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	f, err := os.Open(v.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("vault is empty — nothing to export")
		}
		return fmt.Errorf("opening vault for export: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("exporting vault: %w", err)
	}
	return nil
}

// Import replaces the vault contents with the data from r.
// The import data must be a valid age-encrypted vault blob that can be
// decrypted with the current passphrase. No merge is performed: the existing
// vault is replaced entirely.
func (v *Vault) Import(r io.Reader) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Read import data into a buffer so we can validate before committing.
	raw, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading import data: %w", err)
	}

	// Validate: decrypt and parse to ensure the data is a valid vault with
	// the current passphrase before we overwrite anything.
	if _, err := decryptData(raw, v.passphrase); err != nil {
		return err
	}

	// Atomic write: write to temp, rename into place.
	return atomicWrite(v.path, raw)
}

// Path returns the vault file path.
func (v *Vault) Path() string {
	return v.path
}

// load reads and decrypts the vault from disk.
// Returns os.ErrNotExist if the vault file does not exist.
func (v *Vault) load() (*vaultData, error) {
	raw, err := os.ReadFile(v.path)
	if err != nil {
		return nil, err // may be os.ErrNotExist — callers handle this
	}

	return decryptData(raw, v.passphrase)
}

// save encrypts and atomically writes vault data to disk.
func (v *Vault) save(data *vaultData) error {
	if err := os.MkdirAll(filepath.Dir(v.path), 0o700); err != nil {
		return fmt.Errorf("creating vault directory: %w", err)
	}

	raw, err := encryptData(data, v.passphrase)
	if err != nil {
		return err
	}

	return atomicWrite(v.path, raw)
}

// encryptData serializes and encrypts vault data using age scrypt (passphrase-based).
func encryptData(data *vaultData, passphrase string) ([]byte, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("serializing vault: %w", err)
	}

	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return nil, fmt.Errorf("creating age recipient: %w", err)
	}

	var buf bytes.Buffer
	armorWriter := armor.NewWriter(&buf)

	w, err := age.Encrypt(armorWriter, recipient)
	if err != nil {
		return nil, fmt.Errorf("initializing age encryption: %w", err)
	}

	if _, err := w.Write(jsonBytes); err != nil {
		return nil, fmt.Errorf("encrypting vault data: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("finalizing encryption: %w", err)
	}
	if err := armorWriter.Close(); err != nil {
		return nil, fmt.Errorf("finalizing armor: %w", err)
	}

	return buf.Bytes(), nil
}

// decryptData decrypts and deserializes vault data from age-encrypted bytes.
func decryptData(raw []byte, passphrase string) (*vaultData, error) {
	identity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return nil, fmt.Errorf("creating age identity: %w", err)
	}

	armorReader := armor.NewReader(bytes.NewReader(raw))
	r, err := age.Decrypt(armorReader, identity)
	if err != nil {
		// filippo.io/age does not export typed errors for wrong passphrase (as of v1.x).
		// We detect it by matching known error message substrings. This is fragile:
		// if the library changes its error wording, wrong passphrases will silently
		// fall through to ErrCorruptedVault. Revisit if age exports typed errors in
		// a future release (track: https://github.com/FiloSottile/age/issues).
		msg := err.Error()
		if strings.Contains(msg, "no identity matched") || strings.Contains(msg, "incorrect") {
			return nil, fmt.Errorf("%w: %v\n\nHint: check your passphrase and try again.\nIf you forgot your passphrase, you cannot recover the vault.", ErrWrongPassphrase, err)
		}
		return nil, fmt.Errorf("%w: %v\n\nHint: the vault file may be damaged. Keep a backup copy.\nDo NOT attempt to edit the vault file manually.", ErrCorruptedVault, err)
	}

	plaintext, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: reading decrypted data: %v", ErrCorruptedVault, err)
	}

	var data vaultData
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return nil, fmt.Errorf("%w: parsing vault JSON: %v", ErrCorruptedVault, err)
	}

	if data.Secrets == nil {
		data.Secrets = make(map[string]string)
	}
	return &data, nil
}

// atomicWrite writes data to path atomically: write temp file → fsync → rename.
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".vault-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()

	// Ensure cleanup on failure.
	success := false
	defer func() {
		if !success {
			os.Remove(tmpName)
		}
	}()

	if err := os.Chmod(tmpName, 0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("setting temp file permissions: %w", err)
	}

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing vault data: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("fsyncing vault data: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("committing vault file: %w", err)
	}

	success = true
	return nil
}
