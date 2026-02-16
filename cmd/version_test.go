package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/version"
)

func TestVersionCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantOut string
	}{
		{
			name:    "default version output",
			args:    []string{},
			wantOut: fmt.Sprintf("mine %s", version.Full()),
		},
		{
			name:    "short flag version output",
			args:    []string{"--short"},
			wantOut: version.Short(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the flag to default state
			versionShort = false
			versionCmd.ResetFlags()
			versionCmd.Flags().BoolVar(&versionShort, "short", false, "Print only the version number")

			// Parse the args to set the flag value through cobra
			versionCmd.SetArgs(tt.args)
			err := versionCmd.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("flag parsing failed: %v", err)
			}

			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run the command directly (flag is already set by ParseFlags)
			versionCmd.Run(nil, nil)

			// Restore stdout
			w.Close()
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := strings.TrimSpace(buf.String())

			// Assert exact output
			if output != tt.wantOut {
				t.Errorf("output mismatch\nWant: %s\nGot: %s", tt.wantOut, output)
			}
		})
	}
}
