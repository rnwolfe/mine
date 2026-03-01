package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/rnwolfe/mine/internal/dig"
	"github.com/rnwolfe/mine/internal/store"
)

func TestGatherStatus_EmptyDB(t *testing.T) {
	configTestEnv(t)

	data := gatherStatus()
	if data.DigStreak != 0 {
		t.Errorf("DigStreak = %d, want 0 for empty db", data.DigStreak)
	}
	if data.DigTotalMins != 0 {
		t.Errorf("DigTotalMins = %d, want 0 for empty db", data.DigTotalMins)
	}
	if data.Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestGatherStatus_WithDigData(t *testing.T) {
	configTestEnv(t)

	// Seed dig data via the domain store.
	{
		db, err := store.Open()
		if err != nil {
			t.Fatalf("store.Open: %v", err)
		}
		defer db.Close()

		ds := dig.NewStore(db.Conn())
		if _, err := ds.RecordSession(30*time.Minute, nil, true, time.Now().Add(-30*time.Minute)); err != nil {
			t.Fatalf("RecordSession: %v", err)
		}
	}

	data := gatherStatus()
	if data.DigStreak != 1 {
		t.Errorf("DigStreak = %d, want 1", data.DigStreak)
	}
	if data.DigTotalMins != 30 {
		t.Errorf("DigTotalMins = %d, want 30", data.DigTotalMins)
	}
}

func TestRunStatus_HumanReadable(t *testing.T) {
	configTestEnv(t)

	prevStatusJSON := statusJSON
	prevStatusPrompt := statusPrompt
	t.Cleanup(func() {
		statusJSON = prevStatusJSON
		statusPrompt = prevStatusPrompt
	})
	statusJSON = false
	statusPrompt = false

	output := captureStdout(t, func() {
		if err := runStatus(nil, nil); err != nil {
			t.Fatalf("runStatus: %v", err)
		}
	})

	if !strings.Contains(output, "Todos:") {
		t.Errorf("expected 'Todos:' in output, got: %q", output)
	}
}

func TestFormatPromptSegment_Empty(t *testing.T) {
	data := StatusData{}
	seg := formatPromptSegment(data)
	if seg != "" {
		t.Errorf("expected empty segment, got %q", seg)
	}
}

func TestFormatPromptSegment_WithTodos(t *testing.T) {
	data := StatusData{OpenTodos: 3}
	seg := formatPromptSegment(data)
	if seg != "[3t]" {
		t.Errorf("expected '[3t]', got %q", seg)
	}
}

func TestFormatPromptSegment_WithStreak(t *testing.T) {
	data := StatusData{DigStreak: 5}
	seg := formatPromptSegment(data)
	if seg != "[5d]" {
		t.Errorf("expected '[5d]', got %q", seg)
	}
}

func TestFormatPromptSegment_WithBoth(t *testing.T) {
	data := StatusData{OpenTodos: 2, DigStreak: 4}
	seg := formatPromptSegment(data)
	if seg != "[2t|4d]" {
		t.Errorf("expected '[2t|4d]', got %q", seg)
	}
}
