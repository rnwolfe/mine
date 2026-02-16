package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/version"
)

func TestVersionCommand(t *testing.T) {
	tests := []struct {
		name        string
		shortFlag   bool
		wantContain string
	}{
		{
			name:        "full version includes mine prefix",
			shortFlag:   false,
			wantContain: "mine",
		},
		{
			name:        "full version includes version number",
			shortFlag:   false,
			wantContain: version.Short(),
		},
		{
			name:        "short version outputs only version number",
			shortFlag:   true,
			wantContain: version.Short(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the flag
			versionShort = tt.shortFlag

			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run the command
			versionCmd.Run(nil, nil)

			// Restore stdout
			w.Close()
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := strings.TrimSpace(buf.String())

			// Verify output contains expected string
			if !strings.Contains(output, tt.wantContain) {
				t.Errorf("output missing %q\nGot: %s", tt.wantContain, output)
			}

			// Additional check: short flag should NOT contain "mine" prefix
			if tt.shortFlag && strings.Contains(output, "mine") {
				t.Errorf("short version should not contain 'mine' prefix\nGot: %s", output)
			}

			// Additional check: short version should be exactly the version string
			if tt.shortFlag && output != version.Short() {
				t.Errorf("short version output mismatch\nWant: %s\nGot: %s", version.Short(), output)
			}
		})
	}
}
