package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	id, _ := ts.Add("done task", todo.PrioMedium, nil, nil, nil)
	ts.Complete(id)
	ts.Add("open task", todo.PrioMedium, nil, nil, nil)
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
	ts.Add("global task", todo.PrioMedium, nil, nil, nil)
	ts.Add("project task", todo.PrioMedium, nil, nil, &projDir)
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
	ts.Add("global task", todo.PrioMedium, nil, nil, nil)
	ts.Add("flagproj task", todo.PrioMedium, nil, nil, &projDir)
	ts.Add("other task", todo.PrioMedium, nil, nil, func() *string { s := "/other/proj"; return &s }())
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
	ts.Add("global task", todo.PrioMedium, nil, nil, nil)
	ts.Add("cwd project task", todo.PrioMedium, nil, nil, &projDir)
	ts.Add("other project task", todo.PrioMedium, nil, nil, func() *string { s := "/other/proj"; return &s }())
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
