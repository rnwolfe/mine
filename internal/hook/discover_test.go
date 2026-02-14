package hook

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseHookFilename(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		pattern string
		stage   Stage
		wantErr bool
	}{
		{"simple", "todo.add.preexec.sh", "todo.add", StagePreexec, false},
		{"wildcard", "todo.*.notify.py", "todo.*", StageNotify, false},
		{"global wildcard", "*.postexec.sh", "*", StagePostexec, false},
		{"prevalidate", "todo.add.prevalidate.bash", "todo.add", StagePrevalidate, false},
		{"no extension", "todo.add.preexec", "", "", true}, // requires file extension
		{"invalid stage", "todo.add.badstage.sh", "", "", true},
		{"no dots", "simple", "", "", true},
		{"only stage", ".preexec.sh", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := parseHookFilename(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseHookFilename(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if h.Pattern != tt.pattern {
				t.Errorf("Pattern = %q, want %q", h.Pattern, tt.pattern)
			}
			if h.Stage != tt.stage {
				t.Errorf("Stage = %q, want %q", h.Stage, tt.stage)
			}
		})
	}
}

func TestParseStage(t *testing.T) {
	tests := []struct {
		input   string
		want    Stage
		wantErr bool
	}{
		{"prevalidate", StagePrevalidate, false},
		{"preexec", StagePreexec, false},
		{"postexec", StagePostexec, false},
		{"notify", StageNotify, false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseStage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseStage(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseStage(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDiscover(t *testing.T) {
	// Create temp hooks dir
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	hooksDir := filepath.Join(dir, "mine", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create valid hook script
	script := "#!/bin/bash\ncat"
	validPath := filepath.Join(hooksDir, "todo.add.preexec.sh")
	if err := os.WriteFile(validPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create non-executable file (should be skipped)
	nonExec := filepath.Join(hooksDir, "todo.done.notify.sh")
	if err := os.WriteFile(nonExec, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create file with invalid name (should be skipped)
	invalid := filepath.Join(hooksDir, "README.md")
	if err := os.WriteFile(invalid, []byte("docs"), 0o755); err != nil {
		t.Fatal(err)
	}

	hooks, err := Discover()
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}

	if len(hooks) != 1 {
		t.Fatalf("Discover() found %d hooks, want 1", len(hooks))
	}

	h := hooks[0]
	if h.Pattern != "todo.add" {
		t.Errorf("Pattern = %q, want %q", h.Pattern, "todo.add")
	}
	if h.Stage != StagePreexec {
		t.Errorf("Stage = %q, want %q", h.Stage, StagePreexec)
	}
	if h.Path != validPath {
		t.Errorf("Path = %q, want %q", h.Path, validPath)
	}
}

func TestDiscoverEmptyDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	// No hooks dir exists
	hooks, err := Discover()
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if hooks != nil {
		t.Errorf("Discover() = %v, want nil", hooks)
	}
}

func TestCreateHookScript(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	path, err := CreateHookScript("todo.add", StagePreexec)
	if err != nil {
		t.Fatalf("CreateHookScript() error: %v", err)
	}

	// Verify file exists and is executable
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error: %v", path, err)
	}
	if info.Mode()&0o111 == 0 {
		t.Error("created script is not executable")
	}

	// Verify it contains expected content
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if content[:2] != "#!" {
		t.Error("script missing shebang")
	}

	// Creating same hook again should fail
	_, err = CreateHookScript("todo.add", StagePreexec)
	if err == nil {
		t.Error("expected error creating duplicate hook")
	}
}
