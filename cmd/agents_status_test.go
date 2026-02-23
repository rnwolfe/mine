package cmd

import (
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/agents"
)

func TestRunAgentsStatus_NotInitialized(t *testing.T) {
	agentsTestEnv(t)

	out := captureStdout(t, func() {
		if err := runAgentsStatus(nil, nil); err != nil {
			t.Errorf("runAgentsStatus: %v", err)
		}
	})

	if !strings.Contains(out, "No agents store yet") {
		t.Errorf("expected 'No agents store yet' in not-initialized output, got:\n%s", out)
	}
	if !strings.Contains(out, "mine agents init") {
		t.Errorf("expected 'mine agents init' hint in not-initialized output, got:\n%s", out)
	}
}

func TestRunAgentsStatus_Initialized_Empty(t *testing.T) {
	agentsTestEnv(t)

	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	out := captureStdout(t, func() {
		if err := runAgentsStatus(nil, nil); err != nil {
			t.Errorf("runAgentsStatus: %v", err)
		}
	})

	dir := agents.Dir()
	if !strings.Contains(out, dir) {
		t.Errorf("expected store dir %q in status output, got:\n%s", dir, out)
	}
	if !strings.Contains(out, "Detected Agents") {
		t.Errorf("expected 'Detected Agents' section header in status output, got:\n%s", out)
	}
	if !strings.Contains(out, "No links configured yet") {
		t.Errorf("expected 'No links configured yet' in empty status output, got:\n%s", out)
	}
}

func TestRunAgentsStatus_Initialized_WithLinks(t *testing.T) {
	agentsTestEnv(t)

	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	// Write a manifest with one link entry pointing to a non-existent target
	// (unlinked state — safe to use in tests without creating real files).
	m := &agents.Manifest{
		Agents: []agents.Agent{
			{Name: "claude", Detected: true, ConfigDir: "/home/user/.claude", Binary: "/usr/local/bin/claude"},
		},
		Links: []agents.LinkEntry{
			{Source: "instructions/AGENTS.md", Target: "/home/user/.claude/CLAUDE.md", Agent: "claude", Mode: "symlink"},
		},
	}
	if err := agents.WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	out := captureStdout(t, func() {
		if err := runAgentsStatus(nil, nil); err != nil {
			t.Errorf("runAgentsStatus: %v", err)
		}
	})

	// The store dir must appear.
	dir := agents.Dir()
	if !strings.Contains(out, dir) {
		t.Errorf("expected store dir %q in status output, got:\n%s", dir, out)
	}
	// The link source should appear.
	if !strings.Contains(out, "instructions/AGENTS.md") {
		t.Errorf("expected link source 'instructions/AGENTS.md' in status output, got:\n%s", out)
	}
	// The link target should appear.
	if !strings.Contains(out, "/home/user/.claude/CLAUDE.md") {
		t.Errorf("expected link target in status output, got:\n%s", out)
	}
	// A link health summary must appear.
	if !strings.Contains(out, "Summary") {
		t.Errorf("expected 'Summary' in status output, got:\n%s", out)
	}
}
