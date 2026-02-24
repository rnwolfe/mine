package todo

import (
	"sort"
	"time"
)

// UrgencyWeights holds configurable weights for the urgency scoring algorithm.
type UrgencyWeights struct {
	Overdue       int // bonus for any task past its due date
	ScheduleToday int // weight for today-scheduled tasks
	ScheduleSoon  int // weight for soon-scheduled tasks
	ScheduleLater int // weight for later-scheduled tasks
	PriorityCrit  int // weight for critical priority
	PriorityHigh  int // weight for high priority
	PriorityMed   int // weight for medium priority
	PriorityLow   int // weight for low priority
	AgeCap        int // maximum age bonus (days)
	ProjectBoost  int // bonus when task belongs to the current project
}

// DefaultUrgencyWeights returns the default urgency scoring weights.
func DefaultUrgencyWeights() UrgencyWeights {
	return UrgencyWeights{
		Overdue:       100,
		ScheduleToday: 50,
		ScheduleSoon:  20,
		ScheduleLater: 5,
		PriorityCrit:  40,
		PriorityHigh:  30,
		PriorityMed:   20,
		PriorityLow:   10,
		AgeCap:        30,
		ProjectBoost:  10,
	}
}

// UrgencyScore computes the urgency score for a single todo.
// Higher score means more urgent. now is the reference time (typically time.Now()).
// currentProjectPath is the active project context; nil means no project.
func UrgencyScore(t Todo, now time.Time, currentProjectPath *string, w UrgencyWeights) int {
	score := 0
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Overdue bonus: any task past its due date gets a large bonus.
	if t.DueDate != nil {
		dueDay := time.Date(t.DueDate.Year(), t.DueDate.Month(), t.DueDate.Day(), 0, 0, 0, 0, now.Location())
		if dueDay.Before(today) {
			score += w.Overdue
		}
	}

	// Schedule weight.
	switch t.Schedule {
	case ScheduleToday:
		score += w.ScheduleToday
	case ScheduleSoon:
		score += w.ScheduleSoon
	case ScheduleLater:
		score += w.ScheduleLater
	// Someday: 0 — also excluded from next results at the query level.
	}

	// Priority weight.
	switch t.Priority {
	case PrioCrit:
		score += w.PriorityCrit
	case PrioHigh:
		score += w.PriorityHigh
	case PrioMedium:
		score += w.PriorityMed
	case PrioLow:
		score += w.PriorityLow
	}

	// Age bonus: +1 per day since creation, capped at AgeCap.
	createdDay := time.Date(t.CreatedAt.Year(), t.CreatedAt.Month(), t.CreatedAt.Day(), 0, 0, 0, 0, now.Location())
	age := int(today.Sub(createdDay).Hours() / 24)
	if age > w.AgeCap {
		age = w.AgeCap
	}
	if age > 0 {
		score += age
	}

	// Project boost: bonus if task belongs to the current project.
	if currentProjectPath != nil && t.ProjectPath != nil && *t.ProjectPath == *currentProjectPath {
		score += w.ProjectBoost
	}

	return score
}

// SortByUrgency sorts todos in-place from highest to lowest urgency score.
// Ties are broken by creation date (older tasks first).
func SortByUrgency(todos []Todo, now time.Time, currentProjectPath *string, w UrgencyWeights) {
	sort.SliceStable(todos, func(i, j int) bool {
		si := UrgencyScore(todos[i], now, currentProjectPath, w)
		sj := UrgencyScore(todos[j], now, currentProjectPath, w)
		if si != sj {
			return si > sj
		}
		// Tiebreaker: older task first (prevents staleness).
		return todos[i].CreatedAt.Before(todos[j].CreatedAt)
	})
}
