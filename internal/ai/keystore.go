package ai

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rnwolfe/mine/internal/config"
)

// Keystore manages encrypted API keys.
type Keystore struct {
	path string
	key  []byte
}

// keystoreData is the on-disk format.
type keystoreData struct {
	Keys map[string]string `json:"keys"` // provider -> encrypted key
}

// NewKeystore creates a keystore instance.
// The encryption key is derived from machine-specific data + config path.
func NewKeystore() (*Keystore, error) {
	paths := config.GetPaths()
	keystorePath := filepath.Join(paths.DataDir, "keystore.enc")

	// Derive encryption key from machine-specific info.
	// This isn't perfect security, but it prevents plaintext storage
	// and is better than nothing without external dependencies.
	//
	// TODO: For future enhancement, consider:
	// - OS keychain integration (macOS Keychain, Linux gnome-keyring, Windows Credential Manager)
	// - Deriving key from machine ID files that are harder to predict
	// - User-specific entropy sources beyond hostname and data directory path
	hostname, _ := os.Hostname()
	seed := fmt.Sprintf("%s:%s", hostname, paths.DataDir)
	keyHash := sha256.Sum256([]byte(seed))

	return &Keystore{
		path: keystorePath,
		key:  keyHash[:],
	}, nil
}

// Set stores an encrypted API key for a provider.
func (k *Keystore) Set(provider, apiKey string) error {
	data, err := k.load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	encrypted, err := k.encrypt(apiKey)
	if err != nil {
		return err
	}

	if data.Keys == nil {
		data.Keys = make(map[string]string)
	}
	data.Keys[provider] = encrypted

	return k.save(data)
}

// Get retrieves and decrypts an API key for a provider.
func (k *Keystore) Get(provider string) (string, error) {
	data, err := k.load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("no API key configured for %s", provider)
		}
		return "", err
	}

	encrypted, ok := data.Keys[provider]
	if !ok {
		return "", fmt.Errorf("no API key configured for %s", provider)
	}

	return k.decrypt(encrypted)
}

// Delete removes an API key for a provider.
func (k *Keystore) Delete(provider string) error {
	data, err := k.load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // Already deleted
		}
		return err
	}

	delete(data.Keys, provider)
	return k.save(data)
}

// List returns all providers with stored keys.
func (k *Keystore) List() ([]string, error) {
	data, err := k.load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}
		return nil, err
	}

	providers := make([]string, 0, len(data.Keys))
	for p := range data.Keys {
		providers = append(providers, p)
	}
	return providers, nil
}

func (k *Keystore) load() (*keystoreData, error) {
	raw, err := os.ReadFile(k.path)
	if err != nil {
		return &keystoreData{}, err
	}

	var data keystoreData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (k *Keystore) save(data *keystoreData) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(k.path), 0o700); err != nil {
		return err
	}

	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// Write with restricted permissions and ensure they are enforced even if the file already existed.
	if err := os.WriteFile(k.path, raw, 0o600); err != nil {
		return err
	}
	return os.Chmod(k.path, 0o600)
}

func (k *Keystore) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(k.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (k *Keystore) decrypt(encoded string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(k.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
