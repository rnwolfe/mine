package ssh

import (
	"os"
	"path/filepath"
	"testing"
)

// --- expandTilde ---

func TestExpandTilde_WithTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	result := expandTilde("~/.ssh/id_ed25519")
	expected := home + "/.ssh/id_ed25519"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestExpandTilde_NoTilde(t *testing.T) {
	result := expandTilde("/absolute/path")
	if result != "/absolute/path" {
		t.Fatalf("expected unchanged path, got %q", result)
	}
}

// --- ListKeysFrom ---

func TestListKeysFrom_Empty(t *testing.T) {
	dir := t.TempDir()
	keys, err := ListKeysFrom(dir, filepath.Join(dir, "config"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys, got %d", len(keys))
	}
}

func TestListKeysFrom_FindsKeyPairs(t *testing.T) {
	dir := t.TempDir()

	// Create fake key pair files
	privPath := filepath.Join(dir, "id_ed25519")
	pubPath := filepath.Join(dir, "id_ed25519.pub")
	if err := os.WriteFile(privPath, []byte("fake-private-key"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pubPath, []byte("ssh-ed25519 AAAAB3NzaC1 test@example"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Stub the fingerprint function so we don't need ssh-keygen
	original := fingerprintFunc
	defer func() { fingerprintFunc = original }()
	fingerprintFunc = func(path string) (string, error) {
		return "SHA256:fakeprint", nil
	}

	configPath := filepath.Join(dir, "config")
	keys, err := ListKeysFrom(dir, configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	k := keys[0]
	if k.Name != "id_ed25519" {
		t.Fatalf("expected name 'id_ed25519', got %q", k.Name)
	}
	if k.PrivatePath != privPath {
		t.Fatalf("expected private path %q, got %q", privPath, k.PrivatePath)
	}
	if k.PublicPath != pubPath {
		t.Fatalf("expected public path %q, got %q", pubPath, k.PublicPath)
	}
	if k.Fingerprint != "SHA256:fakeprint" {
		t.Fatalf("expected fingerprint 'SHA256:fakeprint', got %q", k.Fingerprint)
	}
}

func TestListKeysFrom_SkipsNonPubFiles(t *testing.T) {
	dir := t.TempDir()
	// Only .pub file, no private key file needed - but it still needs to have .pub extension
	// to be counted as a key pair
	if err := os.WriteFile(filepath.Join(dir, "known_hosts"), []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	keys, err := ListKeysFrom(dir, filepath.Join(dir, "config"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys (no .pub files), got %d", len(keys))
	}
}

func TestListKeysFrom_KeyUsedByHost(t *testing.T) {
	dir := t.TempDir()
	sshDir := filepath.Join(dir, "ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}

	// Create fake key pair
	privPath := filepath.Join(sshDir, "id_ed25519")
	pubPath := filepath.Join(sshDir, "id_ed25519.pub")
	if err := os.WriteFile(privPath, []byte("priv"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pubPath, []byte("pub"), 0o644); err != nil {
		t.Fatal(err)
	}

	// SSH config referencing this key
	configContent := "Host myserver\n    HostName 10.0.0.1\n    IdentityFile " + privPath + "\n"
	configPath := filepath.Join(dir, "config")
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatal(err)
	}

	original := fingerprintFunc
	defer func() { fingerprintFunc = original }()
	fingerprintFunc = func(string) (string, error) { return "SHA256:x", nil }

	keys, err := ListKeysFrom(sshDir, configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if len(keys[0].UsedBy) != 1 || keys[0].UsedBy[0] != "myserver" {
		t.Fatalf("expected key used by 'myserver', got %v", keys[0].UsedBy)
	}
}

func TestListKeysFrom_MissingDir(t *testing.T) {
	keys, err := ListKeysFrom("/nonexistent/ssh/dir", "/nonexistent/config")
	if err != nil {
		t.Fatalf("expected nil error for missing dir, got: %v", err)
	}
	if keys != nil {
		t.Fatalf("expected nil keys for missing dir")
	}
}

// --- Keygen (stubbed) ---

func TestKeygen_Stubbed(t *testing.T) {
	original := KeygenFunc
	defer func() { KeygenFunc = original }()

	var gotName, gotComment string
	KeygenFunc = func(name, comment string) (string, error) {
		gotName = name
		gotComment = comment
		return "/fake/path/key.pub", nil
	}

	pubPath, err := Keygen("mykey", "test comment")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pubPath != "/fake/path/key.pub" {
		t.Fatalf("unexpected pubPath: %q", pubPath)
	}
	if gotName != "mykey" {
		t.Fatalf("expected name 'mykey', got %q", gotName)
	}
	if gotComment != "test comment" {
		t.Fatalf("expected comment 'test comment', got %q", gotComment)
	}
}

// --- CopyID (stubbed) ---

func TestCopyID_Stubbed(t *testing.T) {
	original := CopyIDFunc
	defer func() { CopyIDFunc = original }()

	var gotAlias string
	CopyIDFunc = func(alias string) error {
		gotAlias = alias
		return nil
	}

	if err := CopyID("myserver"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAlias != "myserver" {
		t.Fatalf("expected alias 'myserver', got %q", gotAlias)
	}
}
