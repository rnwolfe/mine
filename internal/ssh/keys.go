package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Key represents an SSH key pair.
type Key struct {
	Name        string
	PrivatePath string
	PublicPath  string
	Fingerprint string
	// Hosts that reference this key
	UsedBy []string
}

// SSHDir returns the path to the ~/.ssh directory.
func SSHDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("~", ".ssh")
	}
	return filepath.Join(home, ".ssh")
}

// DefaultKeyPath returns the default ed25519 key path.
func DefaultKeyPath() string {
	return filepath.Join(SSHDir(), "id_ed25519")
}

// Keygen generates an ed25519 SSH key pair.
// name is used as the key filename (under ~/.ssh/).
// comment is embedded in the key (defaults to name if empty).
func Keygen(name, comment string) (string, error) {
	return KeygenFunc(name, comment)
}

// KeygenFunc is the replaceable implementation of Keygen (for testing).
var KeygenFunc = keygenReal

func keygenReal(name, comment string) (string, error) {
	sshDir := SSHDir()
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return "", fmt.Errorf("creating ~/.ssh: %w", err)
	}

	keyPath := filepath.Join(sshDir, name)

	// Refuse to overwrite an existing key to avoid silent data loss.
	if _, err := os.Stat(keyPath); err == nil {
		return "", fmt.Errorf("key file already exists: %s (remove it first or choose a different name)", keyPath)
	}

	if comment == "" {
		comment = name
	}

	// Wire stdin/stdout so the user is prompted for a passphrase by ssh-keygen.
	cmd := exec.Command("ssh-keygen",
		"-t", "ed25519",
		"-C", comment,
		"-f", keyPath,
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ssh-keygen failed: %w", err)
	}

	pubPath := keyPath + ".pub"
	return pubPath, nil
}

// CopyID copies the public key for host alias to the remote server.
func CopyID(alias string) error {
	return CopyIDFunc(alias)
}

// CopyIDFunc is the replaceable implementation of CopyID (for testing).
var CopyIDFunc = copyIDReal

func copyIDReal(alias string) error {
	cmd := exec.Command("ssh-copy-id", alias)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh-copy-id: %w", err)
	}
	return nil
}

// ListKeys scans ~/.ssh for key pairs and returns their info including fingerprints.
func ListKeys() ([]Key, error) {
	return ListKeysFrom(SSHDir(), ConfigPath())
}

// ListKeysFrom scans the given directory for key pairs.
func ListKeysFrom(sshDir, configPath string) ([]Key, error) {
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading ~/.ssh: %w", err)
	}

	// Build a map of key path → hosts that use it.
	// Ignore os.IsNotExist (no config file yet) but surface other errors.
	hosts, cfgErr := ReadHostsFrom(configPath)
	if cfgErr != nil && !os.IsNotExist(cfgErr) {
		return nil, fmt.Errorf("reading ssh config for key associations: %w", cfgErr)
	}
	keyToHosts := make(map[string][]string)
	for _, h := range hosts {
		if h.KeyFile != "" {
			expanded := expandTilde(h.KeyFile)
			keyToHosts[expanded] = append(keyToHosts[expanded], h.Alias)
		}
	}

	var keys []Key
	seen := make(map[string]bool)

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Public key files end in .pub
		if !strings.HasSuffix(name, ".pub") {
			continue
		}
		privName := strings.TrimSuffix(name, ".pub")
		privPath := filepath.Join(sshDir, privName)
		pubPath := filepath.Join(sshDir, name)

		if seen[privPath] {
			continue
		}
		seen[privPath] = true

		// Only include the key if the private key file actually exists.
		if fi, err := os.Stat(privPath); err != nil || !fi.Mode().IsRegular() {
			continue
		}

		k := Key{
			Name:        privName,
			PrivatePath: privPath,
			PublicPath:  pubPath,
			UsedBy:      keyToHosts[privPath],
		}

		// Get fingerprint
		fp, err := fingerprintFunc(pubPath)
		if err == nil {
			k.Fingerprint = fp
		}

		keys = append(keys, k)
	}

	return keys, nil
}

// fingerprintFunc is the replaceable implementation for getting key fingerprints.
var fingerprintFunc = fingerprintReal

func fingerprintReal(pubPath string) (string, error) {
	out, err := exec.Command("ssh-keygen", "-l", "-f", pubPath).Output()
	if err != nil {
		return "", err
	}
	// Output: "256 SHA256:xxxx comment (ED25519)"
	// We want the SHA256:xxx part
	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) >= 2 {
		return parts[1], nil
	}
	return strings.TrimSpace(string(out)), nil
}

// expandTilde expands a leading ~ to the home directory.
func expandTilde(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return home + path[1:]
}
