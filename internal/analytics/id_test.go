package analytics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetOrCreateID_GeneratesValidUUID(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	// Ensure the mine data dir exists
	mineDir := filepath.Join(tmp, "mine")
	if err := os.MkdirAll(mineDir, 0o755); err != nil {
		t.Fatal(err)
	}

	id, err := GetOrCreateID()
	if err != nil {
		t.Fatalf("GetOrCreateID failed: %v", err)
	}

	if !isValidUUID(id) {
		t.Fatalf("expected valid UUID, got %q", id)
	}
}

func TestGetOrCreateID_ReturnsSameID(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	mineDir := filepath.Join(tmp, "mine")
	if err := os.MkdirAll(mineDir, 0o755); err != nil {
		t.Fatal(err)
	}

	id1, err := GetOrCreateID()
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	id2, err := GetOrCreateID()
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if id1 != id2 {
		t.Fatalf("expected same ID on second call, got %q and %q", id1, id2)
	}
}

func TestGetOrCreateID_RegeneratesCorruptFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	mineDir := filepath.Join(tmp, "mine")
	if err := os.MkdirAll(mineDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write garbage to the ID file
	idPath := filepath.Join(mineDir, idFileName)
	if err := os.WriteFile(idPath, []byte("not-a-uuid\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	id, err := GetOrCreateID()
	if err != nil {
		t.Fatalf("GetOrCreateID failed on corrupt file: %v", err)
	}

	if !isValidUUID(id) {
		t.Fatalf("expected valid UUID after regeneration, got %q", id)
	}
	if id == "not-a-uuid" {
		t.Fatal("should have regenerated, but returned corrupt value")
	}
}

func TestGetOrCreateID_RegeneratesEmptyFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	mineDir := filepath.Join(tmp, "mine")
	if err := os.MkdirAll(mineDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write empty file
	idPath := filepath.Join(mineDir, idFileName)
	if err := os.WriteFile(idPath, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	id, err := GetOrCreateID()
	if err != nil {
		t.Fatalf("GetOrCreateID failed on empty file: %v", err)
	}

	if !isValidUUID(id) {
		t.Fatalf("expected valid UUID after regeneration, got %q", id)
	}
}

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"a1b2c3d4-e5f6-7890-abcd-ef1234567890", true},
		{"00000000-0000-0000-0000-000000000000", true},
		{"not-a-uuid", false},
		{"", false},
		{"a1b2c3d4-e5f6-7890-abcd", false}, // truncated
	}

	for _, tt := range tests {
		if got := isValidUUID(tt.input); got != tt.want {
			t.Errorf("isValidUUID(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
