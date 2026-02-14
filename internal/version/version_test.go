package version

import (
	"strings"
	"testing"
)

func TestFull(t *testing.T) {
	result := Full()
	if result == "" {
		t.Fatal("Full() returned empty string")
	}
	if !strings.Contains(result, Version) {
		t.Errorf("Full() %q does not contain version %q", result, Version)
	}
}

func TestShort(t *testing.T) {
	result := Short()
	if result != Version {
		t.Errorf("Short() = %q, want %q", result, Version)
	}
}
