package cmd

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rnwolfe/mine/internal/config"
	"github.com/rnwolfe/mine/internal/proj"
	"github.com/rnwolfe/mine/internal/store"
	"github.com/rnwolfe/mine/internal/todo"
)

// todoTestEnv sets up isolated XDG dirs and returns the test temp root.
func todoTestEnv(t *testing.T) string {
	t.Helper()
	configTestEnv(t)
	return t.TempDir()
}

// registerProject registers a directory as a project in the store and returns its path.
func registerProject(t *testing.T, name string) string {
	t.Helper()
	// Create the directory so proj.Add succeeds (needs a real dir).
	dir := t.TempDir()

	// Rename the temp dir to the desired project name by creating a subdir.
	projDir := filepath.Join(dir, name)
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}

	db, err := store.Open()
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer db.Close()

	ps := proj.NewStore(db.Conn())
	if _, err := ps.Add(projDir); err != nil {
		t.Fatalf("register project: %v", err)
	}
	return projDir
}

func TestRunTodoAdd_GlobalTask(t *testing.T) {
	todoTestEnv(t)
	// Reset global state
	todoProjectName = ""
	todoPriority = "med"
	todoDue = ""
	todoTags = ""

	// Ensure cwd is not a registered project (use a fresh temp dir).
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	err := runTodoAdd(nil, []string{"global task"})
	if err != nil {
		t.Fatalf("runTodoAdd: %v", err)
	}

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ts := todo.NewStore(db.Conn())
	todos, err := ts.List(todo.ListOptions{AllProjects: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
	if todos[0].ProjectPath != nil {
		t.Fatalf("expected nil project path for global task, got %q", *todos[0].ProjectPath)
	}
}

func TestRunTodoAdd_InsideProject_AutoDetects(t *testing.T) {
	todoTestEnv(t)
	todoProjectName = ""
	todoPriority = "med"
	todoDue = ""
	todoTags = ""

	projDir := registerProject(t, "myproject")

	// Change cwd to the project directory.
	origDir, _ := os.Getwd()
	os.Chdir(projDir)
	defer os.Chdir(origDir)

	err := runTodoAdd(nil, []string{"project task"})
	if err != nil {
		t.Fatalf("runTodoAdd: %v", err)
	}

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ts := todo.NewStore(db.Conn())
	todos, err := ts.List(todo.ListOptions{AllProjects: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
	if todos[0].ProjectPath == nil {
		t.Fatal("expected project path to be set for task added inside project")
	}
	if *todos[0].ProjectPath != projDir {
		t.Fatalf("expected project path %q, got %q", projDir, *todos[0].ProjectPath)
	}
}

func TestRunTodoAdd_ExplicitProject(t *testing.T) {
	todoTestEnv(t)
	todoPriority = "med"
	todoDue = ""
	todoTags = ""

	projDir := registerProject(t, "explicitproj")

	// Cwd is NOT the project dir.
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	todoProjectName = "explicitproj"
	defer func() { todoProjectName = "" }()

	err := runTodoAdd(nil, []string{"explicit task"})
	if err != nil {
		t.Fatalf("runTodoAdd: %v", err)
	}

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ts := todo.NewStore(db.Conn())
	todos, err := ts.List(todo.ListOptions{AllProjects: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 1 || todos[0].ProjectPath == nil || *todos[0].ProjectPath != projDir {
		t.Fatalf("expected task assigned to project %q", projDir)
	}
}

func TestRunTodoAdd_UnknownProject_Error(t *testing.T) {
	todoTestEnv(t)
	todoPriority = "med"
	todoDue = ""
	todoTags = ""

	todoProjectName = "nonexistent"
	defer func() { todoProjectName = "" }()

	err := runTodoAdd(nil, []string{"task"})
	if err == nil {
		t.Fatal("expected error for unknown project")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected project name in error, got: %v", err)
	}
}

func TestRunTodoList_ShowDone(t *testing.T) {
	todoTestEnv(t)
	todoProjectName = ""
	todoShowAll = false

	// Use cwd outside any project.
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	id, _ := ts.Add("done task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	ts.Complete(id) //nolint:errcheck
	ts.Add("open task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	db.Close()

	// --done=false: only open tasks
	todoShowDone = false
	out := captureStdout(t, func() {
		runTodoList(nil, nil)
	})
	if strings.Contains(out, "done task") {
		t.Error("expected done task to be hidden without --done flag")
	}
	if !strings.Contains(out, "open task") {
		t.Error("expected open task in output")
	}

	// --done: both tasks
	todoShowDone = true
	defer func() { todoShowDone = false }()
	out = captureStdout(t, func() {
		runTodoList(nil, nil)
	})
	if !strings.Contains(out, "done task") {
		t.Error("expected done task with --done flag")
	}
}

func TestRunTodoList_AllProjects(t *testing.T) {
	todoTestEnv(t)
	todoShowDone = false

	projDir := registerProject(t, "projlist")

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	ts.Add("global task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	ts.Add("project task", "", todo.PrioMedium, nil, nil, &projDir, todo.ScheduleLater, todo.RecurrenceNone)
	db.Close()

	// Cwd outside any project.
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Without --all: only global task (cwd not in any project)
	todoProjectName = ""
	todoShowAll = false
	out := captureStdout(t, func() {
		runTodoList(nil, nil)
	})
	if strings.Contains(out, "project task") {
		t.Error("expected project task to be hidden without --all flag")
	}
	if !strings.Contains(out, "global task") {
		t.Error("expected global task in output")
	}

	// With --all: both tasks visible
	todoShowAll = true
	defer func() { todoShowAll = false }()
	out = captureStdout(t, func() {
		runTodoList(nil, nil)
	})
	if !strings.Contains(out, "global task") {
		t.Error("expected global task in --all output")
	}
	if !strings.Contains(out, "project task") {
		t.Error("expected project task in --all output")
	}
}

func TestRunTodoList_ProjectFlag(t *testing.T) {
	todoTestEnv(t)
	todoShowDone = false
	todoShowAll = false

	projDir := registerProject(t, "flagproj")

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	ts.Add("global task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	ts.Add("flagproj task", "", todo.PrioMedium, nil, nil, &projDir, todo.ScheduleLater, todo.RecurrenceNone)
	ts.Add("other task", "", todo.PrioMedium, nil, nil, func() *string { s := "/other/proj"; return &s }(), todo.ScheduleLater, todo.RecurrenceNone)
	db.Close()

	// Cwd outside any project.
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	todoProjectName = "flagproj"
	defer func() { todoProjectName = "" }()

	out := captureStdout(t, func() {
		runTodoList(nil, nil)
	})

	if !strings.Contains(out, "flagproj task") {
		t.Error("expected flagproj task in --project output")
	}
	if !strings.Contains(out, "global task") {
		t.Error("expected global task with --project (global tasks always included)")
	}
	if strings.Contains(out, "other task") {
		t.Error("expected other task to be excluded from --project flagproj output")
	}
}

func TestRunTodoList_CwdResolution(t *testing.T) {
	todoTestEnv(t)
	todoShowDone = false
	todoShowAll = false
	todoProjectName = ""

	projDir := registerProject(t, "cwdproj")

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	ts.Add("global task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	ts.Add("cwd project task", "", todo.PrioMedium, nil, nil, &projDir, todo.ScheduleLater, todo.RecurrenceNone)
	ts.Add("other project task", "", todo.PrioMedium, nil, nil, func() *string { s := "/other/proj"; return &s }(), todo.ScheduleLater, todo.RecurrenceNone)
	db.Close()

	// Change cwd into the registered project directory.
	origDir, _ := os.Getwd()
	os.Chdir(projDir)
	defer os.Chdir(origDir)

	out := captureStdout(t, func() {
		runTodoList(nil, nil)
	})

	if !strings.Contains(out, "cwd project task") {
		t.Error("expected project task when cwd is inside registered project")
	}
	if !strings.Contains(out, "global task") {
		t.Error("expected global task (always included with project context)")
	}
	if strings.Contains(out, "other project task") {
		t.Error("expected other project task to be excluded via cwd resolution")
	}
}

func TestResolveTodoProject_ExplicitName(t *testing.T) {
	todoTestEnv(t)

	projDir := registerProject(t, "resolver")

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ps := proj.NewStore(db.Conn())
	path, err := resolveTodoProject(ps, "resolver")
	if err != nil {
		t.Fatalf("resolveTodoProject: %v", err)
	}
	if path == nil || *path != projDir {
		t.Fatalf("expected path %q, got %v", projDir, path)
	}
}

func TestResolveTodoProject_UnknownName_Error(t *testing.T) {
	todoTestEnv(t)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ps := proj.NewStore(db.Conn())
	_, err = resolveTodoProject(ps, "doesnotexist")
	if err == nil {
		t.Fatal("expected error for unknown project name")
	}
}

func TestResolveTodoProject_CwdInProject(t *testing.T) {
	todoTestEnv(t)

	projDir := registerProject(t, "cwdresolver")

	origDir, _ := os.Getwd()
	os.Chdir(projDir)
	defer os.Chdir(origDir)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ps := proj.NewStore(db.Conn())
	path, err := resolveTodoProject(ps, "")
	if err != nil {
		t.Fatalf("resolveTodoProject: %v", err)
	}
	if path == nil {
		t.Fatal("expected non-nil path when cwd is inside project")
	}
	if *path != projDir {
		t.Fatalf("expected %q, got %q", projDir, *path)
	}
}

func TestResolveTodoProject_CwdOutsideProject(t *testing.T) {
	todoTestEnv(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ps := proj.NewStore(db.Conn())
	path, err := resolveTodoProject(ps, "")
	if err != nil {
		t.Fatalf("resolveTodoProject: %v", err)
	}
	if path != nil {
		t.Fatalf("expected nil path when cwd is outside any project, got %q", *path)
	}
}

// --- Schedule integration tests ---

func TestRunTodoAdd_WithScheduleFlag(t *testing.T) {
	todoTestEnv(t)
	todoPriority = "med"
	todoDue = ""
	todoTags = ""
	todoProjectName = ""

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	todoScheduleFlag = "today"
	defer func() { todoScheduleFlag = "later" }()

	err := runTodoAdd(nil, []string{"urgent task"})
	if err != nil {
		t.Fatalf("runTodoAdd: %v", err)
	}

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ts := todo.NewStore(db.Conn())
	todos, err := ts.List(todo.ListOptions{AllProjects: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
	if todos[0].Schedule != todo.ScheduleToday {
		t.Fatalf("expected schedule %q, got %q", todo.ScheduleToday, todos[0].Schedule)
	}
}

func TestRunTodoAdd_DefaultScheduleIsLater(t *testing.T) {
	todoTestEnv(t)
	todoPriority = "med"
	todoDue = ""
	todoTags = ""
	todoProjectName = ""
	todoScheduleFlag = "later"

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	err := runTodoAdd(nil, []string{"default schedule task"})
	if err != nil {
		t.Fatalf("runTodoAdd: %v", err)
	}

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ts := todo.NewStore(db.Conn())
	todos, err := ts.List(todo.ListOptions{AllProjects: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
	if todos[0].Schedule != todo.ScheduleLater {
		t.Fatalf("expected schedule %q, got %q", todo.ScheduleLater, todos[0].Schedule)
	}
}

func TestRunTodoAdd_InvalidSchedule_Error(t *testing.T) {
	todoTestEnv(t)
	todoPriority = "med"
	todoDue = ""
	todoTags = ""
	todoProjectName = ""

	todoScheduleFlag = "invalid"
	defer func() { todoScheduleFlag = "later" }()

	err := runTodoAdd(nil, []string{"task"})
	if err == nil {
		t.Fatal("expected error for invalid schedule")
	}
	if !strings.Contains(err.Error(), "invalid schedule") {
		t.Errorf("expected 'invalid schedule' in error, got: %v", err)
	}
}

func TestRunTodoSchedule_SetsSchedule(t *testing.T) {
	todoTestEnv(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	id, _ := ts.Add("test task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	db.Close()

	err = runTodoSchedule(nil, []string{strconv.Itoa(id), "today"})
	if err != nil {
		t.Fatalf("runTodoSchedule: %v", err)
	}

	db, err = store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ts = todo.NewStore(db.Conn())
	got, err := ts.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Schedule != todo.ScheduleToday {
		t.Fatalf("expected schedule %q, got %q", todo.ScheduleToday, got.Schedule)
	}
}

func TestRunTodoSchedule_ShortAlias(t *testing.T) {
	todoTestEnv(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	id, _ := ts.Add("alias task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	db.Close()

	// Use short alias "sd" for someday
	err = runTodoSchedule(nil, []string{strconv.Itoa(id), "sd"})
	if err != nil {
		t.Fatalf("runTodoSchedule with alias: %v", err)
	}

	db, err = store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ts = todo.NewStore(db.Conn())
	got, err := ts.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Schedule != todo.ScheduleSomeday {
		t.Fatalf("expected schedule %q, got %q", todo.ScheduleSomeday, got.Schedule)
	}
}

func TestRunTodoSchedule_InvalidSchedule_Error(t *testing.T) {
	todoTestEnv(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	id, _ := ts.Add("task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	db.Close()

	err = runTodoSchedule(nil, []string{strconv.Itoa(id), "invalid"})
	if err == nil {
		t.Fatal("expected error for invalid schedule")
	}
	if !strings.Contains(err.Error(), "invalid schedule") {
		t.Errorf("expected 'invalid schedule' in error, got: %v", err)
	}
}

func TestRunTodoSchedule_InvalidID_Error(t *testing.T) {
	todoTestEnv(t)

	err := runTodoSchedule(nil, []string{"notanumber", "today"})
	if err == nil {
		t.Fatal("expected error for non-numeric ID")
	}
	if !strings.Contains(err.Error(), "not a valid todo ID") {
		t.Errorf("expected 'not a valid todo ID' in error, got: %v", err)
	}
}

func TestRunTodoList_ExcludesSomedayByDefault(t *testing.T) {
	todoTestEnv(t)
	todoShowDone = false
	todoShowAll = true
	todoProjectName = ""
	todoIncludeSomeday = false

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	ts.Add("later task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	ts.Add("someday task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleSomeday, todo.RecurrenceNone)
	db.Close()

	out := captureStdout(t, func() {
		runTodoList(nil, nil)
	})

	if strings.Contains(out, "someday task") {
		t.Error("expected someday task to be hidden in default list output")
	}
	if !strings.Contains(out, "later task") {
		t.Error("expected later task in default list output")
	}
}

func TestRunTodoList_SomedayFlagIncludesSomeday(t *testing.T) {
	todoTestEnv(t)
	todoShowDone = false
	todoShowAll = true
	todoProjectName = ""
	todoIncludeSomeday = true
	defer func() { todoIncludeSomeday = false }()

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	ts.Add("later task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	ts.Add("someday task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleSomeday, todo.RecurrenceNone)
	db.Close()

	out := captureStdout(t, func() {
		runTodoList(nil, nil)
	})

	if !strings.Contains(out, "someday task") {
		t.Error("expected someday task in output with --someday flag")
	}
	if !strings.Contains(out, "later task") {
		t.Error("expected later task in output with --someday flag")
	}
}

// --- mine todo next integration tests ---

func TestRunTodoNext_SingleHighestUrgency(t *testing.T) {
	todoTestEnv(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	ts.Add("low later task", "", todo.PrioLow, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	ts.Add("crit today task", "", todo.PrioCrit, nil, nil, nil, todo.ScheduleToday, todo.RecurrenceNone)
	ts.Add("med soon task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleSoon, todo.RecurrenceNone)
	db.Close()

	out := captureStdout(t, func() {
		runTodoNext(nil, nil) // default n=1
	})

	// crit today scores highest (50+40=90), should appear
	if !strings.Contains(out, "crit today task") {
		t.Errorf("expected highest-urgency task in output, got:\n%s", out)
	}
	// low later should NOT appear (only top 1)
	if strings.Contains(out, "low later task") {
		t.Errorf("expected only top 1 task, but low later task appeared:\n%s", out)
	}
	if strings.Contains(out, "med soon task") {
		t.Errorf("expected only top 1 task, but med soon task appeared:\n%s", out)
	}
}

func TestRunTodoNext_TopN(t *testing.T) {
	todoTestEnv(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	ts.Add("task A", "", todo.PrioLow, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	ts.Add("task B", "", todo.PrioCrit, nil, nil, nil, todo.ScheduleToday, todo.RecurrenceNone)
	ts.Add("task C", "", todo.PrioHigh, nil, nil, nil, todo.ScheduleSoon, todo.RecurrenceNone)
	db.Close()

	out := captureStdout(t, func() {
		runTodoNext(nil, []string{"3"})
	})

	if !strings.Contains(out, "task A") {
		t.Error("expected task A in top-3 output")
	}
	if !strings.Contains(out, "task B") {
		t.Error("expected task B in top-3 output")
	}
	if !strings.Contains(out, "task C") {
		t.Error("expected task C in top-3 output")
	}
}

func TestRunTodoNext_AllClear_NoTasks(t *testing.T) {
	todoTestEnv(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	out := captureStdout(t, func() {
		runTodoNext(nil, nil)
	})

	if !strings.Contains(out, "All clear") {
		t.Errorf("expected 'All clear' message when no tasks, got:\n%s", out)
	}
}

func TestRunTodoNext_InvalidCount_Error(t *testing.T) {
	todoTestEnv(t)

	err := runTodoNext(nil, []string{"0"})
	if err == nil {
		t.Fatal("expected error for count=0")
	}
	if !strings.Contains(err.Error(), "is not a valid count") {
		t.Errorf("expected 'is not a valid count' in error, got: %v", err)
	}

	err = runTodoNext(nil, []string{"-1"})
	if err == nil {
		t.Fatal("expected error for negative count")
	}

	err = runTodoNext(nil, []string{"notanumber"})
	if err == nil {
		t.Fatal("expected error for non-numeric count")
	}
}

func TestRunTodoNext_ExcludesSomeday(t *testing.T) {
	todoTestEnv(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	ts.Add("someday idea", "", todo.PrioCrit, nil, nil, nil, todo.ScheduleSomeday, todo.RecurrenceNone)
	ts.Add("open task", "", todo.PrioLow, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	db.Close()

	out := captureStdout(t, func() {
		runTodoNext(nil, []string{"5"})
	})

	if strings.Contains(out, "someday idea") {
		t.Errorf("someday tasks should be excluded from 'next' output:\n%s", out)
	}
	if !strings.Contains(out, "open task") {
		t.Errorf("expected open task in output:\n%s", out)
	}
}

func TestRunTodoNext_OverdueRanksFirst(t *testing.T) {
	todoTestEnv(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	past := time.Now().AddDate(0, 0, -1)
	ts.Add("overdue low", "", todo.PrioLow, nil, &past, nil, todo.ScheduleLater, todo.RecurrenceNone)
	ts.Add("crit today no due", "", todo.PrioCrit, nil, nil, nil, todo.ScheduleToday, todo.RecurrenceNone)
	db.Close()

	out := captureStdout(t, func() {
		runTodoNext(nil, nil) // top 1
	})

	if !strings.Contains(out, "overdue low") {
		t.Errorf("overdue task should rank first, got:\n%s", out)
	}
	if strings.Contains(out, "crit today no due") {
		t.Errorf("non-overdue task should not appear when overdue exists and n=1:\n%s", out)
	}
}

func TestRunTodoNext_DetailCardFields(t *testing.T) {
	todoTestEnv(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	past := time.Now().AddDate(0, 0, -2)
	ts.Add("tagged task", "", todo.PrioHigh, []string{"docs", "v2"}, &past, nil, todo.ScheduleSoon, todo.RecurrenceNone)
	db.Close()

	out := captureStdout(t, func() {
		runTodoNext(nil, nil)
	})

	if !strings.Contains(out, "tagged task") {
		t.Errorf("expected title in card output:\n%s", out)
	}
	if !strings.Contains(out, "docs") || !strings.Contains(out, "v2") {
		t.Errorf("expected tags in card output:\n%s", out)
	}
	// overdue annotation
	if !strings.Contains(out, "overdue") {
		t.Errorf("expected overdue annotation in card output:\n%s", out)
	}
}

func TestRunTodoNext_NewTodo_ShowsNoAge(t *testing.T) {
	// Regression test for: new todo is 106751 days old (timestamp parse failure).
	// A todo created in the same session should NOT show any "day(s) old" line.
	todoTestEnv(t)

	t.Chdir(t.TempDir())

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	ts.Add("brand new task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleToday, todo.RecurrenceNone)
	db.Close()

	out := captureStdout(t, func() {
		runTodoNext(nil, nil)
	})

	// A freshly created todo has age 0 — the card should NOT print "day(s) old".
	if strings.Contains(out, "day(s) old") {
		t.Errorf("new todo should show no age line, got:\n%s", out)
	}
	if !strings.Contains(out, "brand new task") {
		t.Errorf("expected todo title in output:\n%s", out)
	}
}

// --- urgencyWeightsFromConfig tests ---

func TestUrgencyWeightsFromConfig_AllNilUsesDefaults(t *testing.T) {
	cfg := &config.Config{} // all Urgency fields nil
	got := urgencyWeightsFromConfig(cfg)
	defaults := todo.DefaultUrgencyWeights()

	if got != defaults {
		t.Errorf("all-nil config should return defaults:\n  got:  %+v\n  want: %+v", got, defaults)
	}
}

func TestUrgencyWeightsFromConfig_SingleOverride(t *testing.T) {
	overrideVal := 999
	cfg := &config.Config{
		Todo: config.TodoConfig{
			Urgency: config.UrgencyWeightsConfig{
				Overdue: &overrideVal,
			},
		},
	}
	got := urgencyWeightsFromConfig(cfg)
	defaults := todo.DefaultUrgencyWeights()

	// Overridden field reflects the new value.
	if got.Overdue != overrideVal {
		t.Errorf("Overdue: expected %d, got %d", overrideVal, got.Overdue)
	}
	// All other fields remain at their defaults.
	if got.ScheduleToday != defaults.ScheduleToday {
		t.Errorf("ScheduleToday: expected %d, got %d", defaults.ScheduleToday, got.ScheduleToday)
	}
	if got.ScheduleSoon != defaults.ScheduleSoon {
		t.Errorf("ScheduleSoon: expected %d, got %d", defaults.ScheduleSoon, got.ScheduleSoon)
	}
	if got.ScheduleLater != defaults.ScheduleLater {
		t.Errorf("ScheduleLater: expected %d, got %d", defaults.ScheduleLater, got.ScheduleLater)
	}
	if got.PriorityCrit != defaults.PriorityCrit {
		t.Errorf("PriorityCrit: expected %d, got %d", defaults.PriorityCrit, got.PriorityCrit)
	}
	if got.PriorityHigh != defaults.PriorityHigh {
		t.Errorf("PriorityHigh: expected %d, got %d", defaults.PriorityHigh, got.PriorityHigh)
	}
	if got.PriorityMed != defaults.PriorityMed {
		t.Errorf("PriorityMed: expected %d, got %d", defaults.PriorityMed, got.PriorityMed)
	}
	if got.PriorityLow != defaults.PriorityLow {
		t.Errorf("PriorityLow: expected %d, got %d", defaults.PriorityLow, got.PriorityLow)
	}
	if got.AgeCap != defaults.AgeCap {
		t.Errorf("AgeCap: expected %d, got %d", defaults.AgeCap, got.AgeCap)
	}
	if got.ProjectBoost != defaults.ProjectBoost {
		t.Errorf("ProjectBoost: expected %d, got %d", defaults.ProjectBoost, got.ProjectBoost)
	}
}

// --- mine todo note integration tests ---

func TestRunTodoNote_AppendsNote(t *testing.T) {
	todoTestEnv(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	id, _ := ts.Add("annotated task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	db.Close()

	err = runTodoNote(nil, []string{strconv.Itoa(id), "first note here"})
	if err != nil {
		t.Fatalf("runTodoNote: %v", err)
	}

	db, err = store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ts = todo.NewStore(db.Conn())
	got, err := ts.GetWithNotes(id)
	if err != nil {
		t.Fatalf("GetWithNotes: %v", err)
	}
	if len(got.Notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(got.Notes))
	}
	if got.Notes[0].Body != "first note here" {
		t.Fatalf("expected note body %q, got %q", "first note here", got.Notes[0].Body)
	}
}

func TestRunTodoNote_NotFound_Error(t *testing.T) {
	todoTestEnv(t)

	err := runTodoNote(nil, []string{"9999", "some text"})
	if err == nil {
		t.Fatal("expected error for non-existent todo ID")
	}
	if !strings.Contains(err.Error(), "#9999 not found") {
		t.Errorf("expected '#9999 not found' in error, got: %v", err)
	}
}

func TestRunTodoNote_InvalidID_Error(t *testing.T) {
	todoTestEnv(t)

	err := runTodoNote(nil, []string{"notanumber", "text"})
	if err == nil {
		t.Fatal("expected error for non-numeric ID")
	}
	if !strings.Contains(err.Error(), "not a valid todo ID") {
		t.Errorf("expected 'not a valid todo ID' in error, got: %v", err)
	}
}

// --- mine todo show integration tests ---

func TestRunTodoShow_DisplaysDetail(t *testing.T) {
	todoTestEnv(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	id, _ := ts.Add("show task", "", todo.PrioHigh, []string{"docs"}, nil, nil, todo.ScheduleSoon, todo.RecurrenceNone)
	ts.AddNote(id, "a timestamped note")
	db.Close()

	out := captureStdout(t, func() {
		runTodoShow(nil, []string{strconv.Itoa(id)})
	})

	if !strings.Contains(out, "show task") {
		t.Errorf("expected title in output:\n%s", out)
	}
	if !strings.Contains(out, "docs") {
		t.Errorf("expected tags in output:\n%s", out)
	}
	if !strings.Contains(out, "a timestamped note") {
		t.Errorf("expected note text in output:\n%s", out)
	}
	if !strings.Contains(out, "Notes:") {
		t.Errorf("expected 'Notes:' section in output:\n%s", out)
	}
}

func TestRunTodoShow_NoNotes_OmitsNotesSection(t *testing.T) {
	todoTestEnv(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	id, _ := ts.Add("plain task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	db.Close()

	out := captureStdout(t, func() {
		runTodoShow(nil, []string{strconv.Itoa(id)})
	})

	if !strings.Contains(out, "plain task") {
		t.Errorf("expected title in output:\n%s", out)
	}
	if strings.Contains(out, "Notes:") {
		t.Errorf("expected 'Notes:' section to be omitted when no notes:\n%s", out)
	}
}

func TestRunTodoShow_WithBody_DisplaysBody(t *testing.T) {
	todoTestEnv(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	id, _ := ts.Add("body task", "initial context text", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	db.Close()

	out := captureStdout(t, func() {
		runTodoShow(nil, []string{strconv.Itoa(id)})
	})

	if !strings.Contains(out, "initial context text") {
		t.Errorf("expected body text in output:\n%s", out)
	}
	if !strings.Contains(out, "Body:") {
		t.Errorf("expected 'Body:' section in output:\n%s", out)
	}
}

func TestRunTodoShow_NotFound_Error(t *testing.T) {
	todoTestEnv(t)

	err := runTodoShow(nil, []string{"9999"})
	if err == nil {
		t.Fatal("expected error for non-existent todo")
	}
	if !strings.Contains(err.Error(), "#9999 not found") {
		t.Errorf("expected '#9999 not found' in error, got: %v", err)
	}
}

func TestRunTodoShow_InvalidID_Error(t *testing.T) {
	todoTestEnv(t)

	err := runTodoShow(nil, []string{"notanumber"})
	if err == nil {
		t.Fatal("expected error for non-numeric ID")
	}
	if !strings.Contains(err.Error(), "not a valid todo ID") {
		t.Errorf("expected 'not a valid todo ID' in error, got: %v", err)
	}
}

// --- mine todo add --note flag integration test ---

func TestRunTodoAdd_WithNoteFlag_SetsBody(t *testing.T) {
	todoTestEnv(t)
	todoPriority = "med"
	todoDue = ""
	todoTags = ""
	todoProjectName = ""
	todoScheduleFlag = "later"
	todoNoteFlag = "context for this task"
	defer func() { todoNoteFlag = "" }()

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	err := runTodoAdd(nil, []string{"task with note"})
	if err != nil {
		t.Fatalf("runTodoAdd: %v", err)
	}

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ts := todo.NewStore(db.Conn())
	todos, err := ts.List(todo.ListOptions{AllProjects: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
	if todos[0].Body != "context for this task" {
		t.Fatalf("expected body %q, got %q", "context for this task", todos[0].Body)
	}
}

// --- mine todo stats integration tests ---

// statsInsertCompleted inserts a completed todo with controlled timestamps using raw SQL.
func statsInsertCompleted(t *testing.T, title string, createdAt, completedAt time.Time) {
	t.Helper()
	db, err := store.Open()
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer db.Close()

	_, err = db.Conn().Exec(
		`INSERT INTO todos (title, priority, done, created_at, completed_at, updated_at)
		 VALUES (?, 2, 1, ?, ?, ?)`,
		title,
		createdAt.UTC().Format("2006-01-02 15:04:05"),
		completedAt.UTC().Format("2006-01-02 15:04:05"),
		completedAt.UTC().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		t.Fatalf("statsInsertCompleted: %v", err)
	}
}

func TestRunTodoStats_NoCompletions(t *testing.T) {
	todoTestEnv(t)
	todoStatsProjectFlag = ""

	out := captureStdout(t, func() {
		runTodoStats(nil, nil)
	})

	if !strings.Contains(out, "No completions yet") {
		t.Errorf("expected encouraging message when no completions, got:\n%s", out)
	}
}

func TestRunTodoStats_WithCompletions(t *testing.T) {
	todoTestEnv(t)
	todoStatsProjectFlag = ""

	now := time.Now()

	// Insert a completion today.
	statsInsertCompleted(t, "completed today", now.AddDate(0, 0, -1), now)

	out := captureStdout(t, func() {
		runTodoStats(nil, nil)
	})

	if !strings.Contains(out, "Task Stats") {
		t.Errorf("expected 'Task Stats' header in output:\n%s", out)
	}
	if !strings.Contains(out, "Streak") {
		t.Errorf("expected 'Streak' in output:\n%s", out)
	}
	if !strings.Contains(out, "This week") {
		t.Errorf("expected 'This week' in output:\n%s", out)
	}
	if !strings.Contains(out, "This month") {
		t.Errorf("expected 'This month' in output:\n%s", out)
	}
}

func TestRunTodoStats_ByProjectBreakdown(t *testing.T) {
	todoTestEnv(t)
	todoStatsProjectFlag = ""

	projDir := registerProject(t, "statsproj")

	// Add todos to the project and as global.
	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	ts.Add("global open", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	ts.Add("proj open", "", todo.PrioMedium, nil, nil, &projDir, todo.ScheduleLater, todo.RecurrenceNone)
	db.Close()

	now := time.Now()
	statsInsertCompleted(t, "completed global", now.AddDate(0, 0, -1), now)

	out := captureStdout(t, func() {
		runTodoStats(nil, nil)
	})

	if !strings.Contains(out, "By project") {
		t.Errorf("expected 'By project' section in output:\n%s", out)
	}
	if !strings.Contains(out, "(global)") {
		t.Errorf("expected '(global)' in project breakdown:\n%s", out)
	}
}

func TestRunTodoStats_ProjectFlag_Scoped(t *testing.T) {
	todoTestEnv(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	projDir := registerProject(t, "scopedstats")

	todoStatsProjectFlag = "scopedstats"
	defer func() { todoStatsProjectFlag = "" }()

	// Add one global completion and one project completion.
	now := time.Now()
	statsInsertCompleted(t, "global done", now.AddDate(0, 0, -1), now)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	_, execErr := db.Conn().Exec(
		`INSERT INTO todos (title, priority, done, project_path, created_at, completed_at, updated_at)
		 VALUES (?, 2, 1, ?, ?, ?, ?)`,
		"proj done",
		projDir,
		now.AddDate(0, 0, -1).UTC().Format("2006-01-02 15:04:05"),
		now.UTC().Format("2006-01-02 15:04:05"),
		now.UTC().Format("2006-01-02 15:04:05"),
	)
	db.Close()
	if execErr != nil {
		t.Fatalf("inserting project todo: %v", execErr)
	}

	out := captureStdout(t, func() {
		runTodoStats(nil, nil)
	})

	if !strings.Contains(out, "Task Stats") {
		t.Errorf("expected 'Task Stats' in scoped output:\n%s", out)
	}
	// When scoped, "By project" breakdown should be omitted.
	if strings.Contains(out, "By project") {
		t.Errorf("expected 'By project' to be omitted when --project is set:\n%s", out)
	}
}

func TestRunTodoStats_ProjectFlag_NotFound(t *testing.T) {
	todoTestEnv(t)

	todoStatsProjectFlag = "doesnotexist"
	defer func() { todoStatsProjectFlag = "" }()

	err := runTodoStats(nil, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
	if !strings.Contains(err.Error(), "doesnotexist") {
		t.Errorf("expected project name in error, got: %v", err)
	}
}

// --- mine todo add --every integration tests ---

func TestRunTodoAdd_WithEveryFlag_SetsRecurrence(t *testing.T) {
	todoTestEnv(t)
	todoPriority = "med"
	todoDue = ""
	todoTags = ""
	todoProjectName = ""
	todoScheduleFlag = "later"
	todoNoteFlag = ""
	todoEveryFlag = "week"
	defer func() { todoEveryFlag = "" }()

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	err := runTodoAdd(nil, []string{"Review PRs"})
	if err != nil {
		t.Fatalf("runTodoAdd: %v", err)
	}

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ts := todo.NewStore(db.Conn())
	todos, err := ts.List(todo.ListOptions{AllProjects: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
	if todos[0].Recurrence != todo.RecurrenceWeekly {
		t.Errorf("expected recurrence %q, got %q", todo.RecurrenceWeekly, todos[0].Recurrence)
	}
}

func TestRunTodoAdd_WithEveryShortAlias(t *testing.T) {
	todoTestEnv(t)
	todoPriority = "med"
	todoDue = ""
	todoTags = ""
	todoProjectName = ""
	todoScheduleFlag = "later"
	todoNoteFlag = ""

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	aliases := map[string]string{
		"d":  todo.RecurrenceDaily,
		"wd": todo.RecurrenceWeekday,
		"w":  todo.RecurrenceWeekly,
		"m":  todo.RecurrenceMonthly,
	}
	for alias, want := range aliases {
		t.Run(alias, func(t *testing.T) {
			todoEveryFlag = alias
			defer func() { todoEveryFlag = "" }()

			db, err := store.Open()
			if err != nil {
				t.Fatal(err)
			}
			// Clear todos
			db.Conn().Exec(`DELETE FROM todos`)
			db.Close()

			if err := runTodoAdd(nil, []string{"task " + alias}); err != nil {
				t.Fatalf("runTodoAdd: %v", err)
			}

			db, err = store.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			ts := todo.NewStore(db.Conn())
			todos, _ := ts.List(todo.ListOptions{AllProjects: true})
			if len(todos) != 1 || todos[0].Recurrence != want {
				t.Errorf("alias %q: expected recurrence %q, got %q", alias, want, todos[0].Recurrence)
			}
		})
	}
}

func TestRunTodoAdd_InvalidEveryFlag_Error(t *testing.T) {
	todoTestEnv(t)
	todoPriority = "med"
	todoDue = ""
	todoTags = ""
	todoProjectName = ""
	todoScheduleFlag = "later"
	todoNoteFlag = ""
	todoEveryFlag = "biweekly"
	defer func() { todoEveryFlag = "" }()

	err := runTodoAdd(nil, []string{"task"})
	if err == nil {
		t.Fatal("expected error for invalid --every value")
	}
	if !strings.Contains(err.Error(), "invalid recurrence") {
		t.Errorf("expected 'invalid recurrence' in error, got: %v", err)
	}
}

// --- mine todo done (recurring spawn) integration tests ---

func TestRunTodoDone_RecurringTask_SpawnsNext(t *testing.T) {
	todoTestEnv(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	due := time.Now().AddDate(0, 0, 0) // today
	id, _ := ts.Add("weekly report", "", todo.PrioHigh, nil, &due, nil, todo.ScheduleLater, todo.RecurrenceWeekly)
	db.Close()

	out := captureStdout(t, func() {
		runTodoDone(nil, []string{strconv.Itoa(id)})
	})

	if !strings.Contains(out, "Done!") {
		t.Errorf("expected 'Done!' in output:\n%s", out)
	}
	if !strings.Contains(out, "Next occurrence spawned") {
		t.Errorf("expected 'Next occurrence spawned' in output:\n%s", out)
	}

	// Verify the spawned task exists
	db, err = store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ts = todo.NewStore(db.Conn())
	todos, _ := ts.List(todo.ListOptions{AllProjects: true})
	// id is done, spawned is open
	if len(todos) != 1 {
		t.Fatalf("expected 1 open task (spawned), got %d", len(todos))
	}
	if todos[0].Recurrence != todo.RecurrenceWeekly {
		t.Errorf("spawned task has wrong recurrence: %q", todos[0].Recurrence)
	}
}

func TestRunTodoDone_NonRecurringTask_NoSpawn(t *testing.T) {
	todoTestEnv(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	id, _ := ts.Add("regular task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	db.Close()

	out := captureStdout(t, func() {
		runTodoDone(nil, []string{strconv.Itoa(id)})
	})

	if strings.Contains(out, "Next occurrence spawned") {
		t.Errorf("expected no spawn message for non-recurring task:\n%s", out)
	}

	db, err = store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ts = todo.NewStore(db.Conn())
	open, _, _, _ := ts.Count(nil)
	if open != 0 {
		t.Errorf("expected 0 open tasks after completing non-recurring, got %d", open)
	}
}

// --- mine todo recurring integration tests ---

func TestRunTodoRecurring_ListsRecurringTasks(t *testing.T) {
	todoTestEnv(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	ts.Add("daily standup", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceDaily)
	ts.Add("weekly report", "", todo.PrioHigh, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceWeekly)
	ts.Add("plain task", "", todo.PrioLow, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	db.Close()

	out := captureStdout(t, func() {
		runTodoRecurring(nil, nil)
	})

	if !strings.Contains(out, "daily standup") {
		t.Errorf("expected 'daily standup' in recurring output:\n%s", out)
	}
	if !strings.Contains(out, "weekly report") {
		t.Errorf("expected 'weekly report' in recurring output:\n%s", out)
	}
	if strings.Contains(out, "plain task") {
		t.Errorf("expected non-recurring 'plain task' to be absent from recurring output:\n%s", out)
	}
	if !strings.Contains(out, "↻") {
		t.Errorf("expected recurrence indicator '↻' in output:\n%s", out)
	}
}

func TestRunTodoRecurring_NoTasks_ShowsEmptyMessage(t *testing.T) {
	todoTestEnv(t)

	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	out := captureStdout(t, func() {
		runTodoRecurring(nil, nil)
	})

	if !strings.Contains(out, "No recurring tasks yet") {
		t.Errorf("expected 'No recurring tasks yet' message:\n%s", out)
	}
}

// --- mine proj rm demotion integration tests ---

func TestRunProjRm_DemotesOrphanedTodos(t *testing.T) {
	todoTestEnv(t)

	projDir := registerProject(t, "demoproj")

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ts := todo.NewStore(db.Conn())
	ts.Add("proj task", "", todo.PrioMedium, nil, nil, &projDir, todo.ScheduleLater, todo.RecurrenceWeekly)
	ts.Add("global task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	db.Close()

	projRmYes = true
	defer func() { projRmYes = false }()

	out := captureStdout(t, func() {
		runProjRm(nil, []string{"demoproj"})
	})

	if !strings.Contains(out, "demoted to global") {
		t.Errorf("expected demotion warning in output:\n%s", out)
	}

	// The project task should now be global
	db, err = store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ts = todo.NewStore(db.Conn())
	todos, _ := ts.List(todo.ListOptions{AllProjects: true})
	for _, task := range todos {
		if task.ProjectPath != nil {
			t.Errorf("task %q still has project path after demotion", task.Title)
		}
	}
}
