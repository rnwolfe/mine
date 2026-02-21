package tips

import (
	"testing"
	"time"
)

func TestAll_NonEmpty(t *testing.T) {
	all := All()
	if len(all) == 0 {
		t.Fatal("All() returned empty slice")
	}
	if len(all) < 20 {
		t.Fatalf("All() returned %d tips, want at least 20", len(all))
	}
}

func TestAll_NoEmptyStrings(t *testing.T) {
	for i, tip := range All() {
		if tip == "" {
			t.Errorf("All()[%d] is empty string", i)
		}
	}
}

func TestDaily_Deterministic(t *testing.T) {
	// Same day should return the same tip.
	day := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)

	if Daily(day) != Daily(day2) {
		t.Error("Daily() returned different tips for the same day")
	}
}

func TestDaily_DifferentDays(t *testing.T) {
	// Across 10 sequential days, at least one tip should differ.
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	tips := make(map[string]bool)
	for i := 0; i < 10; i++ {
		tips[Daily(start.AddDate(0, 0, i))] = true
	}
	if len(tips) < 2 {
		t.Error("Daily() returned the same tip for 10 consecutive days — rotation broken")
	}
}

func TestDaily_ReturnsTipFromPool(t *testing.T) {
	all := All()
	allSet := make(map[string]bool, len(all))
	for _, tip := range all {
		allSet[tip] = true
	}

	day := time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC)
	tip := Daily(day)
	if !allSet[tip] {
		t.Errorf("Daily() returned a tip not in All(): %q", tip)
	}
}

func TestRandom_ReturnsTipFromPool(t *testing.T) {
	all := All()
	allSet := make(map[string]bool, len(all))
	for _, tip := range all {
		allSet[tip] = true
	}

	now := time.Now()
	tip := Random(now)
	if !allSet[tip] {
		t.Errorf("Random() returned a tip not in All(): %q", tip)
	}
}
