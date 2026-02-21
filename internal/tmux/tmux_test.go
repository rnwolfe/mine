package tmux

import (
	"fmt"
	"testing"
	"time"
)

func TestParseSessions(t *testing.T) {
	raw := "dev\t3\t1700000000\t1\nwork\t1\t1700000100\t0\n"
	sessions := parseSessions(raw)

	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	s := sessions[0]
	if s.Name != "dev" {
		t.Fatalf("expected name 'dev', got %q", s.Name)
	}
	if s.Windows != 3 {
		t.Fatalf("expected 3 windows, got %d", s.Windows)
	}
	if !s.Attached {
		t.Fatal("expected attached=true")
	}
	if s.Created.Unix() != 1700000000 {
		t.Fatalf("expected created=1700000000, got %d", s.Created.Unix())
	}

	s2 := sessions[1]
	if s2.Name != "work" {
		t.Fatalf("expected name 'work', got %q", s2.Name)
	}
	if s2.Windows != 1 {
		t.Fatalf("expected 1 window, got %d", s2.Windows)
	}
	if s2.Attached {
		t.Fatal("expected attached=false")
	}
}

func TestParseSessions_Empty(t *testing.T) {
	sessions := parseSessions("")
	if sessions != nil {
		t.Fatalf("expected nil for empty input, got %v", sessions)
	}
}

func TestParseSessions_Whitespace(t *testing.T) {
	sessions := parseSessions("  \n  \n")
	if sessions != nil {
		t.Fatalf("expected nil for whitespace-only input, got %v", sessions)
	}
}

func TestParseSessions_MalformedLine(t *testing.T) {
	// Lines with fewer than 4 tab-separated fields should be skipped.
	raw := "bad-line\ngood\t2\t1700000000\t0\n"
	sessions := parseSessions(raw)

	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Name != "good" {
		t.Fatalf("expected name 'good', got %q", sessions[0].Name)
	}
}

func TestSessionItem(t *testing.T) {
	s := Session{
		Name:     "myproject",
		Windows:  3,
		Attached: true,
	}

	if s.FilterValue() != "myproject" {
		t.Fatalf("FilterValue should return name, got %q", s.FilterValue())
	}
	if s.Title() != "myproject" {
		t.Fatalf("Title should return name, got %q", s.Title())
	}

	desc := s.Description()
	if desc != "3 windows  (attached)" {
		t.Fatalf("unexpected description: %q", desc)
	}

	// Test singular
	s.Windows = 1
	s.Attached = false
	desc = s.Description()
	if desc != "1 window" {
		t.Fatalf("unexpected description for 1 window: %q", desc)
	}
}

func TestFuzzyFindSession_ExactMatch(t *testing.T) {
	sessions := []Session{
		{Name: "dev"},
		{Name: "work"},
		{Name: "dev-tools"},
	}

	s, err := FuzzyFindSession("work", sessions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "work" {
		t.Fatalf("expected 'work', got %q", s.Name)
	}
}

func TestFuzzyFindSession_CaseInsensitive(t *testing.T) {
	sessions := []Session{{Name: "DevProject"}}

	s, err := FuzzyFindSession("devproject", sessions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "DevProject" {
		t.Fatalf("expected 'DevProject', got %q", s.Name)
	}
}

func TestFuzzyFindSession_PrefixMatch(t *testing.T) {
	sessions := []Session{
		{Name: "development"},
		{Name: "staging"},
	}

	s, err := FuzzyFindSession("dev", sessions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "development" {
		t.Fatalf("expected 'development', got %q", s.Name)
	}
}

func TestFuzzyFindSession_SubstringMatch(t *testing.T) {
	sessions := []Session{
		{Name: "my-dev-env"},
		{Name: "staging"},
	}

	s, err := FuzzyFindSession("dev", sessions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "my-dev-env" {
		t.Fatalf("expected 'my-dev-env', got %q", s.Name)
	}
}

func TestFuzzyFindSession_NoMatch(t *testing.T) {
	sessions := []Session{
		{Name: "dev"},
		{Name: "work"},
	}

	_, err := FuzzyFindSession("production", sessions)
	if err == nil {
		t.Fatal("expected error for no match")
	}
}

func TestFuzzyFindSession_PrefersExact(t *testing.T) {
	sessions := []Session{
		{Name: "dev-tools"},
		{Name: "dev"},
		{Name: "development"},
	}

	s, err := FuzzyFindSession("dev", sessions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "dev" {
		t.Fatalf("should prefer exact match 'dev', got %q", s.Name)
	}
}

func TestListSessions_Stubbed(t *testing.T) {
	// Replace listSessionsFunc with a stub.
	original := listSessionsFunc
	defer func() { listSessionsFunc = original }()

	listSessionsFunc = func() ([]Session, error) {
		return []Session{
			{Name: "test", Windows: 2, Created: time.Now(), Attached: false},
		}, nil
	}

	sessions, err := ListSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 || sessions[0].Name != "test" {
		t.Fatalf("unexpected sessions: %v", sessions)
	}
}

func TestRenameSession_Stubbed(t *testing.T) {
	var capturedArgs []string
	original := tmuxCmd
	defer func() { tmuxCmd = original }()

	tmuxCmd = func(args ...string) (string, error) {
		capturedArgs = args
		return "", nil
	}

	if err := RenameSession("old", "new"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedArgs) != 4 ||
		capturedArgs[0] != "rename-session" ||
		capturedArgs[1] != "-t" ||
		capturedArgs[2] != "old" ||
		capturedArgs[3] != "new" {
		t.Fatalf("unexpected tmux args: %v", capturedArgs)
	}
}

func TestFindSessionByName_Found(t *testing.T) {
	sessions := []Session{
		{Name: "dev"},
		{Name: "work"},
		{Name: "myapp"},
	}

	s := FindSessionByName("work", sessions)
	if s == nil {
		t.Fatal("expected to find session 'work', got nil")
	}
	if s.Name != "work" {
		t.Fatalf("expected 'work', got %q", s.Name)
	}
}

func TestFindSessionByName_NotFound(t *testing.T) {
	sessions := []Session{
		{Name: "dev"},
		{Name: "work"},
	}

	s := FindSessionByName("production", sessions)
	if s != nil {
		t.Fatalf("expected nil for missing session, got %v", s)
	}
}

func TestFindSessionByName_Empty(t *testing.T) {
	s := FindSessionByName("anything", nil)
	if s != nil {
		t.Fatalf("expected nil for empty session list, got %v", s)
	}
}

func TestResolveProjectSession_NamedDir(t *testing.T) {
	original := listSessionsFunc
	defer func() { listSessionsFunc = original }()

	listSessionsFunc = func() ([]Session, error) {
		return nil, nil
	}

	resolvedDir, name, exists, err := ResolveProjectSession("/home/user/code/myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolvedDir != "/home/user/code/myapp" {
		t.Fatalf("expected resolvedDir '/home/user/code/myapp', got %q", resolvedDir)
	}
	if name != "myapp" {
		t.Fatalf("expected session name 'myapp', got %q", name)
	}
	if exists {
		t.Fatal("expected exists=false for empty session list")
	}
}

func TestResolveProjectSession_CwdFallback(t *testing.T) {
	original := listSessionsFunc
	defer func() { listSessionsFunc = original }()

	listSessionsFunc = func() ([]Session, error) {
		return nil, nil
	}

	resolvedDir, name, exists, err := ResolveProjectSession("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Session name should equal the basename of the working directory.
	if resolvedDir == "" {
		t.Fatal("expected non-empty resolvedDir from cwd fallback")
	}
	if name == "" {
		t.Fatal("expected non-empty session name from cwd fallback")
	}
	if exists {
		t.Fatal("expected exists=false for empty session list")
	}
}

func TestResolveProjectSession_AttachIfExists(t *testing.T) {
	original := listSessionsFunc
	defer func() { listSessionsFunc = original }()

	listSessionsFunc = func() ([]Session, error) {
		return []Session{
			{Name: "myapp", Windows: 1},
			{Name: "other", Windows: 2},
		}, nil
	}

	_, name, exists, err := ResolveProjectSession("/code/myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "myapp" {
		t.Fatalf("expected 'myapp', got %q", name)
	}
	if !exists {
		t.Fatal("expected exists=true when session is in the list")
	}
}

func TestResolveProjectSession_NoMatchInList(t *testing.T) {
	original := listSessionsFunc
	defer func() { listSessionsFunc = original }()

	listSessionsFunc = func() ([]Session, error) {
		return []Session{
			{Name: "other-project", Windows: 1},
		}, nil
	}

	_, name, exists, err := ResolveProjectSession("/code/myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "myapp" {
		t.Fatalf("expected 'myapp', got %q", name)
	}
	if exists {
		t.Fatal("expected exists=false when session not in list")
	}
}

func TestRenameSession_EmptyNewName(t *testing.T) {
	err := RenameSession("old", "")
	if err == nil {
		t.Fatal("expected error for empty new name")
	}
}

func TestRenameSession_SessionNotFound(t *testing.T) {
	original := tmuxCmd
	defer func() { tmuxCmd = original }()

	tmuxCmd = func(args ...string) (string, error) {
		return "", fmt.Errorf("exit status 1: can't find session: notexist")
	}

	err := RenameSession("notexist", "newname")
	if err == nil {
		t.Fatal("expected error when session not found")
	}
}
