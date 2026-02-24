package cmd

import (
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/agents"
)

// ── runAgentsAddSkill ─────────────────────────────────────────────────────────

func TestRunAgentsAddSkill_NotInitialized(t *testing.T) {
	agentsTestEnv(t)

	out := captureStdout(t, func() {
		if err := runAgentsAddSkill(nil, []string{"my-skill"}); err != nil {
			t.Errorf("runAgentsAddSkill: %v", err)
		}
	})

	if !strings.Contains(out, "No agents store yet") {
		t.Errorf("expected 'No agents store yet' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "mine agents init") {
		t.Errorf("expected 'mine agents init' hint in output, got:\n%s", out)
	}
}

func TestRunAgentsAddSkill_HappyPath(t *testing.T) {
	agentsTestEnv(t)
	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	out := captureStdout(t, func() {
		if err := runAgentsAddSkill(nil, []string{"code-review"}); err != nil {
			t.Errorf("runAgentsAddSkill: %v", err)
		}
	})

	if !strings.Contains(out, "code-review") {
		t.Errorf("expected skill name in output, got:\n%s", out)
	}
	if !strings.Contains(out, "skills/code-review") {
		t.Errorf("expected relative path in output, got:\n%s", out)
	}
}

func TestRunAgentsAddSkill_InvalidName(t *testing.T) {
	agentsTestEnv(t)
	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	captureStdout(t, func() {
		err := runAgentsAddSkill(nil, []string{"Bad Name"})
		if err == nil {
			t.Error("runAgentsAddSkill with invalid name = nil, want error")
		}
	})
}

func TestRunAgentsAddSkill_DuplicateReturnsError(t *testing.T) {
	agentsTestEnv(t)
	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	captureStdout(t, func() {
		if err := runAgentsAddSkill(nil, []string{"my-skill"}); err != nil {
			t.Fatalf("first runAgentsAddSkill: %v", err)
		}
	})

	captureStdout(t, func() {
		err := runAgentsAddSkill(nil, []string{"my-skill"})
		if err == nil {
			t.Error("second runAgentsAddSkill (duplicate) = nil, want error")
		}
	})
}

// ── runAgentsAddCommand ───────────────────────────────────────────────────────

func TestRunAgentsAddCommand_NotInitialized(t *testing.T) {
	agentsTestEnv(t)

	out := captureStdout(t, func() {
		if err := runAgentsAddCommand(nil, []string{"deploy"}); err != nil {
			t.Errorf("runAgentsAddCommand: %v", err)
		}
	})

	if !strings.Contains(out, "No agents store yet") {
		t.Errorf("expected 'No agents store yet' in output, got:\n%s", out)
	}
}

func TestRunAgentsAddCommand_HappyPath(t *testing.T) {
	agentsTestEnv(t)
	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	out := captureStdout(t, func() {
		if err := runAgentsAddCommand(nil, []string{"deploy"}); err != nil {
			t.Errorf("runAgentsAddCommand: %v", err)
		}
	})

	if !strings.Contains(out, "deploy") {
		t.Errorf("expected command name in output, got:\n%s", out)
	}
	if !strings.Contains(out, "commands/deploy.md") {
		t.Errorf("expected relative path in output, got:\n%s", out)
	}

	// Verify the file was actually created.
	result, err := agents.List(agents.ListOptions{Type: "commands"})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(result.Commands) != 1 || result.Commands[0].Name != "deploy" {
		t.Errorf("expected 'deploy' command in store, got: %+v", result.Commands)
	}
}

// ── runAgentsAddAgent ─────────────────────────────────────────────────────────

func TestRunAgentsAddAgent_NotInitialized(t *testing.T) {
	agentsTestEnv(t)

	out := captureStdout(t, func() {
		if err := runAgentsAddAgent(nil, []string{"reviewer"}); err != nil {
			t.Errorf("runAgentsAddAgent: %v", err)
		}
	})

	if !strings.Contains(out, "No agents store yet") {
		t.Errorf("expected 'No agents store yet' in output, got:\n%s", out)
	}
}

func TestRunAgentsAddAgent_HappyPath(t *testing.T) {
	agentsTestEnv(t)
	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	out := captureStdout(t, func() {
		if err := runAgentsAddAgent(nil, []string{"reviewer"}); err != nil {
			t.Errorf("runAgentsAddAgent: %v", err)
		}
	})

	if !strings.Contains(out, "reviewer") {
		t.Errorf("expected agent name in output, got:\n%s", out)
	}
	if !strings.Contains(out, "agents/reviewer.md") {
		t.Errorf("expected relative path in output, got:\n%s", out)
	}
}

// ── runAgentsAddRule ──────────────────────────────────────────────────────────

func TestRunAgentsAddRule_NotInitialized(t *testing.T) {
	agentsTestEnv(t)

	out := captureStdout(t, func() {
		if err := runAgentsAddRule(nil, []string{"style"}); err != nil {
			t.Errorf("runAgentsAddRule: %v", err)
		}
	})

	if !strings.Contains(out, "No agents store yet") {
		t.Errorf("expected 'No agents store yet' in output, got:\n%s", out)
	}
}

func TestRunAgentsAddRule_HappyPath(t *testing.T) {
	agentsTestEnv(t)
	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	out := captureStdout(t, func() {
		if err := runAgentsAddRule(nil, []string{"style"}); err != nil {
			t.Errorf("runAgentsAddRule: %v", err)
		}
	})

	if !strings.Contains(out, "style") {
		t.Errorf("expected rule name in output, got:\n%s", out)
	}
	if !strings.Contains(out, "rules/style.md") {
		t.Errorf("expected relative path in output, got:\n%s", out)
	}
}
