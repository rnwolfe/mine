package cmd

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

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
	id, _ := ts.Add("done task", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater)
	ts.Complete(id)
	ts.Add("open task", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater)
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
	ts.Add("global task", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater)
	ts.Add("project task", todo.PrioMedium, nil, nil, &projDir, todo.ScheduleLater)
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
	ts.Add("global task", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater)
	ts.Add("flagproj task", todo.PrioMedium, nil, nil, &projDir, todo.ScheduleLater)
	ts.Add("other task", todo.PrioMedium, nil, nil, func() *string { s := "/other/proj"; return &s }(), todo.ScheduleLater)
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
	ts.Add("global task", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater)
	ts.Add("cwd project task", todo.PrioMedium, nil, nil, &projDir, todo.ScheduleLater)
	ts.Add("other project task", todo.PrioMedium, nil, nil, func() *string { s := "/other/proj"; return &s }(), todo.ScheduleLater)
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
	id, _ := ts.Add("test task", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater)
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
	id, _ := ts.Add("alias task", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater)
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
	id, _ := ts.Add("task", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater)
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
	ts.Add("later task", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater)
	ts.Add("someday task", todo.PrioMedium, nil, nil, nil, todo.ScheduleSomeday)
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
	ts.Add("later task", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater)
	ts.Add("someday task", todo.PrioMedium, nil, nil, nil, todo.ScheduleSomeday)
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
	ts.Add("low later task", todo.PrioLow, nil, nil, nil, todo.ScheduleLater)
	ts.Add("crit today task", todo.PrioCrit, nil, nil, nil, todo.ScheduleToday)
	ts.Add("med soon task", todo.PrioMedium, nil, nil, nil, todo.ScheduleSoon)
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
	ts.Add("task A", todo.PrioLow, nil, nil, nil, todo.ScheduleLater)
	ts.Add("task B", todo.PrioCrit, nil, nil, nil, todo.ScheduleToday)
	ts.Add("task C", todo.PrioHigh, nil, nil, nil, todo.ScheduleSoon)
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
	ts.Add("someday idea", todo.PrioCrit, nil, nil, nil, todo.ScheduleSomeday)
	ts.Add("open task", todo.PrioLow, nil, nil, nil, todo.ScheduleLater)
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
	ts.Add("overdue low", todo.PrioLow, nil, &past, nil, todo.ScheduleLater)
	ts.Add("crit today no due", todo.PrioCrit, nil, nil, nil, todo.ScheduleToday)
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
	ts.Add("tagged task", todo.PrioHigh, []string{"docs", "v2"}, &past, nil, todo.ScheduleSoon)
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
