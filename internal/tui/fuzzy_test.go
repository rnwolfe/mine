package tui

import "testing"

func TestFuzzyMatch_EmptyQuery(t *testing.T) {
	ok, score := FuzzyMatch("", "anything")
	if !ok {
		t.Fatal("empty query should match everything")
	}
	if score != 0 {
		t.Fatalf("empty query score should be 0, got %d", score)
	}
}

func TestFuzzyMatch_ExactMatch(t *testing.T) {
	ok, score := FuzzyMatch("dev", "dev")
	if !ok {
		t.Fatal("exact match should succeed")
	}
	if score <= 0 {
		t.Fatalf("exact match should have positive score, got %d", score)
	}
}

func TestFuzzyMatch_CaseInsensitive(t *testing.T) {
	ok, _ := FuzzyMatch("DEV", "development")
	if !ok {
		t.Fatal("case insensitive match should succeed")
	}

	ok, _ = FuzzyMatch("dev", "Development")
	if !ok {
		t.Fatal("case insensitive match should succeed (reversed)")
	}
}

func TestFuzzyMatch_SubsequenceMatch(t *testing.T) {
	ok, _ := FuzzyMatch("dv", "development")
	if !ok {
		t.Fatal("subsequence 'dv' should match 'development'")
	}
}

func TestFuzzyMatch_NoMatch(t *testing.T) {
	ok, _ := FuzzyMatch("xyz", "development")
	if ok {
		t.Fatal("'xyz' should not match 'development'")
	}
}

func TestFuzzyMatch_OutOfOrder(t *testing.T) {
	ok, _ := FuzzyMatch("vd", "development")
	if ok {
		t.Fatal("out-of-order characters should not match")
	}
}

func TestFuzzyMatch_ConsecutiveBonus(t *testing.T) {
	// "abc" in "abcdef" is consecutive, should score higher
	// than "abc" in "axbxcxdef" which is spread out with no boundary bonuses.
	_, scoreConsec := FuzzyMatch("abc", "abcdef")
	_, scoreSpread := FuzzyMatch("abc", "axbxcxdef")
	if scoreConsec <= scoreSpread {
		t.Fatalf("consecutive match (%d) should score higher than spread match (%d)",
			scoreConsec, scoreSpread)
	}
}

func TestFuzzyMatch_WordBoundaryBonus(t *testing.T) {
	// "s" at a word boundary should score higher.
	_, scoreBoundary := FuzzyMatch("s", "my-session")
	_, scoreMiddle := FuzzyMatch("s", "myssion")
	if scoreBoundary <= scoreMiddle {
		t.Fatalf("word boundary match (%d) should score higher than middle match (%d)",
			scoreBoundary, scoreMiddle)
	}
}

func TestFuzzyMatch_StartBonus(t *testing.T) {
	_, scoreStart := FuzzyMatch("d", "dev")
	_, scoreEnd := FuzzyMatch("d", "aad")
	if scoreStart <= scoreEnd {
		t.Fatalf("start match (%d) should score higher than end match (%d)",
			scoreStart, scoreEnd)
	}
}

func TestFuzzyMatch_EmptyTarget(t *testing.T) {
	ok, _ := FuzzyMatch("a", "")
	if ok {
		t.Fatal("non-empty query should not match empty target")
	}
}

func TestFuzzyMatch_RealWorldCases(t *testing.T) {
	cases := []struct {
		query, target string
		shouldMatch   bool
	}{
		{"mine", "mine-project", true},
		{"mp", "mine-project", true},
		{"proj", "mine-project", true},
		{"mj", "mine-project", true},  // m...j (j in project)
		{"ssh", "my-ssh-tunnel", true},
		{"tun", "my-ssh-tunnel", true},
		{"ml", "my-ssh-tunnel", true}, // m...l in sequence
	}

	for _, tc := range cases {
		ok, _ := FuzzyMatch(tc.query, tc.target)
		if ok != tc.shouldMatch {
			t.Errorf("FuzzyMatch(%q, %q) = %v, want %v", tc.query, tc.target, ok, tc.shouldMatch)
		}
	}
}
