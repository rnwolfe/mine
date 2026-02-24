package cmd

import (
	"strings"
	"testing"
)

// ── runAgentsList ─────────────────────────────────────────────────────────────

func TestRunAgentsList_NotInitialized(t *testing.T) {
	agentsTestEnv(t)

	out := captureStdout(t, func() {
		if err := runAgentsList(nil, nil); err != nil {
			t.Errorf("runAgentsList: %v", err)
		}
	})

	if !strings.Contains(out, "No agents store yet") {
		t.Errorf("expected 'No agents store yet' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "mine agents init") {
		t.Errorf("expected 'mine agents init' hint in output, got:\n%s", out)
	}
}

func TestRunAgentsList_Initialized(t *testing.T) {
	agentsTestEnv(t)
	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	agentsListType = ""
	out := captureStdout(t, func() {
		if err := runAgentsList(nil, nil); err != nil {
			t.Errorf("runAgentsList: %v", err)
		}
	})

	// Init creates instructions/AGENTS.md, so the store is not empty.
	if !strings.Contains(out, "Agent Configs") {
		t.Errorf("expected 'Agent Configs' header in output, got:\n%s", out)
	}
}

func TestRunAgentsList_UnknownTypeReturnsError(t *testing.T) {
	agentsTestEnv(t)
	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	agentsListType = "bogus"
	defer func() { agentsListType = "" }()

	captureStdout(t, func() {
		err := runAgentsList(nil, nil)
		if err == nil {
			t.Error("runAgentsList with unknown type = nil, want error")
		}
		if err != nil && !strings.Contains(err.Error(), "unknown type") {
			t.Errorf("error = %q, want 'unknown type' in message", err.Error())
		}
	})
}

func TestRunAgentsList_HappyPath_WithSkill(t *testing.T) {
	agentsTestEnv(t)
	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	// Add a skill via the domain layer directly.
	captureStdout(t, func() {
		if err := runAgentsAddSkill(nil, []string{"code-review"}); err != nil {
			t.Fatalf("runAgentsAddSkill: %v", err)
		}
	})

	agentsListType = ""
	out := captureStdout(t, func() {
		if err := runAgentsList(nil, nil); err != nil {
			t.Errorf("runAgentsList: %v", err)
		}
	})

	if !strings.Contains(out, "code-review") {
		t.Errorf("expected 'code-review' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Skills") {
		t.Errorf("expected 'Skills' section in output, got:\n%s", out)
	}
}

func TestRunAgentsList_TypeFilter_Skills(t *testing.T) {
	agentsTestEnv(t)
	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	captureStdout(t, func() {
		if err := runAgentsAddSkill(nil, []string{"my-skill"}); err != nil {
			t.Fatalf("runAgentsAddSkill: %v", err)
		}
	})

	agentsListType = "skills"
	defer func() { agentsListType = "" }()

	out := captureStdout(t, func() {
		if err := runAgentsList(nil, nil); err != nil {
			t.Errorf("runAgentsList: %v", err)
		}
	})

	if !strings.Contains(out, "my-skill") {
		t.Errorf("expected 'my-skill' in output, got:\n%s", out)
	}
}

func TestRunAgentsList_EmptyType_ShowsHint(t *testing.T) {
	agentsTestEnv(t)
	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	agentsListType = "skills"
	defer func() { agentsListType = "" }()

	out := captureStdout(t, func() {
		if err := runAgentsList(nil, nil); err != nil {
			t.Errorf("runAgentsList: %v", err)
		}
	})

	// When filtering to skills and none exist, should show add hint.
	if !strings.Contains(out, "mine agents add skill") {
		t.Errorf("expected 'mine agents add skill' hint in output, got:\n%s", out)
	}
}

func TestRunAgentsList_InstructionsType_NoAddHint(t *testing.T) {
	agentsTestEnv(t)
	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	agentsListType = "instructions"
	defer func() { agentsListType = "" }()

	out := captureStdout(t, func() {
		if err := runAgentsList(nil, nil); err != nil {
			t.Errorf("runAgentsList: %v", err)
		}
	})

	// "instructions" type has no add subcommand, so no hint should appear.
	if strings.Contains(out, "mine agents add instruction") {
		t.Errorf("should not show 'mine agents add instruction' hint for instructions type, got:\n%s", out)
	}
}
