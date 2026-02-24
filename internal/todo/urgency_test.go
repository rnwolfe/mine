package todo

import (
	"testing"
	"time"
)

// baseTime is a fixed reference time for deterministic tests.
var baseTime = time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)

func defaultWeights() UrgencyWeights {
	return DefaultUrgencyWeights()
}

func makeTime(daysAgo int) time.Time {
	return baseTime.AddDate(0, 0, -daysAgo)
}

func timePtr(t time.Time) *time.Time { return &t }

func TestUrgencyScore_OverdueBonus(t *testing.T) {
	w := defaultWeights()

	yesterday := makeTime(1)
	withDue := Todo{
		Title:     "overdue task",
		Priority:  PrioMedium,
		Schedule:  ScheduleLater,
		DueDate:   &yesterday,
		CreatedAt: baseTime,
	}
	withoutDue := Todo{
		Title:     "no due date",
		Priority:  PrioMedium,
		Schedule:  ScheduleLater,
		CreatedAt: baseTime,
	}

	scoreWith := UrgencyScore(withDue, baseTime, nil, w)
	scoreWithout := UrgencyScore(withoutDue, baseTime, nil, w)

	if scoreWith <= scoreWithout {
		t.Errorf("overdue task should score higher: got %d vs %d", scoreWith, scoreWithout)
	}
	// Overdue bonus should be +100
	diff := scoreWith - scoreWithout
	if diff != w.Overdue {
		t.Errorf("expected overdue diff %d, got %d", w.Overdue, diff)
	}
}

func TestUrgencyScore_ScheduleWeights(t *testing.T) {
	w := defaultWeights()

	base := Todo{
		Priority:  PrioMedium,
		CreatedAt: baseTime,
	}

	todayT := base
	todayT.Schedule = ScheduleToday

	soonT := base
	soonT.Schedule = ScheduleSoon

	laterT := base
	laterT.Schedule = ScheduleLater

	somedayT := base
	somedayT.Schedule = ScheduleSomeday

	scoreToday := UrgencyScore(todayT, baseTime, nil, w)
	scoreSoon := UrgencyScore(soonT, baseTime, nil, w)
	scoreLater := UrgencyScore(laterT, baseTime, nil, w)
	scoreSomeday := UrgencyScore(somedayT, baseTime, nil, w)

	if scoreToday <= scoreSoon {
		t.Errorf("today should score higher than soon: %d vs %d", scoreToday, scoreSoon)
	}
	if scoreSoon <= scoreLater {
		t.Errorf("soon should score higher than later: %d vs %d", scoreSoon, scoreLater)
	}
	if scoreLater <= scoreSomeday {
		t.Errorf("later should score higher than someday: %d vs %d", scoreLater, scoreSomeday)
	}
	if scoreSomeday != w.PriorityMed {
		t.Errorf("someday schedule should add 0, expected score %d, got %d", w.PriorityMed, scoreSomeday)
	}
}

func TestUrgencyScore_PriorityWeights(t *testing.T) {
	w := defaultWeights()

	base := Todo{Schedule: ScheduleLater, CreatedAt: baseTime}

	crit := base
	crit.Priority = PrioCrit

	high := base
	high.Priority = PrioHigh

	med := base
	med.Priority = PrioMedium

	low := base
	low.Priority = PrioLow

	scoreCrit := UrgencyScore(crit, baseTime, nil, w)
	scoreHigh := UrgencyScore(high, baseTime, nil, w)
	scoreMed := UrgencyScore(med, baseTime, nil, w)
	scoreLow := UrgencyScore(low, baseTime, nil, w)

	if scoreCrit <= scoreHigh {
		t.Errorf("crit should score higher than high: %d vs %d", scoreCrit, scoreHigh)
	}
	if scoreHigh <= scoreMed {
		t.Errorf("high should score higher than med: %d vs %d", scoreHigh, scoreMed)
	}
	if scoreMed <= scoreLow {
		t.Errorf("med should score higher than low: %d vs %d", scoreMed, scoreLow)
	}
}

func TestUrgencyScore_AgeBonus(t *testing.T) {
	w := defaultWeights()

	old := Todo{
		Priority:  PrioMedium,
		Schedule:  ScheduleLater,
		CreatedAt: makeTime(10),
	}
	brand := Todo{
		Priority:  PrioMedium,
		Schedule:  ScheduleLater,
		CreatedAt: baseTime,
	}

	scoreOld := UrgencyScore(old, baseTime, nil, w)
	scoreBrand := UrgencyScore(brand, baseTime, nil, w)

	if scoreOld <= scoreBrand {
		t.Errorf("older task should score higher: %d vs %d", scoreOld, scoreBrand)
	}
	diff := scoreOld - scoreBrand
	if diff != 10 {
		t.Errorf("expected age diff of 10, got %d", diff)
	}
}

func TestUrgencyScore_AgeCap(t *testing.T) {
	w := defaultWeights() // AgeCap = 30

	veryOld := Todo{
		Priority:  PrioMedium,
		Schedule:  ScheduleLater,
		CreatedAt: makeTime(100), // 100 days ago, well past cap
	}
	atCap := Todo{
		Priority:  PrioMedium,
		Schedule:  ScheduleLater,
		CreatedAt: makeTime(30), // exactly at cap
	}
	pastCap := Todo{
		Priority:  PrioMedium,
		Schedule:  ScheduleLater,
		CreatedAt: makeTime(50), // past cap — should equal atCap
	}

	scoreVeryOld := UrgencyScore(veryOld, baseTime, nil, w)
	scoreAtCap := UrgencyScore(atCap, baseTime, nil, w)
	scorePastCap := UrgencyScore(pastCap, baseTime, nil, w)

	if scoreVeryOld != scoreAtCap {
		t.Errorf("very old and at-cap should score equally (capped): %d vs %d", scoreVeryOld, scoreAtCap)
	}
	if scorePastCap != scoreAtCap {
		t.Errorf("past-cap and at-cap should score equally: %d vs %d", scorePastCap, scoreAtCap)
	}
}

func TestUrgencyScore_ProjectBoost(t *testing.T) {
	w := defaultWeights()

	projPath := "/projects/myapp"

	withProj := Todo{
		Priority:    PrioMedium,
		Schedule:    ScheduleLater,
		CreatedAt:   baseTime,
		ProjectPath: &projPath,
	}
	withoutProj := Todo{
		Priority:  PrioMedium,
		Schedule:  ScheduleLater,
		CreatedAt: baseTime,
	}

	// No project context: no boost for either
	scoreWithNoCtx := UrgencyScore(withProj, baseTime, nil, w)
	scoreWithoutNoCtx := UrgencyScore(withoutProj, baseTime, nil, w)
	if scoreWithNoCtx != scoreWithoutNoCtx {
		t.Errorf("without project context, scores should be equal: %d vs %d", scoreWithNoCtx, scoreWithoutNoCtx)
	}

	// Project context matches: boost applies
	scoreWithCtx := UrgencyScore(withProj, baseTime, &projPath, w)
	scoreWithoutCtx := UrgencyScore(withoutProj, baseTime, &projPath, w)
	diff := scoreWithCtx - scoreWithoutCtx
	if diff != w.ProjectBoost {
		t.Errorf("expected project boost diff %d, got %d", w.ProjectBoost, diff)
	}

	// Different project context: no boost
	otherProj := "/projects/other"
	scoreOtherCtx := UrgencyScore(withProj, baseTime, &otherProj, w)
	if scoreOtherCtx != scoreWithNoCtx {
		t.Errorf("different project context should not boost: %d vs %d", scoreOtherCtx, scoreWithNoCtx)
	}
}

func TestUrgencyScore_OverdueAlwaysHigherThanNonOverdue(t *testing.T) {
	w := defaultWeights()

	// Overdue task with low priority vs non-overdue with crit priority, today schedule.
	yesterday := makeTime(1)
	overdue := Todo{
		Priority:  PrioLow,
		Schedule:  ScheduleLater,
		DueDate:   &yesterday,
		CreatedAt: baseTime,
	}
	topTask := Todo{
		Priority:  PrioCrit,
		Schedule:  ScheduleToday,
		CreatedAt: baseTime,
	}

	scoreOverdue := UrgencyScore(overdue, baseTime, nil, w)
	scoreTop := UrgencyScore(topTask, baseTime, nil, w)

	// overdue = 100 + 5 + 10 = 115
	// topTask = 50 + 40 = 90
	if scoreOverdue <= scoreTop {
		t.Errorf("overdue task should always outrank non-overdue: overdue=%d, top=%d", scoreOverdue, scoreTop)
	}
}

func TestSortByUrgency_Order(t *testing.T) {
	w := defaultWeights()

	yesterday := makeTime(1)
	todos := []Todo{
		{ID: 1, Title: "low later", Priority: PrioLow, Schedule: ScheduleLater, CreatedAt: baseTime},
		{ID: 2, Title: "crit today", Priority: PrioCrit, Schedule: ScheduleToday, CreatedAt: baseTime},
		{ID: 3, Title: "overdue med", Priority: PrioMedium, Schedule: ScheduleLater, DueDate: &yesterday, CreatedAt: baseTime},
		{ID: 4, Title: "high soon", Priority: PrioHigh, Schedule: ScheduleSoon, CreatedAt: baseTime},
	}

	SortByUrgency(todos, baseTime, nil, w)

	// Expected order by score:
	// overdue med: 100 + 5 + 20 = 125
	// crit today:  50 + 40 = 90
	// high soon:   20 + 30 = 50
	// low later:   5 + 10 = 15
	if todos[0].ID != 3 {
		t.Errorf("expected first task to be overdue med (ID=3), got ID=%d", todos[0].ID)
	}
	if todos[1].ID != 2 {
		t.Errorf("expected second task to be crit today (ID=2), got ID=%d", todos[1].ID)
	}
	if todos[2].ID != 4 {
		t.Errorf("expected third task to be high soon (ID=4), got ID=%d", todos[2].ID)
	}
	if todos[3].ID != 1 {
		t.Errorf("expected fourth task to be low later (ID=1), got ID=%d", todos[3].ID)
	}
}

func TestSortByUrgency_TiebreakerOlderFirst(t *testing.T) {
	w := defaultWeights()

	// Both tasks created on the same calendar day (4 days ago) so they have
	// identical age bonuses and therefore identical urgency scores. Only the
	// wall-clock time within that day differs, which is what the tiebreaker
	// sorts on — the task created earlier in the day should rank first.
	sameDay := baseTime.AddDate(0, 0, -4)
	older := Todo{
		ID:        1,
		Title:     "older",
		Priority:  PrioMedium,
		Schedule:  ScheduleLater,
		CreatedAt: sameDay.Add(1 * time.Hour), // 01:00 that day
	}
	newer := Todo{
		ID:        2,
		Title:     "newer",
		Priority:  PrioMedium,
		Schedule:  ScheduleLater,
		CreatedAt: sameDay.Add(23 * time.Hour), // 23:00 that day
	}

	todos := []Todo{newer, older}
	SortByUrgency(todos, baseTime, nil, w)

	// Scores are identical (same priority, schedule, and integer age).
	// The tiebreaker puts the task with the earlier CreatedAt first.
	if todos[0].ID != 1 {
		t.Errorf("expected older task (ID=1) first via tiebreaker, got ID=%d", todos[0].ID)
	}
	if todos[1].ID != 2 {
		t.Errorf("expected newer task (ID=2) second via tiebreaker, got ID=%d", todos[1].ID)
	}
}

func TestUrgencyScore_NoAgeBonusForNewTask(t *testing.T) {
	w := defaultWeights()

	brand := Todo{
		Priority:  PrioMedium,
		Schedule:  ScheduleLater,
		CreatedAt: baseTime, // created right now
	}

	score := UrgencyScore(brand, baseTime, nil, w)
	expected := w.ScheduleLater + w.PriorityMed // no age, no overdue, no project
	if score != expected {
		t.Errorf("brand-new task: expected score %d, got %d", expected, score)
	}
}

func TestUrgencyScore_FutureDueDate_NoOverdueBonus(t *testing.T) {
	w := defaultWeights()

	tomorrow := baseTime.AddDate(0, 0, 1)
	task := Todo{
		Priority:  PrioMedium,
		Schedule:  ScheduleLater,
		DueDate:   &tomorrow,
		CreatedAt: baseTime,
	}

	score := UrgencyScore(task, baseTime, nil, w)
	expected := w.ScheduleLater + w.PriorityMed
	if score != expected {
		t.Errorf("future due date should not trigger overdue bonus: expected %d, got %d", expected, score)
	}
}
