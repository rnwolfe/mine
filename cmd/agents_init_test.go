package cmd

import (
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/agents"
)

func TestRunAgentsInit_PrintsLocationAndTip(t *testing.T) {
	agentsTestEnv(t)

	out := captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Errorf("runAgentsInit: %v", err)
		}
	})

	dir := agents.Dir()
	if !strings.Contains(out, dir) {
		t.Errorf("expected store location %q in output, got:\n%s", dir, out)
	}
	if !strings.Contains(out, "instructions/AGENTS.md") {
		t.Errorf("expected 'instructions/AGENTS.md' tip in output, got:\n%s", out)
	}
}

func TestRunAgentsInit_CreatesStore(t *testing.T) {
	agentsTestEnv(t)

	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	if !agents.IsInitialized() {
		t.Error("expected agents store to be initialized after runAgentsInit")
	}
}

func TestRunAgentsInit_Idempotent(t *testing.T) {
	agentsTestEnv(t)

	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("first runAgentsInit: %v", err)
		}
	})
	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Errorf("second runAgentsInit (idempotency): %v", err)
		}
	})
}
