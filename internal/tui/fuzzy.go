package tui

import "strings"

// FuzzyMatch checks whether all characters of query appear in target in order
// (case-insensitive). Returns whether it matched and a relevance score.
//
// Scoring rewards:
//   - consecutive character matches
//   - matches at the start of the string
//   - matches at word boundaries (after space, /, -, _)
func FuzzyMatch(query, target string) (bool, int) {
	if query == "" {
		return true, 0
	}

	q := strings.ToLower(query)
	t := strings.ToLower(target)

	qi := 0
	score := 0
	consecutive := 0

	for ti := 0; ti < len(t) && qi < len(q); ti++ {
		if t[ti] == q[qi] {
			qi++
			consecutive++
			score += consecutive // reward consecutive runs

			// Bonus for matching at the very start.
			if ti == 0 {
				score += 3
			}

			// Bonus for word-boundary match.
			if ti > 0 {
				prev := t[ti-1]
				if prev == ' ' || prev == '/' || prev == '-' || prev == '_' || prev == '.' {
					score += 2
				}
			}
		} else {
			consecutive = 0
		}
	}

	return qi == len(q), score
}
