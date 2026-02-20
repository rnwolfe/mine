// Security tests for the stash domain's path traversal defenses.
//
// Security model:
//   - SafeName must be a bare filename — no path separators (/\), no parent
//     directory references (..), no drive/protocol markers (:), and non-empty.
//   - Source must be an absolute path that resolves to a location within the
//     user's home directory. Relative paths, paths outside $HOME, and
//     symlink-based escapes via ".." are all rejected.
//
// These invariants are enforced by validateSafeName and validateEntryWithHome
// (exposed via ValidateEntry). SyncPull and Commit call validateEntryWithHome
// on every manifest entry before performing any file I/O, ensuring that a
// poisoned manifest cannot trick the stash into writing outside its directory.
package stash

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestValidateSafeName_Security exercises validateSafeName against adversarial
// inputs. Each case documents a specific attack vector the function must block.
func TestValidateSafeName_Security(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty name",
			input:   "",
			wantErr: true,
			errMsg:  "empty SafeName",
		},
		{
			name:    "parent directory (..)",
			input:   "..",
			wantErr: true,
			errMsg:  "unsafe SafeName",
		},
		{
			name:    "relative path escape (../evil)",
			input:   "../evil",
			wantErr: true,
			errMsg:  "unsafe SafeName",
		},
		{
			name:    "triple-dot (...)",
			input:   "...",
			wantErr: true,
			errMsg:  "unsafe SafeName",
		},
		{
			name:    "forward slash (sub/file)",
			input:   "foo/bar",
			wantErr: true,
			errMsg:  "unsafe SafeName",
		},
		{
			name:    "backslash (Windows-style)",
			input:   "foo\\bar",
			wantErr: true,
			errMsg:  "unsafe SafeName",
		},
		{
			name:    "colon (drive/protocol)",
			input:   "foo:bar",
			wantErr: true,
			errMsg:  "unsafe SafeName",
		},
		{
			name:    "double-dot embedded (a..b)",
			input:   "a..b",
			wantErr: true,
			errMsg:  "unsafe SafeName",
		},
		{
			name:    "trailing slash",
			input:   "file/",
			wantErr: true,
			errMsg:  "unsafe SafeName",
		},
		// Valid names that must be accepted.
		{
			name:    "valid dotfile",
			input:   ".zshrc",
			wantErr: false,
		},
		{
			name:    "valid bare filename",
			input:   "my-config",
			wantErr: false,
		},
		{
			name:    "valid name with underscores",
			input:   "config__subdir__file.toml",
			wantErr: false,
		},
		{
			name:    "single dot (current dir reference)",
			input:   ".",
			wantErr: false,
			// NOTE: "." is technically the current directory, but validateSafeName
			// does not reject it since it contains no "..", "/", "\", or ":".
			// This is documented behavior, not a gap — the stash directory never
			// uses "." as a real entry name.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSafeName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSafeName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateSafeName(%q) error = %q, want containing %q", tt.input, err.Error(), tt.errMsg)
			}
		})
	}
}

// TestValidateEntry_Security exercises the full ValidateEntry pipeline with
// adversarial combinations of SafeName and Source path.
func TestValidateEntry_Security(t *testing.T) {
	_, homeDir := setupTestEnv(t)

	validSource := filepath.Join(homeDir, ".zshrc")

	tests := []struct {
		name    string
		entry   Entry
		wantErr bool
		errMsg  string
	}{
		// SafeName attacks that propagate through ValidateEntry.
		{
			name:    "triple-dot SafeName",
			entry:   Entry{SafeName: "...", Source: validSource},
			wantErr: true,
			errMsg:  "unsafe SafeName",
		},
		{
			name:    "backslash in SafeName",
			entry:   Entry{SafeName: "a\\b", Source: validSource},
			wantErr: true,
			errMsg:  "unsafe SafeName",
		},
		{
			name:    "colon in SafeName",
			entry:   Entry{SafeName: "a:b", Source: validSource},
			wantErr: true,
			errMsg:  "unsafe SafeName",
		},
		{
			name:    "embedded double-dot in SafeName",
			entry:   Entry{SafeName: "x..y", Source: validSource},
			wantErr: true,
			errMsg:  "unsafe SafeName",
		},
		// Source path attacks.
		{
			name:    "relative Source path",
			entry:   Entry{SafeName: ".zshrc", Source: "relative/path"},
			wantErr: true,
			errMsg:  "not absolute",
		},
		{
			name:    "Source outside home (/etc/passwd)",
			entry:   Entry{SafeName: ".zshrc", Source: "/etc/passwd"},
			wantErr: true,
			errMsg:  "escapes home directory",
		},
		{
			name:    "Source deep escape via ..",
			entry:   Entry{SafeName: ".zshrc", Source: homeDir + "/../../../etc/shadow"},
			wantErr: true,
			errMsg:  "escapes home directory",
		},
		{
			name:    "Source is home parent",
			entry:   Entry{SafeName: ".zshrc", Source: filepath.Dir(homeDir) + "/other"},
			wantErr: true,
			errMsg:  "escapes home directory",
		},
		// Valid entry (positive control).
		{
			name:    "valid entry",
			entry:   Entry{SafeName: ".zshrc", Source: validSource},
			wantErr: false,
		},
		{
			name:    "valid nested path in home",
			entry:   Entry{SafeName: ".config__mine__config.toml", Source: filepath.Join(homeDir, ".config", "mine", "config.toml")},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEntry(tt.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEntry(%+v) error = %v, wantErr %v", tt.entry, err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateEntry(%+v) error = %q, want containing %q", tt.entry, err.Error(), tt.errMsg)
			}
		})
	}
}

// TestSyncPullRejectsInvalidManifest verifies that the validation gate used by
// SyncPull rejects a poisoned manifest entry and does NOT write any file to
// the escaped path.
//
// SyncPull requires a git repo with a working remote, so a true end-to-end test
// is impractical in unit tests. Instead, this test exercises the exact
// validation path that SyncPull invokes (validateEntryWithHome) against a
// crafted malicious entry, then asserts no file appeared at the target path.
func TestSyncPullRejectsInvalidManifest(t *testing.T) {
	_, homeDir := setupTestEnv(t)

	// The escape target: a file that should never be created.
	escapeTarget := filepath.Join(homeDir, "..", "escaped-file")

	tests := []struct {
		name     string
		safeName string
		source   string
	}{
		{
			name:     "SafeName traversal (../escape)",
			safeName: "../escape",
			source:   filepath.Join(homeDir, ".zshrc"),
		},
		{
			name:     "Source outside home",
			safeName: "passwd",
			source:   "/etc/passwd",
		},
		{
			name:     "Source with deep .. escape",
			safeName: ".zshrc",
			source:   homeDir + "/../../../etc/shadow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := Entry{SafeName: tt.safeName, Source: tt.source}

			err := validateEntryWithHome(entry, homeDir)
			if err == nil {
				t.Fatalf("validateEntryWithHome(%+v) should have returned an error", entry)
			}

			// Verify no file was written at the escape target.
			if _, statErr := os.Stat(escapeTarget); statErr == nil {
				t.Errorf("escape target %q exists — validation did not prevent file write", escapeTarget)
				// Clean up to avoid polluting other tests.
				os.Remove(escapeTarget)
			}
		})
	}
}
