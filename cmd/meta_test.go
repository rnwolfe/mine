package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// TestMetaHelp_ContainsContrib verifies the meta help output still contains fr, bug, and contrib.
func TestMetaHelp_ContainsContrib(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_ = runMetaHelp(nil, nil)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	for _, want := range []string{"mine meta fr", "mine meta bug", "mine meta contrib"} {
		if !strings.Contains(output, want) {
			t.Errorf("meta help output missing %q\nGot: %s", want, output)
		}
	}
}
