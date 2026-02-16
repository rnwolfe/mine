package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestAboutCommand(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the command
	runAbout(nil, nil)

	// Restore stdout
	w.Close()
	os.Stdout = old

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains expected information
	tests := []struct {
		name     string
		contains string
	}{
		{"has header", "About mine"},
		{"has version label", "Version"},
		{"has repository label", "Repository"},
		{"has repository URL", "https://github.com/rnwolfe/mine"},
		{"has license label", "License"},
		{"has license type", "MIT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(output, tt.contains) {
				t.Errorf("output missing %q\nGot: %s", tt.contains, output)
			}
		})
	}
}
